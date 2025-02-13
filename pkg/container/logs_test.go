package container

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"proxmox-lxc-compose/pkg/config"
	"proxmox-lxc-compose/pkg/testutil"
)

func TestGetLogs(t *testing.T) {
	dir, dirCleanup := testutil.TempDir(t)
	defer dirCleanup()

	// Create test container directory and log file
	containerName := "test-container"
	containerDir := filepath.Join(dir, containerName)
	err := os.MkdirAll(containerDir, 0755)
	testutil.AssertNoError(t, err)

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
	testutil.AssertNoError(t, err)

	// Create state manager
	statePath := filepath.Join(dir, "state")
	stateManager, err := NewStateManager(statePath)
	testutil.AssertNoError(t, err)

	// Create manager
	manager := &LXCManager{
		configPath: dir,
		state:      stateManager,
	}

	// Setup mock command
	oldExecCommand := execCommand
	defer func() { execCommand = oldExecCommand }()
	mock, cleanup := testutil.SetupMockCommand(&execCommand)
	defer cleanup()

	// Ensure container exists and is running
	mock.AddContainer(containerName, "RUNNING")
	err = stateManager.SaveContainerState(containerName, &config.Container{}, "RUNNING")
	testutil.AssertNoError(t, err)

	t.Run("reads all logs", func(t *testing.T) {
		logs, err := manager.GetLogs(containerName, LogOptions{})
		testutil.AssertNoError(t, err)
		defer logs.Close()

		content, err := io.ReadAll(logs)
		testutil.AssertNoError(t, err)

		expected := strings.TrimSpace(strings.Join(logContent, "\n"))
		actual := strings.TrimSpace(string(content))
		testutil.AssertEqual(t, expected, actual)
	})

	t.Run("respects tail option", func(t *testing.T) {
		logs, err := manager.GetLogs(containerName, LogOptions{Tail: 2})
		testutil.AssertNoError(t, err)
		defer logs.Close()

		content, err := io.ReadAll(logs)
		testutil.AssertNoError(t, err)

		// Should only contain the last 2 lines
		expected := strings.TrimSpace(strings.Join(logContent[len(logContent)-2:], "\n"))
		actual := strings.TrimSpace(string(content))
		testutil.AssertEqual(t, expected, actual)
	})

	t.Run("respects since option", func(t *testing.T) {
		logs, err := manager.GetLogs(containerName, LogOptions{
			Since: time.Now().Add(-30 * time.Minute),
		})
		testutil.AssertNoError(t, err)
		defer logs.Close()

		content, err := io.ReadAll(logs)
		testutil.AssertNoError(t, err)

		// Should only contain the last line
		expected := strings.TrimSpace(logContent[2])
		actual := strings.TrimSpace(string(content))
		testutil.AssertEqual(t, expected, actual)
	})

	t.Run("handles non-existent container", func(t *testing.T) {
		_, err := manager.GetLogs("nonexistent", LogOptions{})
		testutil.AssertError(t, err)
	})
}

func TestFollowLogs(t *testing.T) {
	dir, dirCleanup := testutil.TempDir(t)
	defer dirCleanup()

	// Create test container directory
	containerName := "test-container"
	containerDir := filepath.Join(dir, containerName)
	err := os.MkdirAll(containerDir, 0755)
	testutil.AssertNoError(t, err)

	// Create empty log file
	logPath := filepath.Join(containerDir, "console.log")
	err = os.WriteFile(logPath, []byte{}, 0644)
	testutil.AssertNoError(t, err)

	// Create state manager
	statePath := filepath.Join(dir, "state")
	stateManager, err := NewStateManager(statePath)
	testutil.AssertNoError(t, err)

	// Create manager
	manager := &LXCManager{
		configPath: dir,
		state:      stateManager,
	}

	// Setup mock command
	oldExecCommand := execCommand
	defer func() { execCommand = oldExecCommand }()
	mock, cleanup := testutil.SetupMockCommand(&execCommand)
	defer cleanup()

	// Ensure container exists and is running
	mock.AddContainer(containerName, "RUNNING")
	err = stateManager.SaveContainerState(containerName, &config.Container{}, "RUNNING")
	testutil.AssertNoError(t, err)

	t.Run("follows log output", func(t *testing.T) {
		var buf bytes.Buffer
		var mu sync.Mutex
		done := make(chan struct{})

		// Start following logs in background
		go func() {
			err := manager.FollowLogs(containerName, &syncWriter{w: &buf, mu: &mu})
			testutil.AssertNoError(t, err)
			close(done)
		}()

		// Wait for logs to be processed
		time.Sleep(100 * time.Millisecond)

		// Get output with mutex protection
		mu.Lock()
		output := buf.String()
		mu.Unlock()

		// Verify output contains expected lines and timestamps
		lines := strings.Split(strings.TrimSpace(output), "\n")
		for _, line := range lines {
			// Each line should start with a timestamp in brackets
			if !strings.HasPrefix(line, "[") || !strings.Contains(line, "]") {
				t.Errorf("expected line to have timestamp, got: %s", line)
			}
			// Should contain one of our log lines
			if !strings.Contains(line, "New log line 1") && !strings.Contains(line, "New log line 2") {
				t.Errorf("unexpected log line: %s", line)
			}
		}
	})

	t.Run("follows log output without timestamps", func(t *testing.T) {
		var buf bytes.Buffer
		var mu sync.Mutex
		done := make(chan struct{})

		// Start following logs in background with timestamps disabled
		go func() {
			logs, err := manager.GetLogs(containerName, LogOptions{Follow: true, Timestamp: false})
			testutil.AssertNoError(t, err)
			defer logs.Close()
			_, err = io.Copy(&syncWriter{w: &buf, mu: &mu}, logs)
			testutil.AssertNoError(t, err)
			close(done)
		}()

		// Wait for logs to be processed
		time.Sleep(100 * time.Millisecond)

		// Get output with mutex protection
		mu.Lock()
		output := buf.String()
		mu.Unlock()

		// Verify output contains expected lines without timestamps
		lines := strings.Split(strings.TrimSpace(output), "\n")
		for _, line := range lines {
			// Lines should not have timestamp brackets
			if strings.HasPrefix(line, "[") && strings.Contains(line, "]") {
				t.Errorf("expected line without timestamp, got: %s", line)
			}
			// Should contain one of our log lines
			if !strings.Contains(line, "New log line 1") && !strings.Contains(line, "New log line 2") {
				t.Errorf("unexpected log line: %s", line)
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
