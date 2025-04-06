package orchestrator

import (
	"context"
	"fmt"
	"strings"

	"golang.org/x/sync/errgroup"

	"driftdetector/internal/driftcheck"
	"driftdetector/internal/models"
	"driftdetector/internal/providers/aws"
	"driftdetector/internal/report"
	"driftdetector/internal/terraform"
	"driftdetector/pkg/logging"
)

// Service orchestrates the drift detection process.
// It coordinates the AWS and Terraform providers, manages concurrent processing
// of instances, and generates reports on the detected drift.
type Service struct {
	config          Config
	awsSrv          aws.InstanceServiceAPI
	terraformParser terraform.IProvider
	reportPrinter   report.IPrinter
	logger          logging.Logger
}

// NewService creates a new orchestrator service with the given configuration.
func NewService(
	config Config,
	awsSrv aws.InstanceServiceAPI,
	terraformParser terraform.IProvider,
	reportPrinter report.IPrinter,
	logger logging.Logger,
) *Service {
	// If logger is nil, use a default logger
	if logger == nil {
		logger = logging.NewDefaultLogger()
	}

	return &Service{
		config:          config,
		awsSrv:          awsSrv,
		terraformParser: terraformParser,
		reportPrinter:   reportPrinter,
		logger:          logger,
	}
}

// NewDefaultService creates a new service with default implementations of dependencies
func NewDefaultService(config Config) (*Service, error) {
	// Create AWS instance service with default configuration
	awsService, err := aws.NewInstanceServiceWithDefaultConfig(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to initialize AWS service: %w", err)
	}

	logger := logging.NewDefaultLogger()

	return NewService(
		config,
		awsService,
		terraform.DefaultParser{},
		report.DefaultPrinter{},
		logger,
	), nil
}

// Run executes the drift detection workflow for all instances
func (s *Service) Run(ctx context.Context) (bool, bool, error) {
	// Validate configuration
	if err := s.validateConfig(); err != nil {
		return false, true, err
	}

	// Parse Terraform configuration (only once, shared across all instances)
	tfConfig, err := s.parseTerrformConfig()
	if err != nil {
		return false, true, err
	}

	// Process all instances concurrently and collect results
	results, err := s.processAllInstances(ctx, tfConfig)
	if err != nil {
		return s.anyDriftDetected(results), true, err
	}

	// Generate summary report
	s.generateSummaryReport(results)

	return s.anyDriftDetected(results), s.anyErrorsOccurred(results), nil
}

// parseTerrformConfig parses the HCL configuration file at the specified path.
// This is done once for all instances to avoid repeated parsing.
func (s *Service) parseTerrformConfig() (*models.InstanceDetails, error) {
	tfConfig, err := s.terraformParser.ParseHCLConfig(s.config.ConfigPath)
	if err != nil {
		return nil, fmt.Errorf("error parsing Terraform configuration: %w", err)
	}
	return tfConfig, nil
}

// processAllInstances handles the concurrent processing of all instances and result collection.
// It returns the results and any error that occurred during processing.
func (s *Service) processAllInstances(ctx context.Context, tfConfig *models.InstanceDetails) ([]DriftDetectionResult, error) {
	// Create a new error group for concurrent processing
	g, gctx := errgroup.WithContext(ctx)

	// Set the concurrency limit if specified to avoid overwhelming the AWS API
	if s.config.ConcurrencyLimit > 0 {
		g.SetLimit(s.config.ConcurrencyLimit)
	}

	// Channel to collect results from individual goroutines
	resultChan := make(chan DriftDetectionResult, len(s.config.InstanceIDs))

	// Start a goroutine for each instance using the error group
	for _, instanceID := range s.config.InstanceIDs {
		// Add the task to the error group
		g.Go(func() error {
			// Process this instance
			result := s.processInstance(gctx, instanceID, tfConfig)

			// Send the result through the channel
			select {
			case resultChan <- result:
				// Result sent successfully
				return nil
			case <-gctx.Done():
				// Context was cancelled, typically due to timeout or cancellation
				return gctx.Err()
			}
		})
	}

	// Wait for all tasks to complete in a separate goroutine
	go func() {
		_ = g.Wait() // Ignore any errors for now, we'll check them after collecting results
		close(resultChan)
	}()

	// Collect results from the channel
	results := s.collectResults(resultChan)

	// Check if there were any errors in the error group
	if err := g.Wait(); err != nil {
		return results, fmt.Errorf("error in concurrent drift detection: %w", err)
	}

	return results, nil
}

// collectResults gathers results from the result channel.
func (s *Service) collectResults(resultChan <-chan DriftDetectionResult) []DriftDetectionResult {
	results := make([]DriftDetectionResult, 0, len(s.config.InstanceIDs))

	for result := range resultChan {
		results = append(results, result)
	}

	return results
}

// anyDriftDetected returns true if any instance has drift.
func (s *Service) anyDriftDetected(results []DriftDetectionResult) bool {
	// Loop through all results to find any instance with drift
	for _, result := range results {
		// Only count instances where HasDrift is true and there was no error
		if result.HasDrift && result.Error == nil {
			return true
		}
	}
	return false
}

// anyErrorsOccurred returns true if any instance processing resulted in an error.
func (s *Service) anyErrorsOccurred(results []DriftDetectionResult) bool {
	return countErrors(results) > 0
}

// processInstance handles drift detection for a single instance.
func (s *Service) processInstance(ctx context.Context, instanceID string, tfConfig *models.InstanceDetails) DriftDetectionResult {
	result := DriftDetectionResult{
		InstanceID: instanceID,
	}

	// Fetch AWS instance details
	awsInstance, err := s.fetchAWSInstanceDetails(ctx, instanceID)
	if err != nil {
		result.Error = err
		return result
	}

	// Detect drift between AWS and Terraform configurations
	driftResult, err := s.detectInstanceDrift(awsInstance, tfConfig)
	if err != nil {
		result.Error = err
		return result
	}

	result.HasDrift = driftResult.HasDrift
	result.Result = driftResult

	// Generate individual report for this instance
	if err := s.generateInstanceReport(instanceID, driftResult); err != nil {
		result.Error = fmt.Errorf("error generating report: %w", err)
	}

	return result
}

// fetchAWSInstanceDetails retrieves the current state of an instance from AWS.
func (s *Service) fetchAWSInstanceDetails(ctx context.Context, instanceID string) (*models.InstanceDetails, error) {
	awsInstance, err := s.awsSrv.GetInstanceDetails(ctx, instanceID)
	if err != nil {
		return nil, fmt.Errorf("error fetching AWS instance details: %w", err)
	}
	return awsInstance, nil
}

// detectInstanceDrift checks for differences between the actual AWS instance state
// and the desired state defined in Terraform.
func (s *Service) detectInstanceDrift(awsInstance, tfConfig *models.InstanceDetails) (*driftcheck.DriftResult, error) {
	driftResult, err := driftcheck.DetectDrift(awsInstance, tfConfig, s.config.AttributesToCheck)
	if err != nil {
		return nil, fmt.Errorf("error detecting drift: %w", err)
	}
	return driftResult, nil
}

// getOutputFormat converts the string format to report.OutputFormatType.
func (s *Service) getOutputFormat() report.OutputFormatType {
	switch strings.ToUpper(s.config.OutputFormat) {
	case "JSON":
		return report.OutputFormatTypeJSON
	default:
		// Default to table format for better human readability
		return report.OutputFormatTypeTABLE
	}
}

// validateConfig checks if the required configuration is provided.
func (s *Service) validateConfig() error {
	if len(s.config.InstanceIDs) == 0 {
		return fmt.Errorf("at least one instance ID is required")
	}
	if s.config.ConfigPath == "" {
		return fmt.Errorf("terraform configuration path is required")
	}
	return nil
}

// generateInstanceReport generates and prints the drift detection report for a single instance.
func (s *Service) generateInstanceReport(instanceID string, driftResult *driftcheck.DriftResult) error {
	// Convert driftResult to []driftcheck.Drift for reporting
	drifts := driftcheck.ConvertToDrifts(driftResult)

	// Determine the output format from the configuration
	format := s.getOutputFormat()

	// Generate and print the report using the configured printer
	return s.reportPrinter.PrintReport(instanceID, drifts, format)
}

// generateSummaryReport generates a summary report for all instances.
// This gives an overview of the drift detection results across all instances,
// which is particularly useful when checking multiple instances at once.
func (s *Service) generateSummaryReport(results []DriftDetectionResult) {
	// Count and log instances with errors
	errCount := countErrors(results)
	if errCount > 0 {
		for _, r := range results {
			if r.Error != nil {
				// Log each error with the associated instance ID for easier troubleshooting
				s.logger.Error("Instance %s: Error - %s", r.InstanceID, r.Error)
			}
		}
	}

	// Only generate a summary if more than one instance was checked
	// For a single instance, the detailed report is sufficient
	if len(results) > 1 {
		s.logger.Info("\nSummary: Checked %d instances, %d with drift, %d with errors",
			len(results),
			countDrifts(results),
			errCount,
		)
	}
}

// countDrifts counts the number of instances with drift.
func countDrifts(results []DriftDetectionResult) int {
	count := 0
	for _, r := range results {
		if r.HasDrift {
			count++
		}
	}
	return count
}

// countErrors counts the number of instances with errors.
func countErrors(results []DriftDetectionResult) int {
	count := 0
	for _, r := range results {
		if r.Error != nil {
			count++
		}
	}
	return count
}
