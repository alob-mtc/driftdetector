package driftcheck

import (
	"fmt"
	"reflect"
	"sort"
	"strings"

	"driftdetector/internal/models"
)

// AttributeComparator is a function type that compares two attributes
// and returns whether they differ, along with their values.
type AttributeComparator func(aws, tf *models.InstanceDetails) (hasDrift bool, awsValue any, tfValue any)

// DetectDrift compares AWS EC2 instance details with Terraform configuration details.
// It returns a DriftResult containing information about detected drifts.
// The attributesToCheck parameter specifies which attributes to compare.
// If attributesToCheck is empty, it checks all comparable attributes.
func DetectDrift(awsInstance, tfInstance *models.InstanceDetails, attributesToCheck []string) (*DriftResult, error) {
	// Validate input parameters
	if awsInstance == nil {
		return nil, fmt.Errorf("AWS instance details are nil")
	}
	if tfInstance == nil {
		return nil, fmt.Errorf("terraform instance details are nil")
	}

	// Initialize the result structure
	result := &DriftResult{
		HasDrift:  false,
		Drifts:    make(map[string]models.DriftDetail),
		AwsConfig: awsInstance,
		TfConfig:  tfInstance,
	}

	// Get the comparators for all supported attributes
	allAttributes := getAttributeComparators()

	// Determine which attributes to check
	if len(attributesToCheck) > 0 {
		// When a subset is provided, check only those attributes
		checkSpecificAttributes(result, awsInstance, tfInstance, attributesToCheck, allAttributes)
	} else {
		// No subset provided: check all attributes except "instance_id"
		checkAllAttributes(result, awsInstance, tfInstance, allAttributes)
	}

	return result, nil
}

// getAttributeComparators returns a map of attribute names to comparison functions.
// This allows for easy extension with new attributes without modifying the main logic.
func getAttributeComparators() map[string]AttributeComparator {
	return map[string]AttributeComparator{
		// Skip instance_id since it's not defined in HCL and is assigned by AWS
		"instance_type": func(aws, tf *models.InstanceDetails) (bool, any, any) {
			return aws.InstanceType != tf.InstanceType, aws.InstanceType, tf.InstanceType
		},
		"tags": func(aws, tf *models.InstanceDetails) (bool, any, any) {
			return !reflect.DeepEqual(aws.Tags, tf.Tags), aws.Tags, tf.Tags
		},
		"ami": func(aws, tf *models.InstanceDetails) (bool, any, any) {
			return aws.AMI != tf.AMI, aws.AMI, tf.AMI
		},
		"security_groups": func(aws, tf *models.InstanceDetails) (bool, any, any) {
			// Compare security groups, if they exist
			if aws.SecurityGroups == nil && tf.SecurityGroups == nil {
				return false, nil, nil
			}

			// Create sorted copies of the slices to ignore order differences
			awsSGs := sortedCopy(aws.SecurityGroups)
			tfSGs := sortedCopy(tf.SecurityGroups)

			// Compare the sorted slices
			return !reflect.DeepEqual(awsSGs, tfSGs), aws.SecurityGroups, tf.SecurityGroups
		},
		"subnet_id": func(aws, tf *models.InstanceDetails) (bool, any, any) {
			return aws.SubnetID != tf.SubnetID, aws.SubnetID, tf.SubnetID
		},
		// Additional attributes can be added here as the model evolves
	}
}

// sortedCopy creates a sorted copy of a string slice
func sortedCopy(original []string) []string {
	if original == nil {
		return nil
	}

	// Create a copy of the slice to avoid modifying the original
	result := make([]string, len(original))
	copy(result, original)
	sort.Strings(result)
	return result
}

// checkSpecificAttributes checks for drift in a specific set of attributes
func checkSpecificAttributes(
	result *DriftResult,
	awsInstance,
	tfInstance *models.InstanceDetails,
	attributesToCheck []string,
	allAttributes map[string]AttributeComparator,
) {
	for _, attr := range attributesToCheck {
		normalizedAttr := normalizeAttributeName(attr)
		if checkFn, exists := allAttributes[normalizedAttr]; exists {
			checkAttributeAndUpdateResult(result, normalizedAttr, checkFn, awsInstance, tfInstance)
		}
		// If an attribute doesn't exist in the map, it's silently ignored
	}
}

// checkAllAttributes checks for drift in all available attributes except instance_id
func checkAllAttributes(
	result *DriftResult,
	awsInstance,
	tfInstance *models.InstanceDetails,
	allAttributes map[string]AttributeComparator,
) {
	for attr, checkFn := range allAttributes {
		checkAttributeAndUpdateResult(result, attr, checkFn, awsInstance, tfInstance)
	}
}

// checkAttributeAndUpdateResult checks a single attribute for drift and updates the result
func checkAttributeAndUpdateResult(
	result *DriftResult,
	attrName string,
	checkFn AttributeComparator,
	awsInstance,
	tfInstance *models.InstanceDetails,
) {
	hasDrift, awsValue, tfValue := checkFn(awsInstance, tfInstance)
	if hasDrift {
		// Mark the overall result as having drift
		result.HasDrift = true

		// Record the specific drift details
		result.Drifts[attrName] = models.DriftDetail{
			Attribute:      attrName,
			AWSValue:       awsValue,
			TerraformValue: tfValue,
		}
	}
}

// normalizeAttributeName standardizes attribute names for comparison.
// This allows users to specify attributes with different formats (e.g., "instance-type" or "instanceType")
// and still have them correctly matched to the appropriate comparator.
func normalizeAttributeName(attr string) string {
	// Convert to lowercase
	normalized := strings.ToLower(attr)

	// Replace common separators with underscore
	normalized = strings.ReplaceAll(normalized, "-", "_")
	normalized = strings.ReplaceAll(normalized, " ", "_")

	// Handle special cases
	switch normalized {
	case "type", "instancetype":
		return "instance_type"
	case "sg", "securitygroup", "security_group", "securitygroups", "security_groups":
		return "security_groups"
	case "subnet":
		return "subnet_id"
	case "vpc":
		return "vpc_id"
	case "id":
		return "instance_id"
	}

	return normalized
}
