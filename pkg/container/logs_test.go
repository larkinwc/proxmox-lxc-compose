package container_test

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/larkinwc/proxmox-lxc-compose/pkg/common"
	"github.com/larkinwc/proxmox-lxc-compose/pkg/container"
	"github.com/larkinwc/proxmox-lxc-compose/pkg/internal/mock"
	testing_internal "github.com/larkinwc/proxmox-lxc-compose/pkg/internal/testing"
)

var execCommand = exec.Command

// TestLogRetrieval tests the log retrieval functionality without CLI dependencies
func TestLogRetrieval(t *testing.T) {
	dir, dirCleanup := testing_internal.TempDir(t)
	defer dirCleanup()

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

	manager, err := container.NewLXCManager(dir)
	testing_internal.AssertNoError(t, err)

	mock, cleanup := mock.SetupMockCommand(&execCommand)
	defer cleanup()

	// Create container with initial state
	mock.AddContainer(containerName, "RUNNING")
	err = manager.Create(containerName, &common.Container{})
	testing_internal.AssertNoError(t, err)

	t.Run("reads_all_logs", func(t *testing.T) {
		logs, err := manager.GetLogs(containerName, container.LogOptions{})
		testing_internal.AssertNoError(t, err)
		defer logs.Close()

		content, err := io.ReadAll(logs)
		testing_internal.AssertNoError(t, err)
		testing_internal.AssertEqual(t, strings.Join(logContent, "\n"), string(content))
	})

	t.Run("reads_logs_with_tail", func(t *testing.T) {
		logs, err := manager.GetLogs(containerName, container.LogOptions{
			Tail: 2,
		})
		testing_internal.AssertNoError(t, err)
		defer logs.Close()

		content, err := io.ReadAll(logs)
		testing_internal.AssertNoError(t, err)
		expected := strings.Join(logContent[len(logContent)-2:], "\n")
		testing_internal.AssertEqual(t, expected, string(content))
	})

	t.Run("reads_logs_since_timestamp", func(t *testing.T) {
		since := time.Now().Add(-90 * time.Minute)
		logs, err := manager.GetLogs(containerName, container.LogOptions{
			Since: since,
		})
		testing_internal.AssertNoError(t, err)
		defer logs.Close()

		content, err := io.ReadAll(logs)
		testing_internal.AssertNoError(t, err)
		testing_internal.AssertContains(t, string(content), "Line 2")
		testing_internal.AssertContains(t, string(content), "Line 3")
		testing_internal.AssertNotContains(t, string(content), "Line 1")
	})
}

// Move TestFollowLogs to integration_test.go when ready
// It requires actual log streaming which is better suited for integration tests
