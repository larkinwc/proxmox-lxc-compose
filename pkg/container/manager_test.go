package container

import (
	"os"
	"path/filepath"
	"testing"

	"proxmox-lxc-compose/pkg/config"
	"proxmox-lxc-compose/pkg/testutil"
)

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

	// Setup mock command
	oldExecCommand := execCommand
	defer func() { execCommand = oldExecCommand }()
	mock, cleanup := testutil.SetupMockCommand(&execCommand)
	defer cleanup()

	// Ensure container exists and is running
	mock.AddContainer(containerName, "RUNNING")

	t.Run("pause running container", func(t *testing.T) {
		err := manager.Pause(containerName)
		testutil.AssertNoError(t, err)

		// Verify lxc-freeze was called correctly
		testutil.AssertEqual(t, "lxc-freeze", mock.Name)
		if len(mock.Args) != 2 || mock.Args[0] != "-n" || mock.Args[1] != containerName {
			t.Fatalf("unexpected args: %v", mock.Args)
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
		mock.SetContainerState(containerName, "FROZEN")

		err = manager.Resume(containerName)
		testutil.AssertNoError(t, err)

		// Verify lxc-unfreeze was called correctly
		testutil.AssertEqual(t, "lxc-unfreeze", mock.Name)
		if len(mock.Args) != 2 || mock.Args[0] != "-n" || mock.Args[1] != containerName {
			t.Fatalf("unexpected args: %v", mock.Args)
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
		mock.SetContainerState(containerName, "STOPPED")

		err = manager.Pause(containerName)
		testutil.AssertError(t, err)
	})

	t.Run("resume non-frozen container", func(t *testing.T) {
		// Set state to RUNNING
		err := stateManager.SaveContainerState(containerName, &config.Container{}, "RUNNING")
		testutil.AssertNoError(t, err)
		mock.SetContainerState(containerName, "RUNNING")

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
