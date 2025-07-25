// Package errors defines custom error types for wallhaven_dl
package errors

import (
	"errors"
	"fmt"
)

// Application error types
var (
	ErrNoWallpapersFound = errors.New("no wallpapers found")
	ErrDownloadFailed    = errors.New("failed to download wallpaper")
	ErrScriptExecution   = errors.New("failed to execute script")
	ErrAPIRequest        = errors.New("API request failed")
	ErrInvalidResponse   = errors.New("invalid API response")
	ErrCacheOperation    = errors.New("cache operation failed")
	ErrInvalidConfig     = errors.New("invalid configuration")
	ErrFileOperation     = errors.New("file operation failed")
	ErrValidation        = errors.New("validation failed")
)

// ValidationError represents a validation error with details
type ValidationError struct {
	Field   string
	Value   string
	Message string
}

func (e ValidationError) Error() string {
	return fmt.Sprintf("validation failed for field '%s' with value '%s': %s", e.Field, e.Value, e.Message)
}

// APIError represents an API-related error
type APIError struct {
	StatusCode int
	Message    string
	Endpoint   string
}

func (e APIError) Error() string {
	return fmt.Sprintf("API error at %s: status %d - %s", e.Endpoint, e.StatusCode, e.Message)
}

// NewValidationError creates a new validation error
func NewValidationError(field, value, message string) error {
	return &ValidationError{
		Field:   field,
		Value:   value,
		Message: message,
	}
}

// NewAPIError creates a new API error
func NewAPIError(endpoint string, statusCode int, message string) error {
	return &APIError{
		Endpoint:   endpoint,
		StatusCode: statusCode,
		Message:    message,
	}
}