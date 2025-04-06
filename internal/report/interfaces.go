package report

import "driftdetector/internal/models"

// IPrinter is the interface for generating reports
//
//go:generate mockery --name=IPrinter --output=./mocks
type IPrinter interface {
	PrintReport(instanceID string, drifts []models.DriftDetail, format OutputFormatType) error
}
