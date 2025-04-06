package terraform

import (
	"path/filepath"
	"testing"

	"driftdetector/pkg/logging"

	"github.com/stretchr/testify/assert"
)

func TestParseHCLConfig_CompleteInstanceDetails(t *testing.T) {
	// Get the path to the test file
	testFile := filepath.Join("testdata", "valid_instance.tf")

	// Create parser and parse the HCL config
	logger := logging.NewMockLogger()
	parser := NewParserWithLogger(logger)
	instance, err := parser.ParseHCLConfig(testFile)

	assert.NoError(t, err)
	assert.NotNil(t, instance)

	// Check all fields
	assert.Equal(t, "ami-0c55b159cbfafe1f0", instance.AMI)
	assert.Equal(t, "t2.micro", instance.InstanceType)
	assert.Equal(t, "subnet-12345", instance.SubnetID)

	// Check security groups
	assert.Len(t, instance.SecurityGroups, 2)
	assert.Equal(t, "sg-12345", instance.SecurityGroups[0])
	assert.Equal(t, "sg-67890", instance.SecurityGroups[1])
}

func TestParseHCLConfig_NoInstance(t *testing.T) {
	// Get the path to the test file
	testFile := filepath.Join("testdata", "no_instance.tf")

	// Create parser and parse the HCL config
	logger := logging.NewMockLogger()
	parser := NewParserWithLogger(logger)
	instance, err := parser.ParseHCLConfig(testFile)

	// Should get an error about no aws_instance found
	assert.Error(t, err)
	assert.Nil(t, instance)
}

func TestParseHCLConfig_InvalidHCL(t *testing.T) {
	// Get the path to the test file
	testFile := filepath.Join("testdata", "invalid_hcl.tf")

	// Create parser and parse the HCL config
	logger := logging.NewMockLogger()
	parser := NewParserWithLogger(logger)
	instance, err := parser.ParseHCLConfig(testFile)

	// Should get an error about invalid HCL
	assert.Error(t, err)
	assert.Nil(t, instance)
}

func TestParseHCLConfig_InvalidAwsInstance(t *testing.T) {
	// Get the path to the test file
	testFile := filepath.Join("testdata", "invalid_aws_instance.tf")

	// Create parser and parse the HCL config
	logger := logging.NewMockLogger()
	parser := NewParserWithLogger(logger)
	instance, err := parser.ParseHCLConfig(testFile)

	// The function should return an error for missing instance_type
	assert.Error(t, err)
	assert.Nil(t, instance)
}

func TestParseHCLConfig_NonExistentFile(t *testing.T) {
	// Parse a file that doesn't exist
	logger := logging.NewMockLogger()
	parser := NewParserWithLogger(logger)
	instance, err := parser.ParseHCLConfig("testdata/non_existent_file.tf")

	// Should get an error about file not found
	assert.Error(t, err)
	assert.Nil(t, instance)
}

// This test covers the DefaultParser implementation
func TestDefaultParser_ParseHCLConfig(t *testing.T) {
	// Test with default logger
	parser := NewDefaultParser()
	testFile := filepath.Join("testdata", "valid_instance.tf")

	// Parse the HCL config using the DefaultParser
	instance, err := parser.ParseHCLConfig(testFile)

	// Assert no error and instance is not nil
	assert.NoError(t, err)
	assert.NotNil(t, instance)

	// Check instance type
	assert.Equal(t, "t2.micro", instance.InstanceType)
}
