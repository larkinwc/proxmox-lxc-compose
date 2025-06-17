package logging

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

func TestConfig_Validation(t *testing.T) {
	tests := []struct {
		name   string
		config Config
		valid  bool
	}{
		{
			name: "valid debug config",
			config: Config{
				Level:         "debug",
				Development:   true,
				DisableCaller: false,
			},
			valid: true,
		},
		{
			name: "valid info config",
			config: Config{
				Level:         "info",
				Development:   false,
				DisableCaller: true,
			},
			valid: true,
		},
		{
			name: "valid warn config",
			config: Config{
				Level:         "warn",
				Development:   false,
				DisableCaller: false,
			},
			valid: true,
		},
		{
			name: "valid error config",
			config: Config{
				Level:         "error",
				Development:   true,
				DisableCaller: true,
			},
			valid: true,
		},
		{
			name: "invalid log level",
			config: Config{
				Level:         "invalid",
				Development:   false,
				DisableCaller: false,
			},
			valid: false,
		},
		{
			name: "empty log level",
			config: Config{
				Level:         "",
				Development:   false,
				DisableCaller: false,
			},
			valid: false,
		},
		{
			name: "uppercase log level",
			config: Config{
				Level:         "DEBUG",
				Development:   false,
				DisableCaller: false,
			},
			valid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Init(tt.config)
			if tt.valid && err != nil {
				t.Errorf("Init() with valid config returned error: %v", err)
			}
			if !tt.valid && err == nil {
				t.Errorf("Init() with invalid config should return error")
			}
		})
	}
}

func TestInit_LogLevels(t *testing.T) {
	tests := []struct {
		name          string
		level         string
		expectedLevel zapcore.Level
	}{
		{"debug level", "debug", zapcore.DebugLevel},
		{"info level", "info", zapcore.InfoLevel},
		{"warn level", "warn", zapcore.WarnLevel},
		{"error level", "error", zapcore.ErrorLevel},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := Config{
				Level:       tt.level,
				Development: false,
			}

			err := Init(config)
			if err != nil {
				t.Fatalf("Init() failed: %v", err)
			}

			if Level.Level() != tt.expectedLevel {
				t.Errorf("Init() set level = %v, want %v", Level.Level(), tt.expectedLevel)
			}
		})
	}
}

func TestInit_DevelopmentMode(t *testing.T) {
	tests := []struct {
		name        string
		development bool
	}{
		{"production mode", false},
		{"development mode", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := Config{
				Level:       "info",
				Development: tt.development,
			}

			err := Init(config)
			if err != nil {
				t.Fatalf("Init() failed: %v", err)
			}

			// Verify logger was initialized
			if log == nil {
				t.Error("Init() did not initialize global logger")
			}
		})
	}
}

func TestInit_DisableCaller(t *testing.T) {
	tests := []struct {
		name          string
		disableCaller bool
	}{
		{"caller enabled", false},
		{"caller disabled", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := Config{
				Level:         "info",
				Development:   false,
				DisableCaller: tt.disableCaller,
			}

			err := Init(config)
			if err != nil {
				t.Fatalf("Init() failed: %v", err)
			}

			// Verify logger was initialized
			if log == nil {
				t.Error("Init() did not initialize global logger")
			}
		})
	}
}

func TestLoggingFunctions(t *testing.T) {
	// Create a test logger with observer to capture log output
	core, recorded := observer.New(zapcore.DebugLevel)
	testLogger := zap.New(core).Sugar()
	
	// Temporarily replace the global logger
	originalLogger := log
	log = testLogger
	defer func() { log = originalLogger }()

	tests := []struct {
		name     string
		logFunc  func(string, ...interface{})
		level    zapcore.Level
		message  string
		keyVals  []interface{}
	}{
		{
			name:     "debug message",
			logFunc:  Debug,
			level:    zapcore.DebugLevel,
			message:  "debug message",
			keyVals:  []interface{}{"key1", "value1"},
		},
		{
			name:     "info message",
			logFunc:  Info,
			level:    zapcore.InfoLevel,
			message:  "info message",
			keyVals:  []interface{}{"key2", "value2"},
		},
		{
			name:     "warn message",
			logFunc:  Warn,
			level:    zapcore.WarnLevel,
			message:  "warn message",
			keyVals:  []interface{}{"key3", "value3"},
		},
		{
			name:     "error message",
			logFunc:  Error,
			level:    zapcore.ErrorLevel,
			message:  "error message",
			keyVals:  []interface{}{"key4", "value4"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear previous recordings
			recorded.TakeAll()

			// Call the logging function
			tt.logFunc(tt.message, tt.keyVals...)

			// Verify the log was recorded
			logs := recorded.All()
			if len(logs) != 1 {
				t.Fatalf("Expected 1 log entry, got %d", len(logs))
			}

			entry := logs[0]
			if entry.Level != tt.level {
				t.Errorf("Log level = %v, want %v", entry.Level, tt.level)
			}

			if entry.Message != tt.message {
				t.Errorf("Log message = %v, want %v", entry.Message, tt.message)
			}

			// Check key-value pairs
			if len(tt.keyVals) > 0 {
				expectedKey := tt.keyVals[0].(string)
				expectedValue := tt.keyVals[1]
				
				if field, exists := entry.ContextMap()[expectedKey]; !exists {
					t.Errorf("Expected key %v not found in log context", expectedKey)
				} else if field != expectedValue {
					t.Errorf("Log context[%v] = %v, want %v", expectedKey, field, expectedValue)
				}
			}
		})
	}
}

func TestLoggingFunctions_MultipleKeyValues(t *testing.T) {
	// Create a test logger with observer to capture log output
	core, recorded := observer.New(zapcore.DebugLevel)
	testLogger := zap.New(core).Sugar()
	
	// Temporarily replace the global logger
	originalLogger := log
	log = testLogger
	defer func() { log = originalLogger }()

	// Test with multiple key-value pairs
	Debug("test message", "key1", "value1", "key2", 42, "key3", true)

	logs := recorded.All()
	if len(logs) != 1 {
		t.Fatalf("Expected 1 log entry, got %d", len(logs))
	}

	entry := logs[0]
	context := entry.ContextMap()

	expectedPairs := map[string]interface{}{
		"key1": "value1",
		"key2": int64(42), // zap converts int to int64
		"key3": true,
	}

	for key, expectedValue := range expectedPairs {
		if actualValue, exists := context[key]; !exists {
			t.Errorf("Expected key %v not found in log context", key)
		} else if actualValue != expectedValue {
			t.Errorf("Log context[%v] = %v (type %T), want %v (type %T)", 
				key, actualValue, actualValue, expectedValue, expectedValue)
		}
	}
}

func TestLoggingFunctions_EmptyKeyValues(t *testing.T) {
	// Create a test logger with observer to capture log output
	core, recorded := observer.New(zapcore.DebugLevel)
	testLogger := zap.New(core).Sugar()
	
	// Temporarily replace the global logger
	originalLogger := log
	log = testLogger
	defer func() { log = originalLogger }()

	// Test with no key-value pairs
	Info("simple message")

	logs := recorded.All()
	if len(logs) != 1 {
		t.Fatalf("Expected 1 log entry, got %d", len(logs))
	}

	entry := logs[0]
	if entry.Message != "simple message" {
		t.Errorf("Log message = %v, want %v", entry.Message, "simple message")
	}

	if entry.Level != zapcore.InfoLevel {
		t.Errorf("Log level = %v, want %v", entry.Level, zapcore.InfoLevel)
	}
}

func TestInit_ErrorHandling(t *testing.T) {
	// Test invalid log level
	config := Config{
		Level: "invalid_level",
	}

	err := Init(config)
	if err == nil {
		t.Error("Init() with invalid level should return error")
	}

	expectedError := "invalid log level: invalid_level"
	if err.Error() != expectedError {
		t.Errorf("Init() error = %v, want %v", err.Error(), expectedError)
	}
}

func TestInit_GlobalStateManagement(t *testing.T) {
	// Save original state
	originalLogger := log
	originalLevel := Level.Level()

	defer func() {
		// Restore original state
		log = originalLogger
		Level.SetLevel(originalLevel)
	}()

	// Test that Init properly sets global state
	config := Config{
		Level:       "warn",
		Development: true,
	}

	err := Init(config)
	if err != nil {
		t.Fatalf("Init() failed: %v", err)
	}

	// Verify global logger was set
	if log == nil {
		t.Error("Init() did not set global logger")
	}

	// Verify global level was set
	if Level.Level() != zapcore.WarnLevel {
		t.Errorf("Init() set level = %v, want %v", Level.Level(), zapcore.WarnLevel)
	}

	// Test that subsequent Init calls replace the logger
	config2 := Config{
		Level:       "error",
		Development: false,
	}

	err = Init(config2)
	if err != nil {
		t.Fatalf("Second Init() failed: %v", err)
	}

	if Level.Level() != zapcore.ErrorLevel {
		t.Errorf("Second Init() set level = %v, want %v", Level.Level(), zapcore.ErrorLevel)
	}
}

// TestFatal_ExitBehavior tests the Fatal function's exit behavior
// This test runs in a separate process to avoid terminating the test suite
func TestFatal_ExitBehavior(t *testing.T) {
	if os.Getenv("TEST_FATAL") == "1" {
		// This is the subprocess that will call Fatal
		config := Config{
			Level:       "error",
			Development: false,
		}
		Init(config)
		Fatal("test fatal message", "key", "value")
		return
	}

	// Run the test in a subprocess
	cmd := exec.Command(os.Args[0], "-test.run=TestFatal_ExitBehavior")
	cmd.Env = append(os.Environ(), "TEST_FATAL=1")
	
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	
	err := cmd.Run()
	
	// Fatal should cause the process to exit with non-zero status
	if err == nil {
		t.Error("Fatal() should cause process to exit with non-zero status")
	}

	// Check that the process exited with status 1
	if exitError, ok := err.(*exec.ExitError); ok {
		if exitError.ExitCode() != 1 {
			t.Errorf("Fatal() exit code = %d, want 1", exitError.ExitCode())
		}
	} else {
		t.Errorf("Fatal() should cause exit error, got: %v", err)
	}
}

func TestFatal_LoggingBehavior(t *testing.T) {
	// Note: We can't easily test the Fatal function directly because it calls os.Exit(1)
	// which would terminate the test process. The Fatal function is simple enough that
	// we can verify its behavior through the subprocess test above.
	// Here we just verify that the function exists and has the right signature.
	
	// Initialize logger first
	config := Config{
		Level:       "error",
		Development: false,
	}
	err := Init(config)
	if err != nil {
		t.Fatalf("Init() failed: %v", err)
	}

	// We can verify that Fatal function exists by checking it's not nil
	// Functions in Go are never nil unless explicitly set to nil, so this is just a sanity check
	// The real test is in the subprocess test above
	t.Log("Fatal function exists and is available for use")
}

func TestLoggingIntegration(t *testing.T) {
	// Test a complete logging workflow
	config := Config{
		Level:         "debug",
		Development:   true,
		DisableCaller: false,
	}

	err := Init(config)
	if err != nil {
		t.Fatalf("Init() failed: %v", err)
	}

	// Create a test logger with observer to capture log output
	core, recorded := observer.New(zapcore.DebugLevel)
	testLogger := zap.New(core).Sugar()
	
	// Temporarily replace the global logger
	originalLogger := log
	log = testLogger
	defer func() { log = originalLogger }()

	// Log messages at different levels
	Debug("debug workflow", "step", 1)
	Info("info workflow", "step", 2)
	Warn("warn workflow", "step", 3)
	Error("error workflow", "step", 4)

	logs := recorded.All()
	if len(logs) != 4 {
		t.Fatalf("Expected 4 log entries, got %d", len(logs))
	}

	expectedLevels := []zapcore.Level{
		zapcore.DebugLevel,
		zapcore.InfoLevel,
		zapcore.WarnLevel,
		zapcore.ErrorLevel,
	}

	for i, entry := range logs {
		if entry.Level != expectedLevels[i] {
			t.Errorf("Log entry %d level = %v, want %v", i, entry.Level, expectedLevels[i])
		}

		expectedMessage := fmt.Sprintf("%s workflow", strings.ToLower(expectedLevels[i].String()))
		if entry.Message != expectedMessage {
			t.Errorf("Log entry %d message = %v, want %v", i, entry.Message, expectedMessage)
		}

		// Check step value
		if step, exists := entry.ContextMap()["step"]; !exists {
			t.Errorf("Log entry %d missing 'step' key", i)
		} else if step != int64(i+1) {
			t.Errorf("Log entry %d step = %v, want %v", i, step, i+1)
		}
	}
}