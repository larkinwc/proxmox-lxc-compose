package testutil

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

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
	
	// Verify directory name pattern
	if !strings.Contains(dir, "lxc-compose-test-") {
		t.Errorf("Expected temp directory to contain 'lxc-compose-test-', got: %s", dir)
	}
	
	// Test cleanup function
	cleanup()
	
	// Verify directory is removed after cleanup
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Errorf("Directory should be removed after cleanup, but %s still exists", dir)
	}
}

func TestTempDir_Multiple(t *testing.T) {
	// Test creating multiple temp directories
	dir1, cleanup1 := TempDir(t)
	dir2, cleanup2 := TempDir(t)
	
	defer cleanup1()
	defer cleanup2()
	
	// Verify directories are different
	if dir1 == dir2 {
		t.Error("Multiple TempDir calls should create different directories")
	}
	
	// Verify both directories exist
	if _, err := os.Stat(dir1); os.IsNotExist(err) {
		t.Errorf("First temp directory should exist: %s", dir1)
	}
	if _, err := os.Stat(dir2); os.IsNotExist(err) {
		t.Errorf("Second temp directory should exist: %s", dir2)
	}
}

func TestWriteFile(t *testing.T) {
	tempDir, cleanup := TempDir(t)
	defer cleanup()
	
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
		{"yaml content", "config.yaml", "key: value\nnumber: 42"},
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
	tempDir, cleanup := TempDir(t)
	defer cleanup()
	
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

func TestAssertNoError(t *testing.T) {
	// Test with no error - should not fail
	AssertNoError(t, nil)
	
	// We can't easily test the failure case without creating a sub-test
	// that we expect to fail, but we can verify the function works with nil
}

func TestAssertError(t *testing.T) {
	// Test with error - should not fail
	err := os.ErrNotExist
	AssertError(t, err)
	
	// We can't easily test the failure case without sub-tests
}

func TestAssertEqual(t *testing.T) {
	// Test with equal values of different types
	AssertEqual(t, 42, 42)
	AssertEqual(t, "hello", "hello")
	AssertEqual(t, true, true)
	AssertEqual(t, false, false)
	
	// Test with int64
	AssertEqual(t, int64(42), int64(42))
	
	// Test with float64
	AssertEqual(t, 3.14, 3.14)
	
	// Test with zero values
	AssertEqual(t, 0, 0)
	AssertEqual(t, "", "")
	
	// Test with pointers
	value := 42
	ptr1 := &value
	ptr2 := &value
	AssertEqual(t, ptr1, ptr2) // Same pointer
}

func TestAssertEqual_DifferentTypes(t *testing.T) {
	// Test that the generic function works with different comparable types
	AssertEqual(t, int8(42), int8(42))
	AssertEqual(t, int16(42), int16(42))
	AssertEqual(t, int32(42), int32(42))
	AssertEqual(t, int64(42), int64(42))
	AssertEqual(t, uint(42), uint(42))
	AssertEqual(t, uint8(42), uint8(42))
	AssertEqual(t, uint16(42), uint16(42))
	AssertEqual(t, uint32(42), uint32(42))
	AssertEqual(t, uint64(42), uint64(42))
	AssertEqual(t, float32(3.14), float32(3.14))
	AssertEqual(t, float64(3.14), float64(3.14))
	AssertEqual(t, complex64(1+2i), complex64(1+2i))
	AssertEqual(t, complex128(1+2i), complex128(1+2i))
}

func TestAssertFileExists(t *testing.T) {
	tempDir, cleanup := TempDir(t)
	defer cleanup()
	
	// Create a test file
	testFile := WriteFile(t, tempDir, "test-file.txt", "test content")
	
	// Test with existing file
	AssertFileExists(t, testFile)
	
	// Test with directory (should also pass since directories exist)
	AssertFileExists(t, tempDir)
}

func TestAssertContains(t *testing.T) {
	// Test string that contains substring
	AssertContains(t, "hello world", "world")
	AssertContains(t, "testing", "test")
	AssertContains(t, "abc", "abc") // Full match
	AssertContains(t, "single", "s") // Single character
	AssertContains(t, "case sensitive", "case")
	
	// Test with special characters
	AssertContains(t, "hello\nworld", "\n")
	AssertContains(t, "tab\there", "\t")
	AssertContains(t, "quote\"test", "\"")
	AssertContains(t, "backslash\\test", "\\")
}

// Integration test using multiple utilities together
func TestUtilities_Integration(t *testing.T) {
	// Create a temporary directory
	tempDir, cleanup := TempDir(t)
	defer cleanup()
	
	// Write multiple files
	content1 := "integration test content 1"
	content2 := "integration test content 2"
	
	file1 := WriteFile(t, tempDir, "file1.txt", content1)
	file2 := WriteFile(t, tempDir, "file2.txt", content2)
	
	// Use assertions to verify everything
	AssertFileExists(t, file1)
	AssertFileExists(t, file2)
	AssertFileExists(t, tempDir)
	
	// Read and verify content
	actualContent1, err := os.ReadFile(file1)
	AssertNoError(t, err)
	AssertEqual(t, content1, string(actualContent1))
	AssertContains(t, string(actualContent1), "integration")
	
	actualContent2, err := os.ReadFile(file2)
	AssertNoError(t, err)
	AssertEqual(t, content2, string(actualContent2))
	AssertContains(t, string(actualContent2), "test")
	
	// Verify files are in the same directory
	dir1 := filepath.Dir(file1)
	dir2 := filepath.Dir(file2)
	AssertEqual(t, dir1, dir2)
	AssertEqual(t, tempDir, dir1)
}

func TestUtilities_ErrorScenarios(t *testing.T) {
	// Test scenarios that should work without errors
	tempDir, cleanup := TempDir(t)
	defer cleanup()
	
	// Test writing to a valid directory
	filePath := WriteFile(t, tempDir, "valid-file.txt", "content")
	AssertFileExists(t, filePath)
	
	// Test assertions with valid inputs
	AssertNoError(t, nil)
	AssertEqual(t, "same", "same")
	AssertContains(t, "container", "contain")
}

func TestUtilities_EdgeCases(t *testing.T) {
	tempDir, cleanup := TempDir(t)
	defer cleanup()
	
	// Test with empty content
	emptyFile := WriteFile(t, tempDir, "empty.txt", "")
	AssertFileExists(t, emptyFile)
	
	content, err := os.ReadFile(emptyFile)
	AssertNoError(t, err)
	AssertEqual(t, "", string(content))
	
	// Test with special filename characters
	specialFile := WriteFile(t, tempDir, "file-with_special.chars.txt", "special content")
	AssertFileExists(t, specialFile)
	AssertContains(t, specialFile, "special")
	
	// Test with long content
	longContent := strings.Repeat("long content line\n", 1000)
	longFile := WriteFile(t, tempDir, "long-file.txt", longContent)
	AssertFileExists(t, longFile)
	
	actualLongContent, err := os.ReadFile(longFile)
	AssertNoError(t, err)
	AssertEqual(t, longContent, string(actualLongContent))
	AssertContains(t, string(actualLongContent), "long content line")
}

// Test that helper functions properly use t.Helper()
func TestUtilities_HelperUsage(t *testing.T) {
	// We can't easily test that t.Helper() is called without complex reflection,
	// but we can verify the functions work correctly when called from helper functions
	
	helperFunction := func(t *testing.T) {
		tempDir, cleanup := TempDir(t)
		defer cleanup()
		
		filePath := WriteFile(t, tempDir, "helper-test.txt", "helper content")
		AssertFileExists(t, filePath)
		AssertEqual(t, "helper content", "helper content")
		AssertContains(t, "helper content", "helper")
		AssertNoError(t, nil)
	}
	
	// This should work without issues
	helperFunction(t)
}

func TestUtilities_ConcurrentUsage(t *testing.T) {
	// Test that utilities can be used concurrently
	done := make(chan bool, 5)
	
	for i := 0; i < 5; i++ {
		go func(id int) {
			defer func() { done <- true }()
			
			tempDir, cleanup := TempDir(t)
			defer cleanup()
			
			content := "concurrent content " + string(rune(id+'0'))
			filename := "concurrent-" + string(rune(id+'0')) + ".txt"
			
			filePath := WriteFile(t, tempDir, filename, content)
			AssertFileExists(t, filePath)
			
			actualContent, err := os.ReadFile(filePath)
			AssertNoError(t, err)
			AssertEqual(t, content, string(actualContent))
			AssertContains(t, string(actualContent), "concurrent")
		}(i)
	}
	
	// Wait for all goroutines to complete
	for i := 0; i < 5; i++ {
		<-done
	}
}