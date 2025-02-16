package errors

import (
	"fmt"
)

// ErrorType represents the type of error
type ErrorType string

const (
	// Configuration errors
	ErrConfig     ErrorType = "Configuration"
	ErrValidation ErrorType = "Validation"
	// Runtime errors
	ErrRuntime   ErrorType = "Runtime"
	ErrContainer ErrorType = "Container"
	ErrNetwork   ErrorType = "Network"
	ErrStorage   ErrorType = "Storage"
	// Image related errors
	ErrImage    ErrorType = "Image"
	ErrRegistry ErrorType = "Registry"
	// System errors
	ErrSystem   ErrorType = "System"
	ErrInternal ErrorType = "Internal"
)

// Error represents a structured error
type Error struct {
	Type    ErrorType
	Message string
	Cause   error
	Details map[string]interface{}
}

func (e *Error) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s error: %s: %v", e.Type, e.Message, e.Cause)
	}
	return fmt.Sprintf("%s error: %s", e.Type, e.Message)
}

// New creates a new error with the given type and message
func New(errType ErrorType, msg string) *Error {
	return &Error{
		Type:    errType,
		Message: msg,
		Details: make(map[string]interface{}),
	}
}

// Wrap wraps an existing error with additional context
func Wrap(err error, errType ErrorType, msg string) *Error {
	return &Error{
		Type:    errType,
		Message: msg,
		Cause:   err,
		Details: make(map[string]interface{}),
	}
}

// WithDetails adds details to an error
func (e *Error) WithDetails(details map[string]interface{}) *Error {
	for k, v := range details {
		e.Details[k] = v
	}
	return e
}

// IsType checks if an error is of a specific type
func IsType(err error, errType ErrorType) bool {
	if err == nil {
		return false
	}
	if e, ok := err.(*Error); ok {
		return e.Type == errType
	}
	return false
}
