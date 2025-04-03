package terraform

import "driftdetector/internal/models"

// IProvider is the interface for Terraform operations
//
//go:generate mockery --name=IProvider --output=./mocks
type IProvider interface {
	ParseHCLConfig(configPath string) (*models.InstanceDetails, error)
}
