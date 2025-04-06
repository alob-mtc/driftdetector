package orchestrator

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"driftdetector/internal/driftcheck"
	"driftdetector/internal/models"
	awsMocks "driftdetector/internal/providers/aws/mocks"
	"driftdetector/internal/report"
	reportMocks "driftdetector/internal/report/mocks"
	terraformMocks "driftdetector/internal/terraform/mocks"
	"driftdetector/pkg/logging"
	loggerMocks "driftdetector/pkg/logging/mocks"
)

// createMocks is a helper function to create mock instances for testing
// It initializes all the required dependencies with mocks that can be configured
// with expectations for each test case.
func createMocks(t *testing.T) (*awsMocks.InstanceServiceAPI, *terraformMocks.IProvider, *reportMocks.IPrinter, logging.Logger) {
	parserMock := terraformMocks.NewIProvider(t)
	instanceMock := awsMocks.NewInstanceServiceAPI(t)
	reportMock := reportMocks.NewIPrinter(t)
	loggerMock := logging.NewMockLogger()

	return instanceMock, parserMock, reportMock, loggerMock
}

// setupServiceWithMocks creates a new Service instance with the provided configuration and mocks
func setupServiceWithMocks(t *testing.T, config Config) (*Service, *awsMocks.InstanceServiceAPI, *terraformMocks.IProvider, *reportMocks.IPrinter) {
	instanceMock, parserMock, reportMock, loggerMock := createMocks(t)
	service := NewService(config, instanceMock, parserMock, reportMock, loggerMock)
	return service, instanceMock, parserMock, reportMock
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
			service, _, _, _ := setupServiceWithMocks(t, tt.config)

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
			service, _, _, _ := setupServiceWithMocks(t, Config{OutputFormat: tt.formatString})

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
	service, _, _, reportMock := setupServiceWithMocks(t, Config{OutputFormat: "table"})

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
		awsInstance *models.InstanceDetails
		awsError    error
		expectErr   bool
		expectDrift bool
	}{
		{
			name:        "Success case - no drift",
			awsInstance: createTestDriftInstance("i-success", "t2.micro"),
			awsError:    nil,
			expectErr:   false,
			expectDrift: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// Create service and mocks
			service, _, _, reportMock := setupServiceWithMocks(t, Config{})

			// Configure report mock if needed
			if !tc.expectErr {
				reportMock.On("PrintReport", tc.awsInstance.InstanceID, mock.Anything, mock.Anything).Return(nil)
			}

			// Create Terraform config without drift
			tfConfig := &models.InstanceDetails{
				InstanceType: "t2.micro", // same as AWS for no drift
				Tags: map[string]string{
					"Environment": "test",
				},
			}

			// Process the instance
			result := service.processInstance(tc.awsInstance, tfConfig)

			// Verify results
			if tc.expectErr {
				assert.NotNil(t, result.Error, "Should have an error")
			} else {
				assert.Nil(t, result.Error, "Should not have an error")
			}
			assert.Equal(t, tc.expectDrift, result.HasDrift, "Drift detection result should match expectations")
			assert.Equal(t, tc.awsInstance.InstanceID, result.InstanceID, "Instance ID should be preserved")
		})
	}
}

// TestGenerateSummaryReport tests the summary report generation
// to ensure it correctly logs the overview of drift detection results.
func TestGenerateSummaryReport(t *testing.T) {
	// Create a standard error for testing
	expectedErr := errors.New("error")

	// Set up test data with a mix of success, error, and drift results
	results := []DriftDetectionResult{
		{InstanceID: "i-1", HasDrift: true},     // Instance with drift
		{InstanceID: "i-2", Error: expectedErr}, // Instance with error
		{InstanceID: "i-3", HasDrift: false},    // Instance without drift
	}

	// Create service and configure mocks
	parserMock := terraformMocks.NewIProvider(t)
	instanceMock := awsMocks.NewInstanceServiceAPI(t)
	reportMock := reportMocks.NewIPrinter(t)
	loggerMock := loggerMocks.NewLogger(t)
	service := NewService(Config{}, instanceMock, parserMock, reportMock, loggerMock)

	// Configure logger mock with expected calls
	// First, expect an error log for the instance with an error
	loggerMock.On("Error", "Instance %s: Error - %s", "i-2", expectedErr).Return()
	// Then, expect a summary info log with the drift and error statistics
	loggerMock.On("Info", "Summary: Checked %d instances, %d with drift, %d with errors",
		3, 1, 1).Return()

	// Run the function being tested
	service.generateSummaryReport(results)
}

// ================
// Run function tests
// ================

// testCase defines a test case for the Run function
type testCase struct {
	name             string
	config           Config
	mockTFConfig     *models.InstanceDetails
	mockAWSInstances []*models.InstanceDetails
	awsErrors        map[string]error
	expectedAnyDrift bool
	expectedAnyError bool
	expectErr        bool
	tfConfigError    error
}

// createTestRunCase creates a standard test case for the Run function
// which reduces duplicate code for setting up similar cases
func createTestRunCase(name string, instanceIDs []string, hasDrift bool, hasError bool, expectError bool) testCase {
	// Initialize a basic test case with common configurations
	return testCase{
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
		mockAWSInstances: make([]*models.InstanceDetails, 0, len(instanceIDs)),
		awsErrors:        nil,
		expectedAnyDrift: hasDrift,
		expectedAnyError: hasError,
		expectErr:        expectError,
		tfConfigError:    nil,
	}
}

// TestRun tests the main Run function of the orchestrator
// to ensure it correctly coordinates the drift detection workflow.
func TestRun(t *testing.T) {
	// Set up test cases
	tests := []testCase{
		// Create a test case for a successful run with drift
		func() testCase {
			// Start with a standard test case
			tc := createTestRunCase("Successful run - with drift",
				[]string{"i-123", "i-456"}, true, false, false)

			// Customize AWS instances to create drift in one instance
			tc.mockAWSInstances = []*models.InstanceDetails{
				{
					InstanceID:   "i-123",
					InstanceType: "t2.large", // Drift in instance type
					Tags: map[string]string{
						"Environment": "test",
					},
				},
				{
					InstanceID:   "i-456",
					InstanceType: "t2.micro", // No drift
					Tags: map[string]string{
						"Environment": "test",
					},
				},
			}
			return tc
		}(),

		// Create a test case for a successful run without drift
		func() testCase {
			// Start with a standard test case
			tc := createTestRunCase("Successful run - no drift",
				[]string{"i-123"}, false, false, false)

			// Add one AWS instance without drift
			tc.mockAWSInstances = []*models.InstanceDetails{
				{
					InstanceID:   "i-123",
					InstanceType: "t2.micro", // No drift
					Tags: map[string]string{
						"Environment": "test",
					},
				},
			}

			return tc
		}(),

		// Create a test case with AWS error
		func() testCase {
			// Start with a standard test case
			tc := createTestRunCase("AWS error",
				[]string{"i-123", "i-error"}, false, true, true)

			// Add AWS error
			tc.awsErrors = map[string]error{
				"i-error": errors.New("AWS error"),
			}

			return tc
		}(),

		// Create a test case for Terraform configuration error
		func() testCase {
			tc := createTestRunCase("Terraform config error",
				[]string{"i-123"}, false, false, true)

			// Set Terraform config error
			tc.mockTFConfig = nil
			tc.tfConfigError = errors.New("failed to parse config")

			return tc
		}(),

		// Create a test case for invalid configuration
		func() testCase {
			tc := createTestRunCase("Invalid config - no instances",
				[]string{}, false, false, true)

			return tc
		}(),

		// Create a test case with concurrency limit
		func() testCase {
			// Start with a standard test case
			tc := createTestRunCase("With concurrency limit",
				[]string{"i-123", "i-456"}, false, false, false)

			// Set concurrency limit
			tc.config.ConcurrencyLimit = 1

			// Add AWS instances that match Terraform config (no drift)
			tc.mockAWSInstances = []*models.InstanceDetails{
				{
					InstanceID:   "i-123",
					InstanceType: "t2.micro",
					Tags: map[string]string{
						"Environment": "test",
					},
				},
				{
					InstanceID:   "i-456",
					InstanceType: "t2.micro",
					Tags: map[string]string{
						"Environment": "test",
					},
				},
			}

			return tc
		}(),
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create service and configure mocks
			service, instanceMock, parserMock, reportMock := setupServiceWithMocks(t, tt.config)

			// Configure Terraform parser mock if instance IDs are provided
			if len(tt.config.InstanceIDs) != 0 {
				parserMock.On("ParseHCLConfig", tt.config.ConfigPath).Return(tt.mockTFConfig, tt.tfConfigError)
			}

			// Configure AWS mock for each instance
			if len(tt.mockAWSInstances) > 0 {
				instanceMock.On("GetInstancesDetails", mock.Anything, tt.config.InstanceIDs).Return(tt.mockAWSInstances, nil)
			}

			// Configure AWS error mock if needed
			if tt.awsErrors != nil {
				// Return error for the error case
				errorInstances := make([]*models.InstanceDetails, 0)
				instanceMock.On("GetInstancesDetails", mock.Anything, tt.config.InstanceIDs).Return(errorInstances, tt.awsErrors["i-error"])
			}

			// Configure report mock if not expecting a configuration error
			if !tt.expectErr {
				reportMock.On("PrintReport", mock.Anything, mock.Anything, mock.Anything).Return(nil)
			}

			// Run the function being tested
			anyDrift, anyError, err := service.Run(context.Background())

			// Verify results
			if tt.expectErr {
				assert.Error(t, err, "Expected an error")
				return // Don't check drift/error flags if we expected an error
			} else {
				assert.NoError(t, err, "Did not expect an error")
			}

			assert.Equal(t, tt.expectedAnyDrift, anyDrift, "Drift detection result should match expectations")
			assert.Equal(t, tt.expectedAnyError, anyError, "Error status should match expectations")
		})
	}
}
