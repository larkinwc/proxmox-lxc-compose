package testutil

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TempDir creates a temporary directory for testing and returns a cleanup function
func TempDir(t *testing.T) (string, func()) {
	t.Helper()
	dir, err := os.MkdirTemp("", "lxc-compose-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	return dir, func() { os.RemoveAll(dir) }
}

// WriteFile writes content to a file in the given directory
func WriteFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}
	return path
}

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
		t.Fatal("expected error, got nil")
	}
}

// AssertEqual fails the test if expected != actual
func AssertEqual[T comparable](t *testing.T, expected, actual T) {
	t.Helper()
	if expected != actual {
		t.Fatalf("expected %v, got %v", expected, actual)
	}
}

// AssertFileExists fails the test if the file doesn't exist
func AssertFileExists(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("file %s does not exist: %v", path, err)
	}
}

// AssertContains fails the test if the string does not contain the substring
func AssertContains(t *testing.T, s, substr string) {
	t.Helper()
	if !strings.Contains(s, substr) {
		t.Errorf("expected %q to contain %q", s, substr)
	}
}
