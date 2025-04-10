package report_test

import (
	"bytes"
	"driftdetector/internal/models"
	"io"
	"os"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"

	"driftdetector/internal/report"
)

// captureOutput temporarily redirects os.Stdout so we can capture what PrintReport writes.
func captureOutput(f func()) string {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	f()

	w.Close()
	var buf bytes.Buffer
	io.Copy(&buf, r)
	os.Stdout = old
	return buf.String()
}

func TestPrintReport_JSON(t *testing.T) {
	instanceID := "i-1234567890abcdef0"
	drifts := []models.DriftDetail{
		{
			Attribute:      "instance_type",
			AWSValue:       "t2.micro",
			TerraformValue: "t2.small",
		},
	}

	output := captureOutput(func() {
		err := report.PrintReport(&sync.Mutex{}, instanceID, drifts, report.OutputFormatTypeJSON)
		assert.NoError(t, err, "unexpected error")
	})

	// Check that the output contains JSON keys.
	assert.Contains(t, output, "\"instance_id\"", "JSON output should contain instance_id field")
	assert.Contains(t, output, "\"drifts\"", "JSON output should contain drifts field")
}

func TestPrintReport_Table(t *testing.T) {
	instanceID := "i-1234567890abcdef0"
	drifts := []models.DriftDetail{
		{
			Attribute:      "instance_type",
			AWSValue:       "t2.micro",
			TerraformValue: "t2.small",
		},
	}

	output := captureOutput(func() {
		err := report.PrintReport(&sync.Mutex{}, instanceID, drifts, report.OutputFormatTypeTABLE)
		assert.NoError(t, err, "unexpected error")
	})

	// Check that the output contains the table headers and expected values.
	assert.Contains(t, output, "INSTANCE ID:", "Table output should contain INSTANCE ID header")
	assert.Contains(t, output, "instance_type", "Table output should contain instance_type field")
	assert.Contains(t, output, "t2.micro", "Table output should contain AWS value")
	assert.Contains(t, output, "t2.small", "Table output should contain Terraform value")
}

func TestPrintReport_InvalidFormat(t *testing.T) {
	instanceID := "i-1234567890abcdef0"
	drifts := []models.DriftDetail{
		{
			Attribute:      "instance_type",
			AWSValue:       "t2.micro",
			TerraformValue: "t2.small",
		},
	}

	err := report.PrintReport(&sync.Mutex{}, instanceID, drifts, "invalid")
	assert.Error(t, err, "expected error for invalid output format")
}

func TestFormatValueForTable(t *testing.T) {
	// We need to call the package-private function, so we'll test indirectly
	// by comparing the output from PrintReport

	// Test nil value
	nilTest := []models.DriftDetail{
		{
			Attribute:      "nil_test",
			AWSValue:       nil,
			TerraformValue: "not-nil",
		},
	}

	nilOutput := captureOutput(func() {
		_ = report.PrintReport(&sync.Mutex{}, "test", nilTest, report.OutputFormatTypeTABLE)
	})

	assert.Contains(t, nilOutput, "<nil>", "Nil value should be formatted as '<nil>'")

	// Test empty string
	emptyTest := []models.DriftDetail{
		{
			Attribute:      "empty_test",
			AWSValue:       "",
			TerraformValue: "not-empty",
		},
	}

	emptyOutput := captureOutput(func() {
		_ = report.PrintReport(&sync.Mutex{}, "test", emptyTest, report.OutputFormatTypeTABLE)
	})

	assert.Contains(t, emptyOutput, "<empty>", "Empty string should be formatted as '<empty>'")
}
