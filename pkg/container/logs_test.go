package container_test

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"proxmox-lxc-compose/pkg/config"
	"proxmox-lxc-compose/pkg/container"
	"proxmox-lxc-compose/pkg/internal/mock"
	testing_internal "proxmox-lxc-compose/pkg/internal/testing"
)

// Mock exec.Command for testing
var execCommand = func(name string, args ...string) *exec.Cmd {
	return exec.Command(name, args...)
}

func TestGetLogs(t *testing.T) {
	dir, dirCleanup := testing_internal.TempDir(t)
	defer dirCleanup()

	// Create test container directory and log file
	containerName := "test-container"
	containerDir := filepath.Join(dir, containerName)
	err := os.MkdirAll(containerDir, 0755)
	testing_internal.AssertNoError(t, err)

	// Create test log file with timestamps
	logContent := []string{
		fmt.Sprintf("[%s] Line 1", time.Now().Add(-2*time.Hour).Format(time.RFC3339)),
		fmt.Sprintf("[%s] Line 2", time.Now().Add(-1*time.Hour).Format(time.RFC3339)),
		fmt.Sprintf("[%s] Line 3", time.Now().Format(time.RFC3339)),
	}
	err = os.WriteFile(
		filepath.Join(containerDir, "console.log"),
		[]byte(strings.Join(logContent, "\n")),
		0644,
	)
	testing_internal.AssertNoError(t, err)

	// Create manager
	manager, err := container.NewLXCManager(dir)
	testing_internal.AssertNoError(t, err)

	// Setup mock command
	oldExecCommand := execCommand
	defer func() { execCommand = oldExecCommand }()
	mock, cleanup := mock.SetupMockCommand(&execCommand)
	defer cleanup()

	// Ensure container exists and is running
	mock.AddContainer(containerName, "RUNNING")

	// Create container with initial state
	err = manager.Create(containerName, &config.Container{})
	testing_internal.AssertNoError(t, err)

	t.Run("reads all logs", func(t *testing.T) {
		logs, err := manager.GetLogs(containerName, container.LogOptions{})
		testing_internal.AssertNoError(t, err)
		defer logs.Close()

		content, err := io.ReadAll(logs)
		testing_internal.AssertNoError(t, err)

		expected := strings.TrimSpace(strings.Join(logContent, "\n"))
		actual := strings.TrimSpace(string(content))
		testing_internal.AssertEqual(t, expected, actual)
	})

	t.Run("respects tail option", func(t *testing.T) {
		logs, err := manager.GetLogs(containerName, container.LogOptions{Tail: 2})
		testing_internal.AssertNoError(t, err)
		defer logs.Close()

		content, err := io.ReadAll(logs)
		testing_internal.AssertNoError(t, err)

		// Should only contain the last 2 lines
		expected := strings.TrimSpace(strings.Join(logContent[len(logContent)-2:], "\n"))
		actual := strings.TrimSpace(string(content))
		testing_internal.AssertEqual(t, expected, actual)
	})

	t.Run("respects since option", func(t *testing.T) {
		logs, err := manager.GetLogs(containerName, container.LogOptions{
			Since: time.Now().Add(-30 * time.Minute),
		})
		testing_internal.AssertNoError(t, err)
		defer logs.Close()

		content, err := io.ReadAll(logs)
		testing_internal.AssertNoError(t, err)

		// Should only contain the last line
		expected := strings.TrimSpace(logContent[2])
		actual := strings.TrimSpace(string(content))
		testing_internal.AssertEqual(t, expected, actual)
	})

	t.Run("handles non-existent container", func(t *testing.T) {
		_, err := manager.GetLogs("nonexistent", container.LogOptions{})
		testing_internal.AssertError(t, err)
	})
}

func TestFollowLogs(t *testing.T) {
	dir, dirCleanup := testing_internal.TempDir(t)
	defer dirCleanup()

	// Create test container directory
	containerName := "test-container"
	containerDir := filepath.Join(dir, containerName)
	err := os.MkdirAll(containerDir, 0755)
	testing_internal.AssertNoError(t, err)

	// Create empty log file
	logPath := filepath.Join(containerDir, "console.log")
	err = os.WriteFile(logPath, []byte{}, 0644)
	testing_internal.AssertNoError(t, err)

	// Create manager
	manager, err := container.NewLXCManager(dir)
	testing_internal.AssertNoError(t, err)

	// Setup mock command
	oldExecCommand := execCommand
	defer func() { execCommand = oldExecCommand }()
	mock, cleanup := mock.SetupMockCommand(&execCommand)
	defer cleanup()

	// Ensure container exists and is running
	mock.AddContainer(containerName, "RUNNING")
	err = manager.Create(containerName, &config.Container{})
	testing_internal.AssertNoError(t, err)

	t.Run("follows log output", func(t *testing.T) {
		var buf bytes.Buffer
		var mu sync.Mutex
		done := make(chan struct{})

		// Start following logs in background
		go func() {
			logs, err := manager.GetLogs(containerName, container.LogOptions{Follow: true})
			testing_internal.AssertNoError(t, err)
			defer logs.Close()
			_, err = io.Copy(&syncWriter{w: &buf, mu: &mu}, logs)
			testing_internal.AssertNoError(t, err)
			close(done)
		}()

		// Wait for logs to be processed
		time.Sleep(300 * time.Millisecond)

		// Get output with mutex protection
		mu.Lock()
		output := buf.String()
		mu.Unlock()

		// Verify output contains expected lines
		lines := strings.Split(strings.TrimSpace(output), "\n")
		expectedMessages := []string{
			"Container started",
			"Service initialized",
			"Ready to accept connections",
		}

		if len(lines) < len(expectedMessages) {
			t.Errorf("expected at least %d lines, got %d", len(expectedMessages), len(lines))
		}

		for i, msg := range expectedMessages {
			if i >= len(lines) {
				break
			}
			if !strings.Contains(lines[i], msg) {
				t.Errorf("line %d: expected to contain %q, got: %q", i, msg, lines[i])
			}
		}
	})

	t.Run("follows log output without timestamps", func(t *testing.T) {
		var buf bytes.Buffer
		var mu sync.Mutex
		done := make(chan struct{})

		// Start following logs in background with timestamps disabled
		go func() {
			logs, err := manager.GetLogs(containerName, container.LogOptions{Follow: true, Timestamp: false})
			testing_internal.AssertNoError(t, err)
			defer logs.Close()
			_, err = io.Copy(&syncWriter{w: &buf, mu: &mu}, logs)
			testing_internal.AssertNoError(t, err)
			close(done)
		}()

		// Wait for logs to be processed
		time.Sleep(300 * time.Millisecond)

		// Get output with mutex protection
		mu.Lock()
		output := buf.String()
		mu.Unlock()

		// Verify output contains expected lines without timestamps
		lines := strings.Split(strings.TrimSpace(output), "\n")
		expectedMessages := []string{
			"Container started",
			"Service initialized",
			"Ready to accept connections",
		}

		if len(lines) < len(expectedMessages) {
			t.Errorf("expected at least %d lines, got %d", len(expectedMessages), len(lines))
		}

		for i, msg := range expectedMessages {
			if i >= len(lines) {
				break
			}
			if !strings.Contains(lines[i], msg) {
				t.Errorf("line %d: expected to contain %q, got: %q", i, msg, lines[i])
			}
		}
	})
}

// syncWriter wraps an io.Writer with mutex protection
type syncWriter struct {
	w  io.Writer
	mu *sync.Mutex
}

func (sw *syncWriter) Write(p []byte) (n int, err error) {
	sw.mu.Lock()
	defer sw.mu.Unlock()
	return sw.w.Write(p)
}
