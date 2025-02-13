package container

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"proxmox-lxc-compose/pkg/config"
	"proxmox-lxc-compose/pkg/testutil"
)

type mockCmd struct {
	name string
	args []string
}

var execCommand = exec.Command

// mockExecCommand replaces exec.Command for testing
func mockExecCommand(mock *mockCmd) func(name string, args ...string) *exec.Cmd {
	return func(name string, args ...string) *exec.Cmd {
		mock.name = name
		mock.args = args
		return exec.Command("echo", "mock") // Just use echo as a dummy command
	}
}

func TestPauseResume(t *testing.T) {
	dir, cleanup := testutil.TempDir(t)
	defer cleanup()

	// Create test container directory
	containerName := "test-container"
	containerDir := filepath.Join(dir, containerName)
	err := os.MkdirAll(containerDir, 0755)
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

	// Save initial state as RUNNING
	err = stateManager.SaveContainerState(containerName, &config.Container{}, "RUNNING")
	testutil.AssertNoError(t, err)

	t.Run("pause running container", func(t *testing.T) {
		mock := &mockCmd{}
		oldExec := execCommand
		execCommand = mockExecCommand(mock)
		defer func() { execCommand = oldExec }()

		err := manager.Pause(containerName)
		testutil.AssertNoError(t, err)

		// Verify lxc-freeze was called correctly
		testutil.AssertEqual(t, "lxc-freeze", mock.name)
		if len(mock.args) != 2 || mock.args[0] != "-n" || mock.args[1] != containerName {
			t.Fatalf("unexpected args: %v", mock.args)
		}

		// Verify state was updated
		state, err := stateManager.GetContainerState(containerName)
		testutil.AssertNoError(t, err)
		testutil.AssertEqual(t, "FROZEN", state.Status)
	})

	t.Run("resume paused container", func(t *testing.T) {
		// Set state to FROZEN first
		err := stateManager.SaveContainerState(containerName, &config.Container{}, "FROZEN")
		testutil.AssertNoError(t, err)

		mock := &mockCmd{}
		oldExec := execCommand
		execCommand = mockExecCommand(mock)
		defer func() { execCommand = oldExec }()

		err = manager.Resume(containerName)
		testutil.AssertNoError(t, err)

		// Verify lxc-unfreeze was called correctly
		testutil.AssertEqual(t, "lxc-unfreeze", mock.name)
		if len(mock.args) != 2 || mock.args[0] != "-n" || mock.args[1] != containerName {
			t.Fatalf("unexpected args: %v", mock.args)
		}

		// Verify state was updated
		state, err := stateManager.GetContainerState(containerName)
		testutil.AssertNoError(t, err)
		testutil.AssertEqual(t, "RUNNING", state.Status)
	})

	t.Run("pause non-running container", func(t *testing.T) {
		// Set state to STOPPED
		err := stateManager.SaveContainerState(containerName, &config.Container{}, "STOPPED")
		testutil.AssertNoError(t, err)

		err = manager.Pause(containerName)
		testutil.AssertError(t, err)
	})

	t.Run("resume non-frozen container", func(t *testing.T) {
		// Set state to RUNNING
		err := stateManager.SaveContainerState(containerName, &config.Container{}, "RUNNING")
		testutil.AssertNoError(t, err)

		err = manager.Resume(containerName)
		testutil.AssertError(t, err)
	})

	t.Run("pause non-existent container", func(t *testing.T) {
		err := manager.Pause("nonexistent")
		testutil.AssertError(t, err)
	})

	t.Run("resume non-existent container", func(t *testing.T) {
		err := manager.Resume("nonexistent")
		testutil.AssertError(t, err)
	})
}
