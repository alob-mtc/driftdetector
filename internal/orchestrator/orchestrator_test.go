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
	"driftdetector/internal/driftcheck/report"
	reportMocks "driftdetector/internal/driftcheck/report/mocks"
	"driftdetector/internal/models"
	awsMocks "driftdetector/internal/providers/aws/mocks"
	terraformMocks "driftdetector/internal/terraform/mocks"
	loggerMocks "driftdetector/pkg/logging/mocks"
)

// createMocks is a helper function to create mock instances for testing
func createMocks(t *testing.T) (*awsMocks.InstanceServiceAPI, *terraformMocks.IProvider, *reportMocks.IPrinter, *loggerMocks.Logger) {
	parserMock := terraformMocks.NewIProvider(t)
	instanceMock := awsMocks.NewInstanceServiceAPI(t)
	reportMock := reportMocks.NewIPrinter(t)
	loggerMock := loggerMocks.NewLogger(t)

	return instanceMock, parserMock, reportMock, loggerMock
}

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
			instanceMock, parserMock, reportMock, loggerMock := createMocks(t)

			service := NewService(tt.config, instanceMock, parserMock, reportMock, loggerMock)
			err := service.validateConfig()

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestCountDrifts(t *testing.T) {
	results := []DriftDetectionResult{
		{HasDrift: true},
		{HasDrift: false},
		{HasDrift: true},
		{Error: errors.New("some error")},
	}

	count := countDrifts(results)
	assert.Equal(t, 2, count)
}

func TestCountErrors(t *testing.T) {
	results := []DriftDetectionResult{
		{HasDrift: true},
		{Error: errors.New("error 1")},
		{HasDrift: true},
		{Error: errors.New("error 2")},
	}

	count := countErrors(results)
	assert.Equal(t, 2, count)
}

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
			name:         "Default to table",
			formatString: "unknown",
			expected:     report.OutputFormatTypeTABLE,
		},
		{
			name:         "Table format",
			formatString: "table",
			expected:     report.OutputFormatTypeTABLE,
		},
		{
			name:         "Empty string",
			formatString: "",
			expected:     report.OutputFormatTypeTABLE,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			instanceMock, parserMock, reportMock, loggerMock := createMocks(t)

			service := NewService(Config{OutputFormat: tt.formatString}, instanceMock, parserMock, reportMock, loggerMock)
			format := service.getOutputFormat()
			assert.Equal(t, tt.expected, format)
		})
	}
}

func TestGenerateInstanceReport(t *testing.T) {
	instanceMock, parserMock, reportMock, loggerMock := createMocks(t)

	// Set up test case
	service := NewService(Config{OutputFormat: "table"}, instanceMock, parserMock, reportMock, loggerMock)
	instanceID := "i-12345"
	driftResult := &driftcheck.DriftResult{
		HasDrift: true,
		Drifts: map[string]models.DriftDetail{
			"instance_type": {
				Attribute:      "instance_type",
				AWValue:        "t2.micro",
				TerraformValue: "t2.small",
			},
		},
	}

	reportMock.On("PrintReport", instanceID, mock.Anything, report.OutputFormatTypeTABLE).Return(nil)

	err := service.generateInstanceReport(instanceID, driftResult)

	assert.NoError(t, err)
}

func TestProcessInstance(t *testing.T) {
	instanceMock, parserMock, reportMock, loggerMock := createMocks(t)

	cases := []struct {
		name        string
		instanceID  string
		awsInstance *models.InstanceDetails
		awsError    error
		expectErr   bool
		expectDrift bool
	}{
		{
			name:       "Success case - no drift",
			instanceID: "i-success",
			awsInstance: &models.InstanceDetails{
				InstanceID:   "i-success",
				InstanceType: "t2.micro",
			},
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
			instanceMock.On("GetInstanceDetails", mock.Anything, tc.instanceID).Return(tc.awsInstance, tc.awsError)
			if !tc.expectErr {
				reportMock.On("PrintReport", tc.instanceID, mock.Anything, mock.Anything).Return(nil)
			}

			service := NewService(Config{}, instanceMock, parserMock, reportMock, loggerMock)
			tfConfig := &models.InstanceDetails{
				InstanceType: "t2.micro", // same as AWS for no drift
			}

			result := service.processInstance(context.Background(), tc.instanceID, tfConfig)

			if tc.expectErr {
				assert.NotNil(t, result.Error)
			} else {
				assert.Nil(t, result.Error)
			}
			assert.Equal(t, tc.expectDrift, result.HasDrift)
			assert.Equal(t, tc.instanceID, result.InstanceID)
		})
	}
}

func TestGenerateSummaryReport(t *testing.T) {
	// Capture stdout to test output
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	expectedErr := errors.New("error")
	// Set up test data
	results := []DriftDetectionResult{
		{InstanceID: "i-1", HasDrift: true},
		{InstanceID: "i-2", Error: expectedErr},
		{InstanceID: "i-3", HasDrift: false},
	}

	// Create mocks
	instanceMock, parserMock, reportMock, loggerMock := createMocks(t)

	// Configure logger mock to verify expected calls
	loggerMock.On("Error", "Instance %s: Error - %s", "i-2", expectedErr).Return()
	loggerMock.On("Info", "\nSummary: Checked %d instances, %d with drift, %d with errors",
		3, 1, 1).Return()

	// Set up service
	service := NewService(Config{}, instanceMock, parserMock, reportMock, loggerMock)

	// Run the function
	service.generateSummaryReport(results)

	// Close writer to flush the buffer
	w.Close()

	// Read the captured output
	var buf bytes.Buffer
	io.Copy(&buf, r)

	// Restore stdout
	os.Stdout = old

	// Verify logger mock was called correctly
	loggerMock.AssertExpectations(t)
}

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
		{
			name: "Successful run - with drift",
			config: Config{
				InstanceIDs: []string{"i-123", "i-456"},
				ConfigPath:  "test.tf",
			},
			mockTFConfig: &models.InstanceDetails{
				InstanceType: "t2.micro",
				Tags: map[string]string{
					"Environment": "test",
				},
			},
			mockAWSInstances: map[string]*models.InstanceDetails{
				"i-123": {
					InstanceID:   "i-123",
					InstanceType: "t2.large", // Drift in instance type
					Tags: map[string]string{
						"Environment": "test",
					},
				},
				"i-456": {
					InstanceID:   "i-456",
					InstanceType: "t2.micro", // No drift
					Tags: map[string]string{
						"Environment": "test",
					},
				},
			},
			expectedAnyDrift: true,
			expectedAnyError: false,
			expectErr:        false,
		},
		{
			name: "Successful run - no drift",
			config: Config{
				InstanceIDs: []string{"i-123"},
				ConfigPath:  "test.tf",
			},
			mockTFConfig: &models.InstanceDetails{
				InstanceType: "t2.micro",
				Tags: map[string]string{
					"Environment": "test",
				},
			},
			mockAWSInstances: map[string]*models.InstanceDetails{
				"i-123": {
					InstanceID:   "i-123",
					InstanceType: "t2.micro", // No drift
					Tags: map[string]string{
						"Environment": "test",
					},
				},
			},
			expectedAnyDrift: false,
			expectedAnyError: false,
			expectErr:        false,
		},
		{
			name: "AWS error",
			config: Config{
				InstanceIDs: []string{"i-123", "i-error"},
				ConfigPath:  "test.tf",
			},
			mockTFConfig: &models.InstanceDetails{
				InstanceType: "t2.micro",
			},
			mockAWSInstances: map[string]*models.InstanceDetails{
				"i-123": {
					InstanceID:   "i-123",
					InstanceType: "t2.micro",
				},
			},
			awsErrors: map[string]error{
				"i-error": errors.New("AWS error"),
			},
			expectedAnyDrift: false,
			expectedAnyError: true,
			expectErr:        false,
		},
		{
			name: "Terraform config error",
			config: Config{
				InstanceIDs: []string{"i-123"},
				ConfigPath:  "test.tf",
			},
			mockTFConfig:     nil,
			tfConfigError:    errors.New("failed to parse config"),
			expectedAnyDrift: false,
			expectedAnyError: false,
			expectErr:        true,
		},
		{
			name: "Invalid config - no instances",
			config: Config{
				InstanceIDs: []string{},
				ConfigPath:  "test.tf",
			},
			expectedAnyDrift: false,
			expectedAnyError: false,
			expectErr:        true,
		},
		{
			name: "With concurrency limit",
			config: Config{
				InstanceIDs:      []string{"i-123", "i-456"},
				ConfigPath:       "test.tf",
				ConcurrencyLimit: 1,
			},
			mockTFConfig: &models.InstanceDetails{
				InstanceType: "t2.micro",
			},
			mockAWSInstances: map[string]*models.InstanceDetails{
				"i-123": {
					InstanceID:   "i-123",
					InstanceType: "t2.micro",
				},
				"i-456": {
					InstanceID:   "i-456",
					InstanceType: "t2.micro",
				},
			},
			expectedAnyDrift: false,
			expectedAnyError: false,
			expectErr:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset mocks
			instanceMock, parserMock, reportMock, loggerMock := createMocks(t)

			if len(tt.config.InstanceIDs) != 0 {
				parserMock.On("ParseHCLConfig", tt.config.ConfigPath).Return(tt.mockTFConfig, tt.tfConfigError)
			}

			// Set up AWS instance expectations
			for instanceID, instance := range tt.mockAWSInstances {
				instanceMock.On("GetInstanceDetails", mock.Anything, instanceID).Return(instance, nil)
			}

			// Set up AWS error & logger expectations
			if tt.awsErrors != nil {
				for instanceID, err := range tt.awsErrors {
					instanceMock.On("GetInstanceDetails", mock.Anything, instanceID).Return(nil, err)
					loggerMock.On("Error", "Instance %s: Error - %s", instanceID, mock.Anything).Return()
				}
			}

			// Set up report expectations if not a config error
			if !tt.expectErr {
				reportMock.On("PrintReport", mock.Anything, mock.Anything, mock.Anything).Return(nil)
			}

			// Set up Summary logger expectations
			if len(tt.config.InstanceIDs) > 1 {
				loggerMock.On(
					"Info",
					"\nSummary: Checked %d instances, %d with drift, %d with errors",
					len(tt.config.InstanceIDs),
					mock.Anything,
					mock.Anything,
				).Return()
			}

			// Create the service
			service := NewService(tt.config, instanceMock, parserMock, reportMock, loggerMock)

			anyDrift, anyError, err := service.Run(context.Background())

			if tt.expectErr {
				assert.Error(t, err)
				return
			} else {
				assert.NoError(t, err)
			}

			assert.Equal(t, tt.expectedAnyDrift, anyDrift, "anyDrift mismatch")
			assert.Equal(t, tt.expectedAnyError, anyError, "anyError mismatch")
		})
	}
}
