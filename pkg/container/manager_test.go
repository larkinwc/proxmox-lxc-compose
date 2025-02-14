package container

import (
	"os"
	"path/filepath"
	"testing"

	"proxmox-lxc-compose/pkg/config"
	"proxmox-lxc-compose/pkg/internal/mock"
	. "proxmox-lxc-compose/pkg/internal/testing"
)

func TestPauseResume(t *testing.T) {
	containerName := "test-container-pause"
	configPath := t.TempDir()

	// Create state directory
	statePath := filepath.Join(configPath, "state")
	err := os.MkdirAll(statePath, 0755)
	AssertNoError(t, err)

	// Set config path for mock command
	os.Setenv("CONTAINER_CONFIG_PATH", configPath)
	defer os.Unsetenv("CONTAINER_CONFIG_PATH")

	mock, cleanup := mock.SetupMockCommand(&execCommand)
	defer cleanup()

	manager, err := NewLXCManager(configPath)
	AssertNoError(t, err)

	t.Run("pause_running_container", func(t *testing.T) {
		// Create container directory and state file first
		containerDir := filepath.Join(configPath, containerName)
		err = os.MkdirAll(containerDir, 0755)
		AssertNoError(t, err)

		// Create container with RUNNING state
		mock.AddContainer(containerName, "RUNNING")

		// Verify initial state
		container, err := manager.Get(containerName)
		AssertNoError(t, err)
		AssertEqual(t, "RUNNING", container.State)

		// Pause container
		err = manager.Pause(containerName)
		AssertNoError(t, err)

		// Verify final state
		container, err = manager.Get(containerName)
		AssertNoError(t, err)
		AssertEqual(t, "FROZEN", container.State)
	})

	t.Run("resume_paused_container", func(t *testing.T) {
		// Create container directory and state file first
		containerDir := filepath.Join(configPath, containerName)
		err = os.MkdirAll(containerDir, 0755)
		AssertNoError(t, err)

		// Create container with FROZEN state
		mock.AddContainer(containerName, "FROZEN")

		// Verify initial state
		container, err := manager.Get(containerName)
		AssertNoError(t, err)
		AssertEqual(t, "FROZEN", container.State)

		// Resume container
		err = manager.Resume(containerName)
		AssertNoError(t, err)

		// Verify final state
		container, err = manager.Get(containerName)
		AssertNoError(t, err)
		AssertEqual(t, "RUNNING", container.State)
	})

	t.Run("pause_non-running_container", func(t *testing.T) {
		// Create container directory
		err = os.MkdirAll(filepath.Join(configPath, containerName), 0755)
		AssertNoError(t, err)

		// Add container and set state to STOPPED
		mock.AddContainer(containerName, "STOPPED")
		err = manager.state.SaveContainerState(containerName, &config.Container{}, "STOPPED")
		AssertNoError(t, err)

		err = manager.Pause(containerName)
		AssertError(t, err)
	})

	t.Run("resume_non-frozen_container", func(t *testing.T) {
		// Create container directory
		err = os.MkdirAll(filepath.Join(configPath, containerName), 0755)
		AssertNoError(t, err)

		// Add container and set state to RUNNING
		mock.AddContainer(containerName, "RUNNING")
		err = manager.state.SaveContainerState(containerName, &config.Container{}, "RUNNING")
		AssertNoError(t, err)

		err = manager.Resume(containerName)
		AssertError(t, err)
	})

	t.Run("pause_non-existent_container", func(t *testing.T) {
		err := manager.Pause("nonexistent")
		AssertError(t, err)
	})

	t.Run("resume_non-existent_container", func(t *testing.T) {
		err := manager.Resume("nonexistent")
		AssertError(t, err)
	})
}

func TestRestart(t *testing.T) {
	containerName := "test-container-restart"
	configPath := t.TempDir()

	// Create state directory
	statePath := filepath.Join(configPath, "state")
	err := os.MkdirAll(statePath, 0755)
	AssertNoError(t, err)

	// Set config path for mock command
	os.Setenv("CONTAINER_CONFIG_PATH", configPath)
	defer os.Unsetenv("CONTAINER_CONFIG_PATH")

	mock, cleanup := mock.SetupMockCommand(&execCommand)
	defer cleanup()

	manager, err := NewLXCManager(configPath)
	AssertNoError(t, err)

	t.Run("restart_running_container", func(t *testing.T) {
		// Create container directory and state file first
		containerDir := filepath.Join(configPath, containerName)
		err = os.MkdirAll(containerDir, 0755)
		AssertNoError(t, err)

		// Create container with RUNNING state
		mock.AddContainer(containerName, "RUNNING")

		// Verify initial state
		container, err := manager.Get(containerName)
		AssertNoError(t, err)
		AssertEqual(t, "RUNNING", container.State)

		// Restart container
		err = manager.Restart(containerName)
		AssertNoError(t, err)

		// Verify final state
		container, err = manager.Get(containerName)
		AssertNoError(t, err)
		AssertEqual(t, "RUNNING", container.State)
	})

	t.Run("restart_stopped_container", func(t *testing.T) {
		// Create container directory
		err = os.MkdirAll(filepath.Join(configPath, containerName), 0755)
		AssertNoError(t, err)

		// Add container and set state to STOPPED
		mock.AddContainer(containerName, "STOPPED")
		err = manager.state.SaveContainerState(containerName, &config.Container{}, "STOPPED")
		AssertNoError(t, err)

		// Verify initial state
		container, err := manager.Get(containerName)
		AssertNoError(t, err)
		AssertEqual(t, "STOPPED", container.State)

		err = manager.Restart(containerName)
		AssertNoError(t, err)

		// Verify state was updated
		container, err = manager.Get(containerName)
		AssertNoError(t, err)
		AssertEqual(t, "RUNNING", container.State)
	})

	t.Run("restart_nonexistent_container", func(t *testing.T) {
		err := manager.Restart("nonexistent")
		AssertError(t, err)
	})
}

func TestUpdate(t *testing.T) {
	containerName := "test-container-update"
	configPath := t.TempDir()

	// Create state directory
	statePath := filepath.Join(configPath, "state")
	err := os.MkdirAll(statePath, 0755)
	AssertNoError(t, err)

	// Set config path for mock command
	os.Setenv("CONTAINER_CONFIG_PATH", configPath)
	defer os.Unsetenv("CONTAINER_CONFIG_PATH")

	mock, cleanup := mock.SetupMockCommand(&execCommand)
	defer cleanup()

	manager, err := NewLXCManager(configPath)
	AssertNoError(t, err)

	t.Run("update_running_container", func(t *testing.T) {
		// Create container directory and state file first
		containerDir := filepath.Join(configPath, containerName)
		err = os.MkdirAll(containerDir, 0755)
		AssertNoError(t, err)

		// Create container with RUNNING state
		mock.AddContainer(containerName, "RUNNING")

		// Verify initial state
		container, err := manager.Get(containerName)
		AssertNoError(t, err)
		AssertEqual(t, "RUNNING", container.State)

		// Update container configuration
		newConfig := &config.Container{
			Network: &config.NetworkConfig{
				Type: "bridge",
			},
		}
		err = manager.Update(containerName, newConfig)
		AssertNoError(t, err)

		// Verify final state
		container, err = manager.Get(containerName)
		AssertNoError(t, err)
		AssertEqual(t, "RUNNING", container.State)
		AssertEqual(t, newConfig, container.Config)
	})

	t.Run("update_stopped_container", func(t *testing.T) {
		// Create container directory
		err = os.MkdirAll(filepath.Join(configPath, containerName), 0755)
		AssertNoError(t, err)

		// Add container and set state to STOPPED
		mock.AddContainer(containerName, "STOPPED")
		err = manager.state.SaveContainerState(containerName, &config.Container{}, "STOPPED")
		AssertNoError(t, err)

		// Verify initial state
		container, err := manager.Get(containerName)
		AssertNoError(t, err)
		AssertEqual(t, "STOPPED", container.State)

		// Update container configuration
		newConfig := &config.Container{
			Network: &config.NetworkConfig{
				Type: "bridge",
			},
		}
		err = manager.Update(containerName, newConfig)
		AssertNoError(t, err)

		// Verify final state
		container, err = manager.Get(containerName)
		AssertNoError(t, err)
		AssertEqual(t, "STOPPED", container.State)
		AssertEqual(t, newConfig, container.Config)
	})

	t.Run("update_nonexistent_container", func(t *testing.T) {
		err := manager.Update("nonexistent", &config.Container{})
		AssertError(t, err)
	})
}
