package driftcheck

import (
	"driftdetector/internal/models"
)

// DriftResult represents the drift detection result between AWS and Terraform configurations.
type DriftResult struct {
	HasDrift  bool                          // True if any drift is detected
	Drifts    map[string]models.DriftDetail // Map of attribute names to drift details
	AwsConfig *models.InstanceDetails       // The AWS configuration used for comparison
	TfConfig  *models.InstanceDetails       // The Terraform configuration used for comparison
}

// ConvertToDrifts converts a DriftResult to a slice of Drift for backward compatibility.
func ConvertToDrifts(result *DriftResult) []models.DriftDetail {
	drifts := make([]models.DriftDetail, 0, len(result.Drifts))
	for _, detail := range result.Drifts {
		drifts = append(drifts, models.DriftDetail{
			Attribute:      detail.Attribute,
			AWSValue:       detail.AWSValue,
			TerraformValue: detail.TerraformValue,
		})
	}
	return drifts
}
