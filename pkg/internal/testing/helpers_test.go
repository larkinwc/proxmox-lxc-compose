package testutil

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAssertNoError(t *testing.T) {
	// Test with no error - should not fail
	AssertNoError(t, nil)
	
	// Test with error - we can't easily test this without creating a sub-test
	// that we expect to fail, but we can verify the function exists and works
	// with nil errors
}

func TestAssertError(t *testing.T) {
	// Test with error - should not fail
	err := os.ErrNotExist
	AssertError(t, err)
	
	// We can't easily test the failure case without sub-tests
}

func TestAssertEqual(t *testing.T) {
	// Test with equal values
	AssertEqual(t, 42, 42)
	AssertEqual(t, "hello", "hello")
	AssertEqual(t, true, true)
	AssertEqual(t, nil, nil)
	
	// Test with different types that are equal
	var a interface{} = 42
	var b interface{} = 42
	AssertEqual(t, a, b)
}

func TestAssertContains(t *testing.T) {
	// Test string that contains substring
	AssertContains(t, "hello world", "world")
	AssertContains(t, "testing", "test")
	AssertContains(t, "abc", "abc") // Full match
	AssertContains(t, "single", "s") // Single character
}

func TestAssertNotContains(t *testing.T) {
	// Test string that does not contain substring
	AssertNotContains(t, "hello world", "xyz")
	AssertNotContains(t, "testing", "production")
	AssertNotContains(t, "abc", "def")
	AssertNotContains(t, "", "anything") // Empty string
}

func TestAssertFileExists(t *testing.T) {
	// Create a temporary file
	tempDir := t.TempDir()
	tempFile := filepath.Join(tempDir, "test-file.txt")
	
	err := os.WriteFile(tempFile, []byte("test content"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	
	// Test with existing file
	AssertFileExists(t, tempFile)
}

func TestAssertFileNotExists(t *testing.T) {
	// Test with non-existent file
	nonExistentFile := "/path/that/does/not/exist/file.txt"
	AssertFileNotExists(t, nonExistentFile)
	
	// Test with a path in temp directory that we know doesn't exist
	tempDir := t.TempDir()
	nonExistentInTemp := filepath.Join(tempDir, "does-not-exist.txt")
	AssertFileNotExists(t, nonExistentInTemp)
}

func TestAssertNotNil(t *testing.T) {
	// Test with non-nil values
	AssertNotNil(t, "string")
	AssertNotNil(t, 42)
	AssertNotNil(t, []int{1, 2, 3})
	AssertNotNil(t, map[string]int{"key": 1})
	
	// Test with pointer to something
	value := 42
	AssertNotNil(t, &value)
	
	// Test with interface containing value
	var iface interface{} = "value"
	AssertNotNil(t, iface)
}

func TestTempDir(t *testing.T) {
	// Test TempDir creation
	dir, cleanup := TempDir(t)
	
	// Verify directory exists
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		t.Errorf("TempDir should create a directory, but %s does not exist", dir)
	}
	
	// Verify it's actually a directory
	info, err := os.Stat(dir)
	if err != nil {
		t.Fatalf("Failed to stat temp directory: %v", err)
	}
	if !info.IsDir() {
		t.Errorf("TempDir should create a directory, but %s is not a directory", dir)
	}
	
	// Test cleanup function
	cleanup()
	
	// Note: t.TempDir() automatically cleans up, so the directory might still exist
	// The cleanup function is mainly for compatibility/explicit cleanup
}

func TestWriteFile(t *testing.T) {
	tempDir := t.TempDir()
	
	tests := []struct {
		name     string
		filename string
		content  string
	}{
		{"simple text file", "test.txt", "hello world"},
		{"empty file", "empty.txt", ""},
		{"file with newlines", "multiline.txt", "line1\nline2\nline3"},
		{"file with special chars", "special.txt", "special chars: !@#$%^&*()"},
		{"json content", "data.json", `{"key": "value", "number": 42}`},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filePath := WriteFile(t, tempDir, tt.filename, tt.content)
			
			// Verify file was created at expected path
			expectedPath := filepath.Join(tempDir, tt.filename)
			if filePath != expectedPath {
				t.Errorf("Expected file path %s, got %s", expectedPath, filePath)
			}
			
			// Verify file exists
			if _, err := os.Stat(filePath); os.IsNotExist(err) {
				t.Errorf("File should exist at %s", filePath)
			}
			
			// Verify file content
			actualContent, err := os.ReadFile(filePath)
			if err != nil {
				t.Fatalf("Failed to read file: %v", err)
			}
			
			if string(actualContent) != tt.content {
				t.Errorf("Expected content %q, got %q", tt.content, string(actualContent))
			}
			
			// Verify file permissions
			info, err := os.Stat(filePath)
			if err != nil {
				t.Fatalf("Failed to stat file: %v", err)
			}
			
			expectedMode := os.FileMode(0644)
			if info.Mode().Perm() != expectedMode {
				t.Errorf("Expected file mode %v, got %v", expectedMode, info.Mode().Perm())
			}
		})
	}
}

func TestWriteFile_NestedDirectory(t *testing.T) {
	tempDir := t.TempDir()
	
	// Create nested directory structure
	nestedDir := filepath.Join(tempDir, "nested", "deep")
	err := os.MkdirAll(nestedDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create nested directory: %v", err)
	}
	
	// Write file in nested directory
	content := "nested file content"
	filePath := WriteFile(t, nestedDir, "nested-file.txt", content)
	
	// Verify file was created correctly
	actualContent, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("Failed to read nested file: %v", err)
	}
	
	if string(actualContent) != content {
		t.Errorf("Expected content %q, got %q", content, string(actualContent))
	}
}

func TestContains(t *testing.T) {
	tests := []struct {
		name     string
		s        string
		substr   string
		expected bool
	}{
		{"normal contains", "hello world", "world", true},
		{"normal not contains", "hello world", "xyz", false},
		{"empty string", "", "test", false},
		{"empty substring", "test", "", false},
		{"both empty", "", "", false},
		{"exact match", "test", "test", true},
		{"substring at start", "testing", "test", true},
		{"substring at end", "unittest", "test", true},
		{"case sensitive", "Hello", "hello", false},
		{"single character", "abc", "b", true},
		{"single character not found", "abc", "x", false},
		{"longer substring", "short", "longer", false},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Contains(tt.s, tt.substr)
			if result != tt.expected {
				t.Errorf("Contains(%q, %q) = %v, want %v", tt.s, tt.substr, result, tt.expected)
			}
		})
	}
}

func TestContains_ComparedToStandardLibrary(t *testing.T) {
	// Test that our Contains function behaves consistently with strings.Contains
	// for non-empty strings
	testCases := []struct {
		s      string
		substr string
	}{
		{"hello world", "world"},
		{"hello world", "xyz"},
		{"testing", "test"},
		{"testing", "ing"},
		{"abc", "abc"},
		{"single", "s"},
		{"case", "CASE"},
	}
	
	for _, tc := range testCases {
		ourResult := Contains(tc.s, tc.substr)
		stdResult := strings.Contains(tc.s, tc.substr)
		
		// Our function should match standard library for non-empty strings
		if tc.s != "" && tc.substr != "" {
			if ourResult != stdResult {
				t.Errorf("Contains(%q, %q) = %v, but strings.Contains = %v", 
					tc.s, tc.substr, ourResult, stdResult)
			}
		}
	}
}

// Test helper functions with edge cases
func TestHelpers_EdgeCases(t *testing.T) {
	t.Run("AssertEqual with different types", func(t *testing.T) {
		// These should be equal according to Go's == operator
		AssertEqual(t, int64(42), int64(42))
		AssertEqual(t, float64(3.14), float64(3.14))
		
		// Test with zero values
		AssertEqual(t, 0, 0)
		AssertEqual(t, "", "")
		AssertEqual(t, false, false)
	})
	
	t.Run("AssertContains with special characters", func(t *testing.T) {
		AssertContains(t, "hello\nworld", "\n")
		AssertContains(t, "tab\there", "\t")
		AssertContains(t, "quote\"test", "\"")
		AssertContains(t, "backslash\\test", "\\")
	})
	
	t.Run("AssertNotContains with similar strings", func(t *testing.T) {
		AssertNotContains(t, "testing", "Testing") // Case sensitive
		AssertNotContains(t, "hello", "hello ") // Extra space
		AssertNotContains(t, "test", "tests") // Plural
	})
}

// Test that helper functions properly use t.Helper()
func TestHelpers_UseHelper(t *testing.T) {
	// We can't easily test that t.Helper() is called without complex reflection,
	// but we can verify the functions work correctly when called from helper functions
	
	helperFunction := func(t *testing.T) {
		AssertEqual(t, 1, 1)
		AssertContains(t, "test", "est")
		AssertNotNil(t, "value")
	}
	
	// This should work without issues
	helperFunction(t)
}

// Integration test using multiple helper functions together
func TestHelpers_Integration(t *testing.T) {
	// Create a temporary directory and file
	tempDir := t.TempDir()
	content := "integration test content"
	filePath := WriteFile(t, tempDir, "integration.txt", content)
	
	// Use multiple assertions
	AssertFileExists(t, filePath)
	AssertNotNil(t, filePath)
	AssertContains(t, filePath, "integration.txt")
	AssertNotContains(t, filePath, "nonexistent")
	
	// Read and verify content
	actualContent, err := os.ReadFile(filePath)
	AssertNoError(t, err)
	AssertEqual(t, content, string(actualContent))
	AssertContains(t, string(actualContent), "integration")
	
	// Test Contains function
	containsResult := Contains(string(actualContent), "test")
	AssertEqual(t, true, containsResult)
}