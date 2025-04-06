package orchestrator

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"driftdetector/internal/driftcheck"
	"driftdetector/internal/models"
	awsMocks "driftdetector/internal/providers/aws/mocks"
	"driftdetector/internal/report"
	reportMocks "driftdetector/internal/report/mocks"
	terraformMocks "driftdetector/internal/terraform/mocks"
	loggerMocks "driftdetector/pkg/logging/mocks"
)

// createMocks is a helper function to create mock instances for testing
// It initializes all the required dependencies with mocks that can be configured
// with expectations for each test case.
func createMocks(t *testing.T) (*awsMocks.InstanceServiceAPI, *terraformMocks.IProvider, *reportMocks.IPrinter, *loggerMocks.Logger) {
	parserMock := terraformMocks.NewIProvider(t)
	instanceMock := awsMocks.NewInstanceServiceAPI(t)
	reportMock := reportMocks.NewIPrinter(t)
	loggerMock := loggerMocks.NewLogger(t)

	return instanceMock, parserMock, reportMock, loggerMock
}

// setupServiceWithMocks creates a new Service instance with the provided configuration and mocks
func setupServiceWithMocks(t *testing.T, config Config) (*Service, *awsMocks.InstanceServiceAPI, *terraformMocks.IProvider, *reportMocks.IPrinter, *loggerMocks.Logger) {
	instanceMock, parserMock, reportMock, loggerMock := createMocks(t)
	service := NewService(config, instanceMock, parserMock, reportMock, loggerMock)
	return service, instanceMock, parserMock, reportMock, loggerMock
}

// TestValidateConfig tests the configuration validation logic
// It ensures that the service properly validates required fields and returns
// appropriate errors for invalid configurations.
func TestValidateConfig(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr bool
	}{
		{
			name: "Valid config with single instance",
			config: Config{
				InstanceIDs: []string{"i-12345"},
				ConfigPath:  "/path/to/config.tf",
			},
			wantErr: false,
		},
		{
			name: "Valid config with multiple instances",
			config: Config{
				InstanceIDs: []string{"i-12345", "i-67890"},
				ConfigPath:  "/path/to/config.tf",
			},
			wantErr: false,
		},
		{
			name: "Missing instance IDs",
			config: Config{
				ConfigPath: "/path/to/config.tf",
			},
			wantErr: true,
		},
		{
			name: "Missing config path",
			config: Config{
				InstanceIDs: []string{"i-12345"},
			},
			wantErr: true,
		},
		{
			name:    "Empty config",
			config:  Config{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Use the helper function to create service with mocks
			service, _, _, _, _ := setupServiceWithMocks(t, tt.config)

			// Test the validation function
			err := service.validateConfig()

			// Assert the expected outcome
			if tt.wantErr {
				assert.Error(t, err, "Expected an error for invalid config")
			} else {
				assert.NoError(t, err, "Expected no error for valid config")
			}
		})
	}
}

// TestCountDrifts tests the countDrifts function to ensure it correctly
// counts instances with drift.
func TestCountDrifts(t *testing.T) {
	// Create test data with a mix of drifted and non-drifted instances
	results := []DriftDetectionResult{
		{HasDrift: true},  // Should be counted
		{HasDrift: false}, // Should not be counted
		{HasDrift: true},  // Should be counted
		{Error: errors.New("some error"), HasDrift: false}, // Should not be counted even with error
	}

	count := countDrifts(results)
	assert.Equal(t, 2, count, "Should count exactly 2 instances with drift")
}

// TestCountErrors tests the countErrors function to ensure it correctly
// counts instances with errors.
func TestCountErrors(t *testing.T) {
	// Create test data with a mix of errors and successful results
	results := []DriftDetectionResult{
		{HasDrift: true},               // No error
		{Error: errors.New("error 1")}, // Has error
		{HasDrift: true},               // No error
		{Error: errors.New("error 2")}, // Has error
	}

	count := countErrors(results)
	assert.Equal(t, 2, count, "Should count exactly 2 instances with errors")
}

// TestGetOutputFormat tests the getOutputFormat function to ensure it
// correctly converts string format specifications to the appropriate enum value.
func TestGetOutputFormat(t *testing.T) {
	tests := []struct {
		name         string
		formatString string
		expected     report.OutputFormatType
	}{
		{
			name:         "JSON format",
			formatString: "json",
			expected:     report.OutputFormatTypeJSON,
		},
		{
			name:         "JSON uppercase",
			formatString: "JSON",
			expected:     report.OutputFormatTypeJSON,
		},
		{
			name:         "Default to table when unrecognized",
			formatString: "unknown",
			expected:     report.OutputFormatTypeTABLE,
		},
		{
			name:         "Table format",
			formatString: "table",
			expected:     report.OutputFormatTypeTABLE,
		},
		{
			name:         "Empty string defaults to table",
			formatString: "",
			expected:     report.OutputFormatTypeTABLE,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create service with the test format string
			service, _, _, _, _ := setupServiceWithMocks(t, Config{OutputFormat: tt.formatString})

			// Get the format and check it matches expected value
			format := service.getOutputFormat()
			assert.Equal(t, tt.expected, format, "Format conversion should match expected type")
		})
	}
}

// TestGenerateInstanceReport tests the report generation for a single instance
// to ensure it correctly calls the report printer with the right parameters.
func TestGenerateInstanceReport(t *testing.T) {
	// Create service and mocks
	service, _, _, reportMock, _ := setupServiceWithMocks(t, Config{OutputFormat: "table"})

	// Set up test data
	instanceID := "i-12345"
	driftResult := &driftcheck.DriftResult{
		HasDrift: true,
		Drifts: map[string]models.DriftDetail{
			"instance_type": {
				Attribute:      "instance_type",
				AWSValue:       "t2.micro",
				TerraformValue: "t2.small",
			},
		},
	}

	// Configure mock expectations
	reportMock.On("PrintReport", instanceID, mock.Anything, report.OutputFormatTypeTABLE).Return(nil)

	// Test the report generation
	err := service.generateInstanceReport(instanceID, driftResult)

	// Verify expectations
	assert.NoError(t, err, "Report generation should not fail")
	reportMock.AssertExpectations(t)
}

// createTestDriftInstance creates a standard instance details object for testing
// with the specified ID and instance type, and default tags
func createTestDriftInstance(instanceID string, instanceType string) *models.InstanceDetails {
	return &models.InstanceDetails{
		InstanceID:   instanceID,
		InstanceType: instanceType,
		Tags: map[string]string{
			"Environment": "test",
		},
	}
}

// TestProcessInstance tests the processing of a single instance
// to ensure it correctly detects drift and handles errors.
func TestProcessInstance(t *testing.T) {
	cases := []struct {
		name        string
		instanceID  string
		awsInstance *models.InstanceDetails
		awsError    error
		expectErr   bool
		expectDrift bool
	}{
		{
			name:        "Success case - no drift",
			instanceID:  "i-success",
			awsInstance: createTestDriftInstance("i-success", "t2.micro"),
			awsError:    nil,
			expectErr:   false,
			expectDrift: false,
		},
		{
			name:        "AWS error case",
			instanceID:  "i-error",
			awsInstance: nil,
			awsError:    errors.New("AWS error"),
			expectErr:   true,
			expectDrift: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// Create service and mocks
			service, instanceMock, _, reportMock, _ := setupServiceWithMocks(t, Config{})

			// Configure AWS mock
			instanceMock.On("GetInstanceDetails", mock.Anything, tc.instanceID).Return(tc.awsInstance, tc.awsError)

			// Configure report mock if needed
			if !tc.expectErr {
				reportMock.On("PrintReport", tc.instanceID, mock.Anything, mock.Anything).Return(nil)
			}

			// Create Terraform config without drift
			tfConfig := &models.InstanceDetails{
				InstanceType: "t2.micro", // same as AWS for no drift
				Tags: map[string]string{
					"Environment": "test",
				},
			}

			// Process the instance
			result := service.processInstance(context.Background(), tc.instanceID, tfConfig)

			// Verify results
			if tc.expectErr {
				assert.NotNil(t, result.Error, "Should have an error")
			} else {
				assert.Nil(t, result.Error, "Should not have an error")
			}
			assert.Equal(t, tc.expectDrift, result.HasDrift, "Drift detection result should match expectations")
			assert.Equal(t, tc.instanceID, result.InstanceID, "Instance ID should be preserved")
		})
	}
}

// TestGenerateSummaryReport tests the summary report generation
// to ensure it correctly logs the overview of drift detection results.
func TestGenerateSummaryReport(t *testing.T) {
	// Capture stdout to test output (although we're now using the logger)
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Create a standard error for testing
	expectedErr := errors.New("error")

	// Set up test data with a mix of success, error, and drift results
	results := []DriftDetectionResult{
		{InstanceID: "i-1", HasDrift: true},     // Instance with drift
		{InstanceID: "i-2", Error: expectedErr}, // Instance with error
		{InstanceID: "i-3", HasDrift: false},    // Instance without drift
	}

	// Create service and configure mocks
	service, _, _, _, loggerMock := setupServiceWithMocks(t, Config{})

	// Configure logger mock with expected calls
	// First, expect an error log for the instance with an error
	loggerMock.On("Error", "Instance %s: Error - %s", "i-2", expectedErr).Return()
	// Then, expect a summary info log with the drift and error statistics
	loggerMock.On("Info", "\nSummary: Checked %d instances, %d with drift, %d with errors",
		3, 1, 1).Return()

	// Run the function being tested
	service.generateSummaryReport(results)

	// Close writer to flush the buffer
	w.Close()

	// Read the captured output (no longer needed but kept for backward compatibility)
	var buf bytes.Buffer
	io.Copy(&buf, r)

	// Restore stdout
	os.Stdout = old

	// Verify logger mock was called with the expected parameters
	loggerMock.AssertExpectations(t)
}

// createTestRunCase creates a standard test case for the Run function
// which reduces duplicate code for setting up similar cases
func createTestRunCase(name string, instanceIDs []string, hasDrift bool, hasError bool, expectError bool) struct {
	name             string
	config           Config
	mockTFConfig     *models.InstanceDetails
	mockAWSInstances map[string]*models.InstanceDetails
	awsErrors        map[string]error
	expectedAnyDrift bool
	expectedAnyError bool
	expectErr        bool
	tfConfigError    error
} {
	// Initialize a basic test case with common configurations
	testCase := struct {
		name             string
		config           Config
		mockTFConfig     *models.InstanceDetails
		mockAWSInstances map[string]*models.InstanceDetails
		awsErrors        map[string]error
		expectedAnyDrift bool
		expectedAnyError bool
		expectErr        bool
		tfConfigError    error
	}{
		name: name,
		config: Config{
			InstanceIDs: instanceIDs,
			ConfigPath:  "test.tf",
		},
		mockTFConfig: &models.InstanceDetails{
			InstanceType: "t2.micro",
			Tags: map[string]string{
				"Environment": "test",
			},
		},
		mockAWSInstances: make(map[string]*models.InstanceDetails),
		awsErrors:        nil,
		expectedAnyDrift: hasDrift,
		expectedAnyError: hasError,
		expectErr:        expectError,
		tfConfigError:    nil,
	}

	// Return the initialized test case
	return testCase
}

// TestRun tests the main Run function of the orchestrator
// to ensure it correctly coordinates the drift detection workflow.
func TestRun(t *testing.T) {
	// Set up test cases
	tests := []struct {
		name             string
		config           Config
		mockTFConfig     *models.InstanceDetails
		mockAWSInstances map[string]*models.InstanceDetails
		awsErrors        map[string]error
		expectedAnyDrift bool
		expectedAnyError bool
		expectErr        bool
		tfConfigError    error
	}{
		// Create a test case for a successful run with drift
		func() struct {
			name             string
			config           Config
			mockTFConfig     *models.InstanceDetails
			mockAWSInstances map[string]*models.InstanceDetails
			awsErrors        map[string]error
			expectedAnyDrift bool
			expectedAnyError bool
			expectErr        bool
			tfConfigError    error
		} {
			// Start with a standard test case
			tc := createTestRunCase("Successful run - with drift",
				[]string{"i-123", "i-456"}, true, false, false)

			// Customize AWS instances to create drift in one instance
			tc.mockAWSInstances["i-123"] = &models.InstanceDetails{
				InstanceID:   "i-123",
				InstanceType: "t2.large", // Drift in instance type
				Tags: map[string]string{
					"Environment": "test",
				},
			}
			tc.mockAWSInstances["i-456"] = &models.InstanceDetails{
				InstanceID:   "i-456",
				InstanceType: "t2.micro", // No drift
				Tags: map[string]string{
					"Environment": "test",
				},
			}

			return tc
		}(),

		// Create a test case for a successful run without drift
		func() struct {
			name             string
			config           Config
			mockTFConfig     *models.InstanceDetails
			mockAWSInstances map[string]*models.InstanceDetails
			awsErrors        map[string]error
			expectedAnyDrift bool
			expectedAnyError bool
			expectErr        bool
			tfConfigError    error
		} {
			// Start with a standard test case
			tc := createTestRunCase("Successful run - no drift",
				[]string{"i-123"}, false, false, false)

			// Add one AWS instance without drift
			tc.mockAWSInstances["i-123"] = &models.InstanceDetails{
				InstanceID:   "i-123",
				InstanceType: "t2.micro", // No drift
				Tags: map[string]string{
					"Environment": "test",
				},
			}

			return tc
		}(),

		// Create a test case with AWS error
		func() struct {
			name             string
			config           Config
			mockTFConfig     *models.InstanceDetails
			mockAWSInstances map[string]*models.InstanceDetails
			awsErrors        map[string]error
			expectedAnyDrift bool
			expectedAnyError bool
			expectErr        bool
			tfConfigError    error
		} {
			// Start with a standard test case
			tc := createTestRunCase("AWS error",
				[]string{"i-123", "i-error"}, false, true, false)

			// Add one successful AWS instance without drift
			tc.mockAWSInstances["i-123"] = &models.InstanceDetails{
				InstanceID:   "i-123",
				InstanceType: "t2.micro",
				Tags: map[string]string{
					"Environment": "test",
				},
			}

			// Add one AWS instance with error
			tc.awsErrors = map[string]error{
				"i-error": errors.New("AWS error"),
			}

			return tc
		}(),

		// Create a test case for Terraform configuration error
		func() struct {
			name             string
			config           Config
			mockTFConfig     *models.InstanceDetails
			mockAWSInstances map[string]*models.InstanceDetails
			awsErrors        map[string]error
			expectedAnyDrift bool
			expectedAnyError bool
			expectErr        bool
			tfConfigError    error
		} {
			tc := createTestRunCase("Terraform config error",
				[]string{"i-123"}, false, false, true)

			// Set Terraform config error
			tc.mockTFConfig = nil
			tc.tfConfigError = errors.New("failed to parse config")

			return tc
		}(),

		// Create a test case for invalid configuration
		func() struct {
			name             string
			config           Config
			mockTFConfig     *models.InstanceDetails
			mockAWSInstances map[string]*models.InstanceDetails
			awsErrors        map[string]error
			expectedAnyDrift bool
			expectedAnyError bool
			expectErr        bool
			tfConfigError    error
		} {
			tc := createTestRunCase("Invalid config - no instances",
				[]string{}, false, false, true)

			return tc
		}(),

		// Create a test case with concurrency limit
		func() struct {
			name             string
			config           Config
			mockTFConfig     *models.InstanceDetails
			mockAWSInstances map[string]*models.InstanceDetails
			awsErrors        map[string]error
			expectedAnyDrift bool
			expectedAnyError bool
			expectErr        bool
			tfConfigError    error
		} {
			// Start with a standard test case
			tc := createTestRunCase("With concurrency limit",
				[]string{"i-123", "i-456"}, false, false, false)

			// Set concurrency limit
			tc.config.ConcurrencyLimit = 1

			// Add AWS instances that match Terraform config (no drift)
			tc.mockAWSInstances["i-123"] = &models.InstanceDetails{
				InstanceID:   "i-123",
				InstanceType: "t2.micro",
				Tags: map[string]string{
					"Environment": "test",
				},
			}
			tc.mockAWSInstances["i-456"] = &models.InstanceDetails{
				InstanceID:   "i-456",
				InstanceType: "t2.micro",
				Tags: map[string]string{
					"Environment": "test",
				},
			}

			return tc
		}(),
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create service and configure mocks
			service, instanceMock, parserMock, reportMock, loggerMock := setupServiceWithMocks(t, tt.config)

			// Configure Terraform parser mock if instance IDs are provided
			if len(tt.config.InstanceIDs) != 0 {
				parserMock.On("ParseHCLConfig", tt.config.ConfigPath).Return(tt.mockTFConfig, tt.tfConfigError)
			}

			// Configure AWS mock for each instance
			for instanceID, instance := range tt.mockAWSInstances {
				instanceMock.On("GetInstanceDetails", mock.Anything, instanceID).Return(instance, nil)
			}

			// Configure AWS error & logger mocks if needed
			if tt.awsErrors != nil {
				for instanceID, err := range tt.awsErrors {
					instanceMock.On("GetInstanceDetails", mock.Anything, instanceID).Return(nil, err)
					loggerMock.On("Error", "Instance %s: Error - %s", instanceID, mock.Anything).Return()
				}
			}

			// Configure report mock if not expecting a configuration error
			if !tt.expectErr {
				reportMock.On("PrintReport", mock.Anything, mock.Anything, mock.Anything).Return(nil)
			}

			// Configure summary logger if multiple instances
			if len(tt.config.InstanceIDs) > 1 {
				loggerMock.On(
					"Info",
					"\nSummary: Checked %d instances, %d with drift, %d with errors",
					len(tt.config.InstanceIDs),
					mock.Anything,
					mock.Anything,
				).Return()
			}

			// Run the orchestrator
			anyDrift, anyError, err := service.Run(context.Background())

			// Verify results
			if tt.expectErr {
				assert.Error(t, err, "Should return an error for invalid configurations")
				return
			} else {
				assert.NoError(t, err, "Should not return an error for valid configurations")
			}

			assert.Equal(t, tt.expectedAnyDrift, anyDrift, "anyDrift flag should match expected value")
			assert.Equal(t, tt.expectedAnyError, anyError, "anyError flag should match expected value")
		})
	}
}
