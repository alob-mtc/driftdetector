package driftcheck

import (
	"fmt"
	"reflect"
	"sort"
	"strings"

	"driftdetector/internal/models"
)

// DetectDrift compares AWS EC2 instance details with Terraform configuration details.
// It returns a DriftResult containing information about detected drifts.
// The attributesToCheck parameter specifies which attributes to compare.
// If attributesToCheck is empty, it checks all comparable attributes.
func DetectDrift(awsInstance, tfInstance *models.InstanceDetails, attributesToCheck []string) (*DriftResult, error) {
	if awsInstance == nil {
		return nil, fmt.Errorf("AWS instance details are nil")
	}
	if tfInstance == nil {
		return nil, fmt.Errorf("terraform instance details are nil")
	}

	result := &DriftResult{
		HasDrift:  false,
		Drifts:    make(map[string]models.DriftDetail),
		AwsConfig: awsInstance,
		TfConfig:  tfInstance,
	}

	// Define all attributes that can be compared
	allAttributes := map[string]func(aws, tf *models.InstanceDetails) (bool, any, any){
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

			// Sort both slices for comparison (to ignore order differences)
			hasDrift := false
			var awsSGs, tfSGs []string

			// Create copies of the slices to avoid modifying the originals
			if aws.SecurityGroups != nil {
				awsSGs = make([]string, len(aws.SecurityGroups))
				copy(awsSGs, aws.SecurityGroups)
				sort.Strings(awsSGs)
			}

			if tf.SecurityGroups != nil {
				tfSGs = make([]string, len(tf.SecurityGroups))
				copy(tfSGs, tf.SecurityGroups)
				sort.Strings(tfSGs)
			}

			// Compare the sorted slices
			hasDrift = !reflect.DeepEqual(awsSGs, tfSGs)

			return hasDrift, aws.SecurityGroups, tf.SecurityGroups
		},
		"subnet_id": func(aws, tf *models.InstanceDetails) (bool, any, any) {
			return aws.SubnetID != tf.SubnetID, aws.SubnetID, tf.SubnetID
		},
		// Additional attributes can be added here as the model evolves
	}

	if len(attributesToCheck) > 0 {
		// When a subset is provided, iterate over attributesToCheck directly.
		for _, attr := range attributesToCheck {
			normalizedAttr := normalizeAttributeName(attr)
			if checkFn, exists := allAttributes[normalizedAttr]; exists {
				hasDrift, awsValue, tfValue := checkFn(awsInstance, tfInstance)
				if hasDrift {
					result.HasDrift = true
					result.Drifts[normalizedAttr] = models.DriftDetail{
						Attribute:      normalizedAttr,
						AWValue:        awsValue,
						TerraformValue: tfValue,
					}
				}
			}
		}
	} else {
		// No subset provided: check all attributes except "instance_id".
		for attr, checkFn := range allAttributes {
			if attr == "instance_id" {
				continue
			}
			hasDrift, awsValue, tfValue := checkFn(awsInstance, tfInstance)
			if hasDrift {
				result.HasDrift = true
				result.Drifts[attr] = models.DriftDetail{
					Attribute:      attr,
					AWValue:        awsValue,
					TerraformValue: tfValue,
				}
			}
		}
	}

	return result, nil
}

// normalizeAttributeName standardizes attribute names for comparison.
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
