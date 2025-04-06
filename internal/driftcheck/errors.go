package driftcheck

import (
	"errors"
	"fmt"
)

// Custom error types for better categorization
const (
	// ErrInvalidInput represents validation errors in input parameters
	ErrInvalidInput = "invalid_input"

	// ErrComparisonFailed represents errors during attribute comparison
	ErrComparisonFailed = "comparison_failed"

	// ErrResourceMissing represents a missing resource or attribute
	ErrResourceMissing = "resource_missing"
)

// DriftError represents an error that occurred during drift detection
// with additional context about what went wrong.
type DriftError struct {
	// Category helps with programmatic error handling
	Category string

	// Message provides human-readable details
	Message string

	// Attribute identifies which attribute had the error (if applicable)
	Attribute string

	// Underlying is the wrapped cause of this error
	Underlying error
}

// Error returns the error message
func (e *DriftError) Error() string {
	if e.Attribute != "" {
		return fmt.Sprintf("%s: %s (attribute: %s)", e.Category, e.Message, e.Attribute)
	}
	return fmt.Sprintf("%s: %s", e.Category, e.Message)
}

// Unwrap returns the underlying error (for errors.Is/As support)
func (e *DriftError) Unwrap() error {
	return e.Underlying
}

// NewDriftError creates a new error with the given category and details
func NewDriftError(category, message, attribute string, underlying error) *DriftError {
	return &DriftError{
		Category:   category,
		Message:    message,
		Attribute:  attribute,
		Underlying: underlying,
	}
}

// IsErrorCategory checks if an error belongs to a specific error category
func IsErrorCategory(err error, category string) bool {
	if err == nil {
		return false
	}

	// First, try direct type assertion
	var e *DriftError
	if errors.As(err, &e) {
		return e.Category == category
	}

	return false
}
