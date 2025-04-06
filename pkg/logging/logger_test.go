package logging

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLogLevels(t *testing.T) {
	buf := new(bytes.Buffer)
	logger := NewDefaultLogger()
	logger.SetOutput(buf)

	tests := []struct {
		name     string
		level    LogLevel
		logFunc  func(string, ...interface{})
		message  string
		expected bool // Whether the message should be logged
	}{
		{
			name:     "Debug logs at DEBUG level",
			level:    DEBUG,
			logFunc:  logger.Debug,
			message:  "debug message",
			expected: true,
		},
		{
			name:     "Info logs at DEBUG level",
			level:    DEBUG,
			logFunc:  logger.Info,
			message:  "info message",
			expected: true,
		},
		{
			name:     "Debug doesn't log at INFO level",
			level:    INFO,
			logFunc:  logger.Debug,
			message:  "debug message",
			expected: false,
		},
		{
			name:     "Info logs at INFO level",
			level:    INFO,
			logFunc:  logger.Info,
			message:  "info message",
			expected: true,
		},
		{
			name:     "Warn logs at INFO level",
			level:    INFO,
			logFunc:  logger.Warn,
			message:  "warn message",
			expected: true,
		},
		{
			name:     "Info doesn't log at WARN level",
			level:    WARN,
			logFunc:  logger.Info,
			message:  "info message",
			expected: false,
		},
		{
			name:     "Warn logs at WARN level",
			level:    WARN,
			logFunc:  logger.Warn,
			message:  "warn message",
			expected: true,
		},
		{
			name:     "Error logs at WARN level",
			level:    WARN,
			logFunc:  logger.Error,
			message:  "error message",
			expected: true,
		},
		{
			name:     "Warn doesn't log at ERROR level",
			level:    ERROR,
			logFunc:  logger.Warn,
			message:  "warn message",
			expected: false,
		},
		{
			name:     "Error logs at ERROR level",
			level:    ERROR,
			logFunc:  logger.Error,
			message:  "error message",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf.Reset() // Clear the buffer
			logger.SetLevel(tt.level)
			tt.logFunc(tt.message)

			if tt.expected {
				assert.Contains(t, buf.String(), tt.message, "Expected message to be logged")
			} else {
				assert.NotContains(t, buf.String(), tt.message, "Expected message not to be logged")
			}
		})
	}
}

func TestLogFormatting(t *testing.T) {
	buf := new(bytes.Buffer)
	logger := NewDefaultLogger()
	logger.SetOutput(buf)
	logger.SetLevel(DEBUG)

	logger.Info("Test message with %s", "formatting")
	output := buf.String()

	// Check timestamp format is present
	assert.Contains(t, output, "[20", "Should contain timestamp prefix")

	// Check log level is present
	assert.Contains(t, output, "INFO", "Should contain log level")

	// Check message with formatting
	assert.Contains(t, output, "Test message with formatting", "Should contain formatted message")
}
