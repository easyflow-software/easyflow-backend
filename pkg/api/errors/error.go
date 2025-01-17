package errors

import (
	"easyflow-backend/pkg/enum"
)

// Represents a standardized error response for the API
type ApiError struct {
	// Code represents the HTTP status code
	Code int `json:"code"`

	// Error represents a predefined error code from the enum package
	Error enum.ErrorCode `json:"error"`

	// Details contains additional error information (optional)
	Details interface{} `json:"details,omitempty"`
}
