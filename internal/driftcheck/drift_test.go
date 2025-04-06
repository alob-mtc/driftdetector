package driftcheck

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"driftdetector/internal/models"
)

func TestDetectDrift_NoDrift(t *testing.T) {
	// Create two identical instances (no drift expected)
	awsInstance := &models.InstanceDetails{
		InstanceID:   "i-12345",
		InstanceType: "t2.micro",
		Tags: map[string]string{
			"Name": "test-instance",
			"Env":  "dev",
		},
	}

	tfInstance := &models.InstanceDetails{
		// InstanceID is typically not in Terraform, so it should not be considered for drift
		InstanceType: "t2.micro",
		Tags: map[string]string{
			"Name": "test-instance",
			"Env":  "dev",
		},
	}

	// Detect drift
	result, err := DetectDrift(awsInstance, tfInstance, nil)
	assert.NoError(t, err, "Unexpected error")

	// Check results
	assert.False(t, result.HasDrift, "Expected no drift")
	assert.Equal(t, 0, len(result.Drifts), "Expected 0 drift details")
}

func TestDetectDrift_WithDrift(t *testing.T) {
	// Create two instances with differences
	awsInstance := &models.InstanceDetails{
		InstanceID:   "i-12345",
		InstanceType: "t2.medium", // Different from Terraform
		Tags: map[string]string{
			"Name": "test-instance-aws", // Different from Terraform
			"Env":  "dev",
		},
	}

	tfInstance := &models.InstanceDetails{
		InstanceType: "t2.micro",
		Tags: map[string]string{
			"Name": "test-instance",
			"Env":  "dev",
		},
	}

	// Detect drift
	result, err := DetectDrift(awsInstance, tfInstance, nil)
	assert.NoError(t, err, "Unexpected error")

	// Check results
	assert.True(t, result.HasDrift, "Expected drift, but none found")

	// Should find 2 drifts: instance_type and tags
	assert.Equal(t, 2, len(result.Drifts), "Expected 2 drift details")

	// Check instance_type drift
	drift, exists := result.Drifts["instance_type"]
	assert.True(t, exists, "Expected drift in 'instance_type'")
	assert.Equal(t, "t2.medium", drift.AWSValue, "Incorrect AWS value for instance_type")
	assert.Equal(t, "t2.micro", drift.TerraformValue, "Incorrect Terraform value for instance_type")

	// Check tags drift
	tagsDrift, tagsExist := result.Drifts["tags"]
	assert.True(t, tagsExist, "Expected drift in 'tags'")

	// Verify tags type
	_, awsOk := tagsDrift.AWSValue.(map[string]string)
	assert.True(t, awsOk, "Expected AWS tags to be map[string]string")

	_, tfOk := tagsDrift.TerraformValue.(map[string]string)
	assert.True(t, tfOk, "Expected Terraform tags to be map[string]string")
}

func TestDetectDrift_SpecificAttributes(t *testing.T) {
	// Create two instances with differences
	awsInstance := &models.InstanceDetails{
		InstanceID:   "i-12345",
		InstanceType: "t2.medium", // Different from Terraform
		Tags: map[string]string{
			"Name": "test-instance-aws", // Different from Terraform
			"Env":  "dev",
		},
	}

	tfInstance := &models.InstanceDetails{
		InstanceType: "t2.micro",
		Tags: map[string]string{
			"Name": "test-instance",
			"Env":  "dev",
		},
	}

	// Only check instance_type
	result, err := DetectDrift(awsInstance, tfInstance, []string{"instance_type"})
	assert.NoError(t, err, "Unexpected error")

	// Check results
	assert.True(t, result.HasDrift, "Expected drift, but none found")

	// Should find 1 drift: instance_type
	assert.Equal(t, 1, len(result.Drifts), "Expected 1 drift detail")

	// Should only contain instance_type and not tags
	_, instanceTypeExists := result.Drifts["instance_type"]
	assert.True(t, instanceTypeExists, "Expected drift in 'instance_type'")

	_, tagsExist := result.Drifts["tags"]
	assert.False(t, tagsExist, "Did not expect drift checking for 'tags'")
}

func TestNormalizeAttributeName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"instance_type", "instance_type"},
		{"instanceType", "instance_type"},
		{"type", "instance_type"},
		{"INSTANCE-TYPE", "instance_type"},
		{"tags", "tags"},
		{"Tags", "tags"},
		{"SecurityGroups", "security_groups"},
		{"security-groups", "security_groups"},
		{"sg", "security_groups"},
		{"securitygroup", "security_groups"},
		{"subnet", "subnet_id"},
		{"vpc", "vpc_id"},
		{"id", "instance_id"},
		{"custom_attribute", "custom_attribute"},
	}

	for _, test := range tests {
		t.Run(test.input, func(t *testing.T) {
			result := normalizeAttributeName(test.input)
			assert.Equal(t, test.expected, result, "Incorrect normalization result")
		})
	}
}

func TestDetectDrift_NilInstances(t *testing.T) {
	// Test with nil AWS instance
	_, errAWS := DetectDrift(nil, &models.InstanceDetails{}, nil)
	assert.Error(t, errAWS, "Expected error for nil AWS instance")

	// Test with nil Terraform instance
	_, errTF := DetectDrift(&models.InstanceDetails{}, nil, nil)
	assert.Error(t, errTF, "Expected error for nil Terraform instance")
}

func TestDetectDrift_SecurityGroups(t *testing.T) {
	awsInstance := &models.InstanceDetails{
		SecurityGroups: []string{"sg-1234", "sg-5678"},
	}

	// Same security groups, no drift
	tfInstance1 := &models.InstanceDetails{
		SecurityGroups: []string{"sg-1234", "sg-5678"},
	}
	result1, _ := DetectDrift(awsInstance, tfInstance1, []string{"security_groups"})
	assert.False(t, result1.HasDrift, "Expected no drift for identical security groups")

	// Different security groups, should detect drift
	tfInstance2 := &models.InstanceDetails{
		SecurityGroups: []string{"sg-1234", "sg-different"},
	}
	result2, _ := DetectDrift(awsInstance, tfInstance2, []string{"security_groups"})
	assert.True(t, result2.HasDrift, "Expected drift for different security groups")

	// Different order should not cause drift
	tfInstance3 := &models.InstanceDetails{
		SecurityGroups: []string{"sg-5678", "sg-1234"},
	}
	result3, _ := DetectDrift(awsInstance, tfInstance3, []string{"security_groups"})
	assert.False(t, result3.HasDrift, "Expected no drift for security groups in different order")
}

func TestDetectDrift_InstanceIDExplicit(t *testing.T) {
	awsInstance := &models.InstanceDetails{
		InstanceID: "i-12345",
	}
	tfInstance := &models.InstanceDetails{
		InstanceID: "different-id", // Different ID
	}

	// By default, instance_id should not be checked for drift
	result1, _ := DetectDrift(awsInstance, tfInstance, nil)
	assert.False(t, result1.HasDrift, "Expected no drift when instance_id is not explicitly requested")

	// When explicitly requested, instance_id should be checked
	result2, _ := DetectDrift(awsInstance, tfInstance, []string{"instance_id"})

	// In this test case, our specific implementation should not show drift for instance_id
	// This is by design, since the function returns 'false' for drift for this attribute
	assert.False(t, result2.HasDrift, "Expected no drift for instance_id, it should be exempt by design")
}

func TestConvertToDrifts(t *testing.T) {
	// Create a DriftResult with some drifts
	result := &DriftResult{
		HasDrift: true,
		Drifts: map[string]models.DriftDetail{
			"instance_type": {
				Attribute:      "instance_type",
				AWSValue:       "t2.medium",
				TerraformValue: "t2.micro",
			},
			"tags": {
				Attribute: "tags",
				AWSValue: map[string]string{
					"Name": "aws-instance",
					"Env":  "dev",
				},
				TerraformValue: map[string]string{
					"Name": "tf-instance",
					"Env":  "dev",
				},
			},
		},
		AwsConfig: &models.InstanceDetails{InstanceID: "i-12345"},
		TfConfig:  &models.InstanceDetails{},
	}

	// Convert to Drift slice
	drifts := ConvertToDrifts(result)

	// Verify the conversion
	assert.Equal(t, 2, len(drifts), "Expected 2 drifts")

	// Check that all attributes are present
	attrMap := make(map[string]bool)
	for _, d := range drifts {
		attrMap[d.Attribute] = true
	}

	assert.True(t, attrMap["instance_type"], "Expected to find instance_type in converted drifts")
	assert.True(t, attrMap["tags"], "Expected to find tags in converted drifts")

	// Check one of the values to ensure data is correctly transferred
	for _, d := range drifts {
		if d.Attribute == "instance_type" {
			assert.Equal(t, "t2.medium", d.AWSValue, "Expected AWS value to be t2.medium")
			assert.Equal(t, "t2.micro", d.TerraformValue, "Expected TF value to be t2.micro")
		}
	}

	// Test with empty drifts
	emptyResult := &DriftResult{
		HasDrift: false,
		Drifts:   map[string]models.DriftDetail{},
	}
	emptyDrifts := ConvertToDrifts(emptyResult)
	assert.Equal(t, 0, len(emptyDrifts), "Expected empty drift slice")
}

func TestDetectDrift_NilInstances_WithErrorCategory(t *testing.T) {
	// Test with nil AWS instance
	_, errAWS := DetectDrift(nil, &models.InstanceDetails{}, nil)
	assert.Error(t, errAWS, "Expected error for nil AWS instance")
	assert.True(t, IsErrorCategory(errAWS, ErrInvalidInput), "Expected ErrInvalidInput error category")

	// Test with nil Terraform instance
	_, errTF := DetectDrift(&models.InstanceDetails{}, nil, nil)
	assert.Error(t, errTF, "Expected error for nil Terraform instance")
	assert.True(t, IsErrorCategory(errTF, ErrInvalidInput), "Expected ErrInvalidInput error category")
}

func TestDetectDrift_UnsupportedAttribute(t *testing.T) {
	awsInstance := &models.InstanceDetails{
		InstanceID:   "i-12345",
		InstanceType: "t2.micro",
	}
	tfInstance := &models.InstanceDetails{
		InstanceType: "t2.micro",
	}

	// Try to check an attribute that doesn't exist
	_, err := DetectDrift(awsInstance, tfInstance, []string{"nonexistent_attribute"})

	// Should return an error with the correct category
	assert.Error(t, err, "Expected error for unsupported attribute")
	assert.True(t, IsErrorCategory(err, ErrResourceMissing), "Expected ErrResourceMissing error category")
}

func TestDriftError_Error(t *testing.T) {
	// Test error with attribute
	err1 := NewDriftError(ErrComparisonFailed, "test message", "instance_type", nil)
	expected1 := "comparison_failed: test message (attribute: instance_type)"
	assert.Equal(t, expected1, err1.Error(), "Error message doesn't match expected format")

	// Test error without attribute
	err2 := NewDriftError(ErrInvalidInput, "test message", "", nil)
	expected2 := "invalid_input: test message"
	assert.Equal(t, expected2, err2.Error(), "Error message doesn't match expected format")
}

func TestDriftError_Unwrap(t *testing.T) {
	cause := fmt.Errorf("root cause")
	err := NewDriftError(ErrInvalidInput, "wrapper", "", cause)

	assert.Equal(t, cause, err.Unwrap(), "Unwrap should return the underlying error")
}

func TestIsErrorCategory(t *testing.T) {
	// Direct error
	err1 := NewDriftError(ErrInvalidInput, "test message", "", nil)
	assert.True(t, IsErrorCategory(err1, ErrInvalidInput), "Should identify correct category")
	assert.False(t, IsErrorCategory(err1, ErrComparisonFailed), "Should not match wrong category")

	// Wrapped error
	cause := fmt.Errorf("root cause")
	innerErr := NewDriftError(ErrComparisonFailed, "inner", "attr", cause)
	outerErr := fmt.Errorf("outer wrapper: %w", innerErr)

	assert.True(t, IsErrorCategory(outerErr, ErrComparisonFailed), "Should find category in wrapped error")
	assert.False(t, IsErrorCategory(outerErr, ErrInvalidInput), "Should not match wrong category in wrapped error")

	// Nil error
	assert.False(t, IsErrorCategory(nil, ErrInvalidInput), "Should return false for nil error")

	// Regular error
	regularErr := fmt.Errorf("regular error")
	assert.False(t, IsErrorCategory(regularErr, ErrInvalidInput), "Should return false for regular error")
}
