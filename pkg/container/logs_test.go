package container

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"proxmox-lxc-compose/pkg/testutil"
)

func TestGetLogs(t *testing.T) {
	dir, cleanup := testutil.TempDir(t)
	defer cleanup()

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

	// Create manager
	manager := &LXCManager{
		configPath: dir,
	}

	t.Run("reads all logs", func(t *testing.T) {
		_, cleanup := testutil.SetupMockCommand(&execCommand)
		defer cleanup()

		logs, err := manager.GetLogs(containerName, LogOptions{})
		testutil.AssertNoError(t, err)
		defer logs.Close()

		content, err := io.ReadAll(logs)
		testutil.AssertNoError(t, err)

		expected := strings.Join(logContent, "\n") + "\n"
		testutil.AssertEqual(t, expected, string(content))
	})

	t.Run("respects tail option", func(t *testing.T) {
		logs, err := manager.GetLogs(containerName, LogOptions{Tail: 2})
		testutil.AssertNoError(t, err)
		defer logs.Close()

		content, err := io.ReadAll(logs)
		testutil.AssertNoError(t, err)

		expected := strings.Join(logContent[1:], "\n") + "\n"
		testutil.AssertEqual(t, expected, string(content))
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
		expected := logContent[2] + "\n"
		testutil.AssertEqual(t, expected, string(content))
	})

	t.Run("handles non-existent container", func(t *testing.T) {
		_, err := manager.GetLogs("nonexistent", LogOptions{})
		testutil.AssertError(t, err)
	})
}

func TestFollowLogs(t *testing.T) {
	dir, cleanup := testutil.TempDir(t)
	defer cleanup()

	// Create test container directory
	containerName := "test-container"
	containerDir := filepath.Join(dir, containerName)
	err := os.MkdirAll(containerDir, 0755)
	testutil.AssertNoError(t, err)

	// Create empty log file
	logPath := filepath.Join(containerDir, "console.log")
	err = os.WriteFile(logPath, []byte{}, 0644)
	testutil.AssertNoError(t, err)

	// Create manager
	manager := &LXCManager{
		configPath: dir,
	}

	t.Run("follows log output", func(t *testing.T) {
		var buf bytes.Buffer
		done := make(chan struct{})

		// Start following logs in background
		go func() {
			err := manager.FollowLogs(containerName, &buf)
			testutil.AssertNoError(t, err)
			close(done)
		}()

		// Write some logs
		time.Sleep(100 * time.Millisecond)
		f, err := os.OpenFile(logPath, os.O_APPEND|os.O_WRONLY, 0644)
		testutil.AssertNoError(t, err)
		fmt.Fprintln(f, "New log line 1")
		fmt.Fprintln(f, "New log line 2")
		f.Close()

		// Wait for logs to be processed
		time.Sleep(100 * time.Millisecond)

		// Verify output contains timestamps and new lines
		output := buf.String()
		if !strings.Contains(output, "New log line 1") ||
			!strings.Contains(output, "New log line 2") {
			t.Fatal("expected output to contain new log lines")
		}
	})
}
