// Package testing provides test helper functions
package testing

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// AssertNoError fails the test if err is not nil
func AssertNoError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// AssertError fails the test if err is nil
func AssertError(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		t.Fatal("expected error but got none")
	}
}

// AssertEqual fails the test if a != b
func AssertEqual(t *testing.T, expected, actual interface{}) {
	t.Helper()
	if expected != actual {
		t.Fatalf("expected %v, got %v", expected, actual)
	}
}

// AssertContains fails the test if s does not contain substr
func AssertContains(t *testing.T, s, substr string) {
	t.Helper()
	if !strings.Contains(s, substr) {
		t.Fatalf("expected %q to contain %q", s, substr)
	}
}

// AssertFileExists fails the test if the file does not exist
func AssertFileExists(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected file %s to exist: %v", path, err)
	}
}

// TempDir creates a temporary directory and returns a cleanup function
func TempDir(t *testing.T) (string, func()) {
	t.Helper()
	dir := t.TempDir()
	return dir, func() {
		os.RemoveAll(dir)
	}
}

// WriteFile creates a file with the given content and returns its path
func WriteFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	err := os.WriteFile(path, []byte(content), 0644)
	if err != nil {
		t.Fatalf("failed to write file: %v", err)
	}
	return path
}
