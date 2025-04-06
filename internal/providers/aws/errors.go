package aws

import (
	"errors"
	"fmt"
	"strings"
)

type ErrorCategory string

// Error categories for better error classification and handling
const (
	// ErrResourceNotFound is returned when a requested AWS resource doesn't exist
	ErrResourceNotFound ErrorCategory = "resource_not_found"

	// ErrPermissionDenied is returned when AWS API access is denied
	ErrPermissionDenied ErrorCategory = "permission_denied"

	// ErrThrottling is returned when AWS API throttles the request
	ErrThrottling ErrorCategory = "request_throttled"

	// ErrConfigurationError is returned when there's an issue with AWS configuration
	ErrConfigurationError ErrorCategory = "configuration_error"

	// ErrNetworkError is returned for network-related errors accessing AWS API
	ErrNetworkError ErrorCategory = "network_error"

	// ErrInvalidInput is returned when invalid input is provided
	ErrInvalidInput ErrorCategory = "invalid_input"

	// ErrInternalError is returned for unexpected internal errors
	ErrInternalError ErrorCategory = "internal_error"
)

// Error represents an error that occurred during AWS operations with
// additional context about what went wrong.
type Error struct {
	// Category for programmatic error handling
	Category ErrorCategory

	// ResourceType identifies the AWS resource type (e.g., EC2, S3)
	ResourceType string

	// ResourceID identifies the specific resource ID when applicable
	ResourceID string

	// Message provides human-readable details
	Message string

	// Underlying is the wrapped cause of this error
	Underlying error
}

// Error returns a formatted error message
func (e *Error) Error() string {
	if e.ResourceID != "" {
		return fmt.Sprintf("%s: %s [resource: %s/%s]", e.Category, e.Message, e.ResourceType, e.ResourceID)
	}
	if e.ResourceType != "" {
		return fmt.Sprintf("%s: %s [resource type: %s]", e.Category, e.Message, e.ResourceType)
	}
	return fmt.Sprintf("%s: %s", e.Category, e.Message)
}

// Unwrap returns the underlying error
func (e *Error) Unwrap() error {
	return e.Underlying
}

// NewAWSError creates a new AWS error with the specified details
func NewAWSError(category ErrorCategory, resourceType, resourceID, message string, underlying error) *Error {
	return &Error{
		Category:     category,
		ResourceType: resourceType,
		ResourceID:   resourceID,
		Message:      message,
		Underlying:   underlying,
	}
}

// IsErrorCategory checks if an error belongs to a specific error category
func IsErrorCategory(err error, category ErrorCategory) bool {
	if err == nil {
		return false
	}

	// Direct type check
	var awsErr *Error
	if errors.As(err, &awsErr) {
		return awsErr.Category == category
	}

	return false
}

// ClassifyAWSError classifies an AWS error based on its message and context
func ClassifyAWSError(err error, resourceType, resourceID string) *Error {
	if err == nil {
		return nil
	}

	errMsg := err.Error()

	switch {
	// Classify based on AWS standard error codes
	// Reference: https://docs.aws.amazon.com/AWSEC2/latest/APIReference/errors-overview.html
	case contains(errMsg, "InvalidResource") ||
		contains(errMsg, "InvalidInstanceID.NotFound") ||
		contains(errMsg, "InvalidInstanceID"):
		return NewAWSError(ErrResourceNotFound, resourceType, resourceID,
			"Resource not found", err)

	case contains(errMsg, "UnauthorizedOperation") ||
		contains(errMsg, "AuthFailure") ||
		contains(errMsg, "InvalidClientTokenId"):
		return NewAWSError(ErrPermissionDenied, resourceType, resourceID,
			"Access denied", err)

	case contains(errMsg, "RequestLimitExceeded"):
		return NewAWSError(ErrThrottling, resourceType, resourceID,
			"Request throttled", err)

	case contains(errMsg, "InvalidParameter") ||
		contains(errMsg, "ValidationError") ||
		contains(errMsg, "MalformedQueryString"):
		return NewAWSError(ErrInvalidInput, resourceType, resourceID,
			"Invalid input", err)

		// Fall back to string-based analysis for non-standard errors
	case contains(errMsg, "no such host") ||
		contains(errMsg, "connection refused") ||
		contains(errMsg, "timeout"):
		return NewAWSError(ErrNetworkError, resourceType, resourceID,
			"Network error while accessing AWS API", err)

	case contains(errMsg, "InvalidClientTokenId") ||
		contains(errMsg, "could not find region") ||
		contains(errMsg, "failed to retrieve credentials"):
		return NewAWSError(ErrConfigurationError, resourceType, resourceID,
			"AWS SDK configuration error", err)

	default:
		// If we can't classify specifically, return a general internal error
		return NewAWSError(ErrInternalError, resourceType, resourceID,
			"Internal error occurred", err)
	}
}

// contains checks if the error message contains any of the provided substrings
func contains(s string, substrings ...string) bool {
	for _, substr := range substrings {
		if strings.Contains(strings.ToLower(s), strings.ToLower(substr)) {
			return true
		}
	}
	return false
}
