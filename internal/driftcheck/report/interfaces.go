package report

import "driftdetector/internal/driftcheck"

// IPrinter is the interface for generating reports
//
//go:generate mockery --name=IPrinter --output=./mocks
type IPrinter interface {
	PrintReport(instanceID string, drifts []driftcheck.Drift, format OutputFormatType) error
}
