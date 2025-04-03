package orchestrator

import "driftdetector/internal/driftcheck"

// Config contains all the parameters needed for the drift detection process.
type Config struct {
	InstanceIDs       []string // AWS EC2 instance IDs
	ConfigPath        string   // Path to Terraform configuration file
	AttributesToCheck []string // List of attributes to check for drift
	OutputFormat      string   // Output format (json or table)
	ConcurrencyLimit  int      // Maximum number of concurrent instance checks (0 = unlimited)
}

// DriftDetectionResult contains the result of a drift detection for a single instance.
type DriftDetectionResult struct {
	InstanceID string
	HasDrift   bool
	Error      error
	Result     *driftcheck.DriftResult
}
