package driftcheck

import "driftdetector/internal/models"

// DriftResult represents the drift detection result between AWS and Terraform configurations.
type DriftResult struct {
	HasDrift  bool                          // True if any drift is detected
	Drifts    map[string]models.DriftDetail // Map of attribute names to drift details
	AwsConfig *models.InstanceDetails       // The AWS configuration used for comparison
	TfConfig  *models.InstanceDetails       // The Terraform configuration used for comparison
}

// Drift is for backward compatibility with existing report code.
type Drift struct {
	Attribute string
	AWSValue  any
	TFValue   any
}

// ConvertToDrifts converts a DriftResult to a slice of Drift for backward compatibility.
func ConvertToDrifts(result *DriftResult) []Drift {
	drifts := make([]Drift, 0, len(result.Drifts))
	for _, detail := range result.Drifts {
		drifts = append(drifts, Drift{
			Attribute: detail.Attribute,
			AWSValue:  detail.AWValue,
			TFValue:   detail.TerraformValue,
		})
	}
	return drifts
}
