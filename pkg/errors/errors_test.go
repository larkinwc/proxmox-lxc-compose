package errors

import (
	"errors"
	"fmt"
	"testing"
)

func TestErrorType_String(t *testing.T) {
	tests := []struct {
		name     string
		errType  ErrorType
		expected string
	}{
		{"Configuration error", ErrConfig, "Configuration"},
		{"Validation error", ErrValidation, "Validation"},
		{"Runtime error", ErrRuntime, "Runtime"},
		{"Container error", ErrContainer, "Container"},
		{"Network error", ErrNetwork, "Network"},
		{"Storage error", ErrStorage, "Storage"},
		{"Image error", ErrImage, "Image"},
		{"Registry error", ErrRegistry, "Registry"},
		{"System error", ErrSystem, "System"},
		{"Internal error", ErrInternal, "Internal"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.errType) != tt.expected {
				t.Errorf("ErrorType string = %v, want %v", string(tt.errType), tt.expected)
			}
		})
	}
}

func TestError_Error(t *testing.T) {
	tests := []struct {
		name     string
		err      *Error
		expected string
	}{
		{
			name: "error without cause",
			err: &Error{
				Type:    ErrConfig,
				Message: "invalid configuration",
			},
			expected: "Configuration error: invalid configuration",
		},
		{
			name: "error with cause",
			err: &Error{
				Type:    ErrValidation,
				Message: "validation failed",
				Cause:   fmt.Errorf("field is required"),
			},
			expected: "Validation error: validation failed: field is required",
		},
		{
			name: "error with empty message",
			err: &Error{
				Type:    ErrRuntime,
				Message: "",
			},
			expected: "Runtime error: ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.err.Error()
			if result != tt.expected {
				t.Errorf("Error.Error() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestNew(t *testing.T) {
	tests := []struct {
		name    string
		errType ErrorType
		msg     string
	}{
		{"config error", ErrConfig, "configuration is invalid"},
		{"validation error", ErrValidation, "field validation failed"},
		{"runtime error", ErrRuntime, "runtime failure"},
		{"empty message", ErrSystem, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := New(tt.errType, tt.msg)

			if err == nil {
				t.Fatal("New() returned nil")
			}

			if err.Type != tt.errType {
				t.Errorf("New() Type = %v, want %v", err.Type, tt.errType)
			}

			if err.Message != tt.msg {
				t.Errorf("New() Message = %v, want %v", err.Message, tt.msg)
			}

			if err.Cause != nil {
				t.Errorf("New() Cause = %v, want nil", err.Cause)
			}

			if err.Details == nil {
				t.Error("New() Details is nil, want empty map")
			}

			if len(err.Details) != 0 {
				t.Errorf("New() Details length = %v, want 0", len(err.Details))
			}
		})
	}
}

func TestWrap(t *testing.T) {
	originalErr := fmt.Errorf("original error")

	tests := []struct {
		name    string
		err     error
		errType ErrorType
		msg     string
	}{
		{"wrap fmt error", originalErr, ErrContainer, "container operation failed"},
		{"wrap nil error", nil, ErrNetwork, "network issue"},
		{"wrap custom error", New(ErrValidation, "validation error"), ErrRuntime, "runtime wrapper"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wrappedErr := Wrap(tt.err, tt.errType, tt.msg)

			if wrappedErr == nil {
				t.Fatal("Wrap() returned nil")
			}

			if wrappedErr.Type != tt.errType {
				t.Errorf("Wrap() Type = %v, want %v", wrappedErr.Type, tt.errType)
			}

			if wrappedErr.Message != tt.msg {
				t.Errorf("Wrap() Message = %v, want %v", wrappedErr.Message, tt.msg)
			}

			if wrappedErr.Cause != tt.err {
				t.Errorf("Wrap() Cause = %v, want %v", wrappedErr.Cause, tt.err)
			}

			if wrappedErr.Details == nil {
				t.Error("Wrap() Details is nil, want empty map")
			}

			if len(wrappedErr.Details) != 0 {
				t.Errorf("Wrap() Details length = %v, want 0", len(wrappedErr.Details))
			}
		})
	}
}

func TestError_WithDetails(t *testing.T) {
	tests := []struct {
		name           string
		initialDetails map[string]interface{}
		addDetails     map[string]interface{}
		expectedTotal  int
	}{
		{
			name:           "add details to empty error",
			initialDetails: nil,
			addDetails:     map[string]interface{}{"key1": "value1", "key2": 42},
			expectedTotal:  2,
		},
		{
			name:           "add details to existing details",
			initialDetails: map[string]interface{}{"existing": "value"},
			addDetails:     map[string]interface{}{"new1": "value1", "new2": true},
			expectedTotal:  3,
		},
		{
			name:           "overwrite existing detail",
			initialDetails: map[string]interface{}{"key1": "old_value"},
			addDetails:     map[string]interface{}{"key1": "new_value", "key2": "value2"},
			expectedTotal:  2,
		},
		{
			name:           "add empty details",
			initialDetails: map[string]interface{}{"existing": "value"},
			addDetails:     map[string]interface{}{},
			expectedTotal:  1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := New(ErrConfig, "test error")
			
			// Set initial details if provided
			if tt.initialDetails != nil {
				for k, v := range tt.initialDetails {
					err.Details[k] = v
				}
			}

			// Add new details
			result := err.WithDetails(tt.addDetails)

			// Should return the same error instance
			if result != err {
				t.Error("WithDetails() should return the same error instance")
			}

			// Check total number of details
			if len(err.Details) != tt.expectedTotal {
				t.Errorf("WithDetails() details count = %v, want %v", len(err.Details), tt.expectedTotal)
			}

			// Check that all added details are present
			for k, expectedValue := range tt.addDetails {
				if actualValue, exists := err.Details[k]; !exists {
					t.Errorf("WithDetails() missing key %v", k)
				} else if actualValue != expectedValue {
					t.Errorf("WithDetails() key %v = %v, want %v", k, actualValue, expectedValue)
				}
			}
		})
	}
}

func TestIsType(t *testing.T) {
	configErr := New(ErrConfig, "config error")
	validationErr := New(ErrValidation, "validation error")
	wrappedErr := Wrap(fmt.Errorf("original"), ErrRuntime, "wrapped error")
	standardErr := fmt.Errorf("standard error")

	tests := []struct {
		name     string
		err      error
		errType  ErrorType
		expected bool
	}{
		{"nil error", nil, ErrConfig, false},
		{"matching type", configErr, ErrConfig, true},
		{"non-matching type", configErr, ErrValidation, false},
		{"wrapped error matching type", wrappedErr, ErrRuntime, true},
		{"wrapped error non-matching type", wrappedErr, ErrConfig, false},
		{"standard error", standardErr, ErrConfig, false},
		{"validation error matching", validationErr, ErrValidation, true},
		{"validation error non-matching", validationErr, ErrContainer, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsType(tt.err, tt.errType)
			if result != tt.expected {
				t.Errorf("IsType(%v, %v) = %v, want %v", tt.err, tt.errType, result, tt.expected)
			}
		})
	}
}

func TestError_ChainedOperations(t *testing.T) {
	// Test chaining operations together
	originalErr := fmt.Errorf("database connection failed")
	
	err := Wrap(originalErr, ErrSystem, "system failure").
		WithDetails(map[string]interface{}{
			"component": "database",
			"retry":     3,
		}).
		WithDetails(map[string]interface{}{
			"timestamp": "2023-01-01T00:00:00Z",
			"retry":     5, // This should overwrite the previous retry value
		})

	// Check that all operations worked correctly
	if err.Type != ErrSystem {
		t.Errorf("Chained operations Type = %v, want %v", err.Type, ErrSystem)
	}

	if err.Message != "system failure" {
		t.Errorf("Chained operations Message = %v, want %v", err.Message, "system failure")
	}

	if err.Cause != originalErr {
		t.Errorf("Chained operations Cause = %v, want %v", err.Cause, originalErr)
	}

	expectedDetails := map[string]interface{}{
		"component": "database",
		"retry":     5, // Should be the overwritten value
		"timestamp": "2023-01-01T00:00:00Z",
	}

	if len(err.Details) != len(expectedDetails) {
		t.Errorf("Chained operations Details length = %v, want %v", len(err.Details), len(expectedDetails))
	}

	for k, expectedValue := range expectedDetails {
		if actualValue, exists := err.Details[k]; !exists {
			t.Errorf("Chained operations missing detail key %v", k)
		} else if actualValue != expectedValue {
			t.Errorf("Chained operations detail %v = %v, want %v", k, actualValue, expectedValue)
		}
	}
}

func TestError_ComplexScenarios(t *testing.T) {
	t.Run("deeply nested error wrapping", func(t *testing.T) {
		// Create a chain of wrapped errors
		originalErr := errors.New("file not found")
		level1 := Wrap(originalErr, ErrStorage, "storage access failed")
		level2 := Wrap(level1, ErrContainer, "container creation failed")
		level3 := Wrap(level2, ErrRuntime, "runtime error")

		// Check that IsType works correctly at each level
		if !IsType(level1, ErrStorage) {
			t.Error("Level 1 should be Storage error")
		}
		if !IsType(level2, ErrContainer) {
			t.Error("Level 2 should be Container error")
		}
		if !IsType(level3, ErrRuntime) {
			t.Error("Level 3 should be Runtime error")
		}

		// Check that the error message includes the chain
		errorStr := level3.Error()
		if !contains(errorStr, "Runtime error") {
			t.Error("Error string should contain 'Runtime error'")
		}
	})

	t.Run("error with complex details", func(t *testing.T) {
		err := New(ErrValidation, "complex validation error")
		
		complexDetails := map[string]interface{}{
			"field":       "email",
			"value":       "invalid-email",
			"constraints": []string{"required", "email_format"},
			"metadata": map[string]string{
				"source": "user_input",
				"form":   "registration",
			},
			"attempt_count": 3,
			"valid":        false,
		}

		err.WithDetails(complexDetails)

		// Verify all complex details are stored correctly
		for k, expectedValue := range complexDetails {
			if actualValue, exists := err.Details[k]; !exists {
				t.Errorf("Missing complex detail key %v", k)
			} else {
				// For complex types, just check they exist and are not nil
				switch expectedValue.(type) {
				case []string, map[string]string:
					if actualValue == nil {
						t.Errorf("Complex detail %v should not be nil", k)
					}
				default:
					if actualValue != expectedValue {
						t.Errorf("Complex detail %v = %v, want %v", k, actualValue, expectedValue)
					}
				}
			}
		}
	})
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 || 
		(len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || 
		containsAt(s, substr))))
}

func containsAt(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}