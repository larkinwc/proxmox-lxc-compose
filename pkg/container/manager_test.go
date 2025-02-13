package container

import (
	"os"
	"path/filepath"
	"testing"

	"proxmox-lxc-compose/pkg/config"
	"proxmox-lxc-compose/pkg/testutil"
)

func TestPauseResume(t *testing.T) {
	containerName := "test-container-pause"
	configPath := t.TempDir()

	// Create state directory
	statePath := filepath.Join(configPath, "state")
	err := os.MkdirAll(statePath, 0755)
	testutil.AssertNoError(t, err)

	// Create container directory
	containerPath := filepath.Join(configPath, containerName)
	err = os.MkdirAll(containerPath, 0755)
	testutil.AssertNoError(t, err)

	mock, cleanup := testutil.SetupMockCommand(&execCommand)
	defer cleanup()

	manager, err := NewLXCManager(configPath)
	testutil.AssertNoError(t, err)

	t.Run("pause_running_container", func(t *testing.T) {
		// Create container directory
		err = os.MkdirAll(filepath.Join(configPath, containerName), 0755)
		testutil.AssertNoError(t, err)

		// Add container and set state to RUNNING
		mock.AddContainer(containerName, "RUNNING")
		err = manager.state.SaveContainerState(containerName, &config.Container{}, "RUNNING")
		testutil.AssertNoError(t, err)

		// Verify initial state
		container, err := manager.Get(containerName)
		testutil.AssertNoError(t, err)
		testutil.AssertEqual(t, "RUNNING", container.State)

		err = manager.Pause(containerName)
		testutil.AssertNoError(t, err)

		// Verify lxc-freeze was called correctly
		testutil.AssertEqual(t, "lxc-freeze", mock.Name)
		if len(mock.Args) != 2 || mock.Args[0] != "-n" || mock.Args[1] != containerName {
			t.Fatalf("unexpected args: %v", mock.Args)
		}

		// Verify state was updated
		container, err = manager.Get(containerName)
		testutil.AssertNoError(t, err)
		testutil.AssertEqual(t, "FROZEN", container.State)
	})

	t.Run("resume_paused_container", func(t *testing.T) {
		// Create container directory
		err = os.MkdirAll(filepath.Join(configPath, containerName), 0755)
		testutil.AssertNoError(t, err)

		// Add container and set state to FROZEN
		mock.AddContainer(containerName, "FROZEN")
		err = manager.state.SaveContainerState(containerName, &config.Container{}, "FROZEN")
		testutil.AssertNoError(t, err)

		// Verify initial state
		container, err := manager.Get(containerName)
		testutil.AssertNoError(t, err)
		testutil.AssertEqual(t, "FROZEN", container.State)

		err = manager.Resume(containerName)
		testutil.AssertNoError(t, err)

		// Verify lxc-unfreeze was called correctly
		testutil.AssertEqual(t, "lxc-unfreeze", mock.Name)
		if len(mock.Args) != 2 || mock.Args[0] != "-n" || mock.Args[1] != containerName {
			t.Fatalf("unexpected args: %v", mock.Args)
		}

		// Verify state was updated
		container, err = manager.Get(containerName)
		testutil.AssertNoError(t, err)
		testutil.AssertEqual(t, "RUNNING", container.State)
	})

	t.Run("pause_non-running_container", func(t *testing.T) {
		// Add container and set state to STOPPED
		mock.AddContainer(containerName, "STOPPED")
		err = manager.state.SaveContainerState(containerName, &config.Container{}, "STOPPED")
		testutil.AssertNoError(t, err)

		err = manager.Pause(containerName)
		testutil.AssertError(t, err)
	})

	t.Run("resume_non-frozen_container", func(t *testing.T) {
		// Add container and set state to RUNNING
		mock.AddContainer(containerName, "RUNNING")
		err = manager.state.SaveContainerState(containerName, &config.Container{}, "RUNNING")
		testutil.AssertNoError(t, err)

		err = manager.Resume(containerName)
		testutil.AssertError(t, err)
	})

	t.Run("pause_non-existent_container", func(t *testing.T) {
		err := manager.Pause("nonexistent")
		testutil.AssertError(t, err)
	})

	t.Run("resume_non-existent_container", func(t *testing.T) {
		err := manager.Resume("nonexistent")
		testutil.AssertError(t, err)
	})
}

func TestRestart(t *testing.T) {
	containerName := "test-container-restart"
	configPath := t.TempDir()

	// Create state directory
	statePath := filepath.Join(configPath, "state")
	err := os.MkdirAll(statePath, 0755)
	testutil.AssertNoError(t, err)

	// Create container directory
	containerPath := filepath.Join(configPath, containerName)
	err = os.MkdirAll(containerPath, 0755)
	testutil.AssertNoError(t, err)

	mock, cleanup := testutil.SetupMockCommand(&execCommand)
	defer cleanup()

	manager, err := NewLXCManager(configPath)
	testutil.AssertNoError(t, err)

	t.Run("restart_running_container", func(t *testing.T) {
		// Create container directory
		err = os.MkdirAll(filepath.Join(configPath, containerName), 0755)
		testutil.AssertNoError(t, err)

		// Add container and set state to RUNNING
		mock.AddContainer(containerName, "RUNNING")
		err = manager.state.SaveContainerState(containerName, &config.Container{}, "RUNNING")
		testutil.AssertNoError(t, err)

		// Verify initial state
		container, err := manager.Get(containerName)
		testutil.AssertNoError(t, err)
		testutil.AssertEqual(t, "RUNNING", container.State)

		err = manager.Restart(containerName)
		testutil.AssertNoError(t, err)

		// Verify state was updated
		container, err = manager.Get(containerName)
		testutil.AssertNoError(t, err)
		testutil.AssertEqual(t, "RUNNING", container.State)
	})

	t.Run("restart_stopped_container", func(t *testing.T) {
		// Create container directory
		err = os.MkdirAll(filepath.Join(configPath, containerName), 0755)
		testutil.AssertNoError(t, err)

		// Add container and set state to STOPPED
		mock.AddContainer(containerName, "STOPPED")
		err = manager.state.SaveContainerState(containerName, &config.Container{}, "STOPPED")
		testutil.AssertNoError(t, err)

		// Verify initial state
		container, err := manager.Get(containerName)
		testutil.AssertNoError(t, err)
		testutil.AssertEqual(t, "STOPPED", container.State)

		err = manager.Restart(containerName)
		testutil.AssertNoError(t, err)

		// Verify state was updated
		container, err = manager.Get(containerName)
		testutil.AssertNoError(t, err)
		testutil.AssertEqual(t, "RUNNING", container.State)
	})

	t.Run("restart_nonexistent_container", func(t *testing.T) {
		err := manager.Restart("nonexistent")
		testutil.AssertError(t, err)
	})
}

func TestUpdate(t *testing.T) {
	containerName := "test-container-update"
	configPath := t.TempDir()

	// Create state directory
	statePath := filepath.Join(configPath, "state")
	err := os.MkdirAll(statePath, 0755)
	testutil.AssertNoError(t, err)

	// Create container directory
	containerPath := filepath.Join(configPath, containerName)
	err = os.MkdirAll(containerPath, 0755)
	testutil.AssertNoError(t, err)

	mock, cleanup := testutil.SetupMockCommand(&execCommand)
	defer cleanup()

	manager, err := NewLXCManager(configPath)
	testutil.AssertNoError(t, err)

	t.Run("update_running_container", func(t *testing.T) {
		// Create container directory
		err = os.MkdirAll(filepath.Join(configPath, containerName), 0755)
		testutil.AssertNoError(t, err)

		// Add container and set state to RUNNING
		mock.AddContainer(containerName, "RUNNING")
		err = manager.state.SaveContainerState(containerName, &config.Container{}, "RUNNING")
		testutil.AssertNoError(t, err)

		// Verify initial state
		container, err := manager.Get(containerName)
		testutil.AssertNoError(t, err)
		testutil.AssertEqual(t, "RUNNING", container.State)

		// Update container configuration
		newConfig := &config.Container{
			Network: &config.NetworkConfig{
				Type: "bridge",
			},
		}
		err = manager.Update(containerName, newConfig)
		testutil.AssertNoError(t, err)

		// Verify state was updated
		container, err = manager.Get(containerName)
		testutil.AssertNoError(t, err)
		testutil.AssertEqual(t, "RUNNING", container.State)
		testutil.AssertEqual(t, newConfig, container.Config)
	})

	t.Run("update_stopped_container", func(t *testing.T) {
		// Create container directory
		err = os.MkdirAll(filepath.Join(configPath, containerName), 0755)
		testutil.AssertNoError(t, err)

		// Add container and set state to STOPPED
		mock.AddContainer(containerName, "STOPPED")
		err = manager.state.SaveContainerState(containerName, &config.Container{}, "STOPPED")
		testutil.AssertNoError(t, err)

		// Verify initial state
		container, err := manager.Get(containerName)
		testutil.AssertNoError(t, err)
		testutil.AssertEqual(t, "STOPPED", container.State)

		// Update container configuration
		newConfig := &config.Container{
			Network: &config.NetworkConfig{
				Type: "bridge",
			},
		}
		err = manager.Update(containerName, newConfig)
		testutil.AssertNoError(t, err)

		// Verify state was updated
		container, err = manager.Get(containerName)
		testutil.AssertNoError(t, err)
		testutil.AssertEqual(t, "STOPPED", container.State)
		testutil.AssertEqual(t, newConfig, container.Config)
	})

	t.Run("update_nonexistent_container", func(t *testing.T) {
		err := manager.Update("nonexistent", &config.Container{})
		testutil.AssertError(t, err)
	})
}
