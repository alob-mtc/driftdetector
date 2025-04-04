package orchestrator

import (
	"context"
	"fmt"
	"strings"

	"golang.org/x/sync/errgroup"

	"driftdetector/internal/driftcheck"
	"driftdetector/internal/driftcheck/report"
	"driftdetector/internal/models"
	aws "driftdetector/internal/providers/aws"
	"driftdetector/internal/terraform"
)

// Service orchestrates the drift detection process.
type Service struct {
	config          Config
	awsSrv          aws.InstanceServiceAPI
	terraformParser terraform.IProvider
	reportPrinter   report.IPrinter
}

// NewService creates a new orchestrator service with the given configuration.
func NewService(
	config Config,
	awsSrv aws.InstanceServiceAPI,
	terraformParser terraform.IProvider,
	reportPrinter report.IPrinter,
) *Service {
	return &Service{
		config:          config,
		awsSrv:          awsSrv,
		terraformParser: terraformParser,
		reportPrinter:   reportPrinter,
	}
}

// NewDefaultService creates a new service with default implementations of dependencies
func NewDefaultService(config Config) (*Service, error) {
	// Create AWS instance service
	awsService, err := aws.NewInstanceServiceWithDefaultConfig(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to initialize AWS service: %w", err)
	}

	return NewService(config, awsService, terraform.DefaultParser{}, report.DefaultPrinter{}), nil
}

// Run executes the drift detection workflow for all instances
func (s *Service) Run(ctx context.Context) (bool, bool, error) {
	// Validate configuration
	if err := s.validateConfig(); err != nil {
		return false, true, err
	}

	// Parse Terraform configuration (only once, shared across all instances)
	tfConfig, err := s.terraformParser.ParseHCLConfig(s.config.ConfigPath)
	if err != nil {
		return false, true, fmt.Errorf("error parsing Terraform configuration: %w", err)
	}

	g, gctx := errgroup.WithContext(ctx)
	// Set the concurrency limit if specified
	if s.config.ConcurrencyLimit > 0 {
		g.SetLimit(s.config.ConcurrencyLimit)
	}

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
				// Context was cancelled
				return gctx.Err()
			}
		})
	}

	// Wait for all tasks to complete in a separate goroutine
	go func() {
		_ = g.Wait() // Ignore any errors for now, we'll check them after collecting results
		close(resultChan)
	}()

	// Collect and process results
	var anyDrift, anyError bool
	results := make([]DriftDetectionResult, 0, len(s.config.InstanceIDs))

	for result := range resultChan {
		results = append(results, result)
		if result.HasDrift {
			anyDrift = true
		}
		if result.Error != nil {
			anyError = true
		}
	}

	// Check if there were any errors in the error group
	if err := g.Wait(); err != nil {
		return anyDrift, true, fmt.Errorf("error in concurrent drift detection: %w", err)
	}

	// Generate summary report
	s.generateSummaryReport(results)

	return anyDrift, anyError, nil
}

// processInstance handles drift detection for a single instance.
func (s *Service) processInstance(ctx context.Context, instanceID string, tfConfig *models.InstanceDetails) DriftDetectionResult {
	result := DriftDetectionResult{
		InstanceID: instanceID,
	}

	// Fetch AWS instance details
	awsInstance, err := s.awsSrv.GetInstanceDetails(ctx, instanceID)
	if err != nil {
		result.Error = fmt.Errorf("error fetching AWS instance details: %w", err)
		return result
	}

	// Detect drift
	driftResult, err := driftcheck.DetectDrift(awsInstance, tfConfig, s.config.AttributesToCheck)
	if err != nil {
		result.Error = fmt.Errorf("error detecting drift: %w", err)
		return result
	}

	result.HasDrift = driftResult.HasDrift
	result.Result = driftResult

	// Generate individual report
	if err := s.generateInstanceReport(instanceID, driftResult); err != nil {
		result.Error = fmt.Errorf("error generating report: %w", err)
	}

	return result
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

	// Determine the output format
	format := s.getOutputFormat()

	// Generate and print the report
	return s.reportPrinter.PrintReport(instanceID, drifts, format)
}

// generateSummaryReport generates a summary report for all instances.
func (s *Service) generateSummaryReport(results []DriftDetectionResult) {
	errCount := countErrors(results)
	if errCount > 0 {
		for _, r := range results {
			if r.Error != nil {
				fmt.Printf("Instance %s: Error - %s\n", r.InstanceID, r.Error)
			}
		}
	}

	if len(results) > 1 {
		fmt.Printf("\n\nSummary: Checked %d instances, %d with drift, %d with errors\n",
			len(results),
			countDrifts(results),
			errCount,
		)
	}
}

// getOutputFormat converts the string format to report.OutputFormatType.
func (s *Service) getOutputFormat() report.OutputFormatType {
	switch strings.ToUpper(s.config.OutputFormat) {
	case "JSON":
		return report.OutputFormatTypeJSON
	default:
		return report.OutputFormatTypeTABLE
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
