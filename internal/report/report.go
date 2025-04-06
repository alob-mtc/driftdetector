package report

import (
	"driftdetector/internal/models"
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"
)

// OutputFormatType defines the format types for the drift report.
type OutputFormatType string

const (
	// OutputFormatTypeJSON represents JSON output format
	OutputFormatTypeJSON OutputFormatType = "JSON"
	// OutputFormatTypeTABLE represents table output format
	OutputFormatTypeTABLE OutputFormatType = "TABLE"
)

// DriftReport represents a report for a single instance.
type DriftReport struct {
	InstanceID string               `json:"instance_id"`
	Drifts     []models.DriftDetail `json:"drifts"`
}

// PrintReport prints the drift report for a given instance using the specified output format.
// Supported formats: "json" (machine-readable) and "table" (human-friendly).
func PrintReport(instanceID string, drifts []models.DriftDetail, outputFormat OutputFormatType) error {
	report := DriftReport{
		InstanceID: instanceID,
		Drifts:     drifts,
	}

	switch outputFormat {
	case OutputFormatTypeJSON:
		return printJSONReport(report)
	case OutputFormatTypeTABLE:
		return printTableReport(report)
	default:
		return fmt.Errorf("unsupported output format: %s", outputFormat)
	}
}

// printJSONReport prints the report in JSON format
func printJSONReport(report DriftReport) error {
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return fmt.Errorf("error marshaling report to JSON: %w", err)
	}
	fmt.Println(string(data))
	return nil
}

// printTableReport prints the report in a human-friendly table format
func printTableReport(report DriftReport) error {
	// Using tabwriter to produce a nicely aligned table output.
	writer := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)

	// Print header
	fmt.Fprintf(writer, "\nINSTANCE ID:\t%s\n\n", report.InstanceID)
	fmt.Fprintln(writer, "ATTRIBUTE\tAWS VALUE\tTERRAFORM VALUE\tSTATUS")
	fmt.Fprintln(writer, "---------\t---------\t---------------\t------")

	// Print each attribute comparison
	for _, d := range report.Drifts {
		fmt.Fprintf(writer, "%s\t%v\t%v\t%s\n",
			d.Attribute,
			formatValueForTable(d.AWSValue),
			formatValueForTable(d.TerraformValue),
			"DRIFT")
	}

	// Print summary
	fmt.Fprintln(writer, "")
	fmt.Fprintf(writer, "Summary: %d attributes with drift found\n", len(report.Drifts))

	return writer.Flush()
}

// formatValueForTable formats values for better display in the table
func formatValueForTable(v any) string {
	if v == nil {
		return "<nil>"
	}

	// Handle empty strings
	if s, ok := v.(string); ok && s == "" {
		return "<empty>"
	}

	return fmt.Sprintf("%v", v)
}

// DefaultPrinter is the default implementation of the report printer
type DefaultPrinter struct{}

// PrintReport implements the printer interface
func (p DefaultPrinter) PrintReport(instanceID string, drifts []models.DriftDetail, format OutputFormatType) error {
	return PrintReport(instanceID, drifts, format)
}
