package container

import (
	"os"
	"path/filepath"
	"testing"
	"time"

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

func TestStartStop(t *testing.T) {
	containerName := "test-container-startstop"
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

	t.Run("start_stopped_container", func(t *testing.T) {
		// Create container directory and initialize state
		containerDir := filepath.Join(configPath, containerName)
		err = os.MkdirAll(containerDir, 0755)
		AssertNoError(t, err)

		// Enable debug logging
		mock.SetDebug(true)

		mock.AddContainer(containerName, "STOPPED")
		err = manager.state.SaveContainerState(containerName, &config.Container{}, "STOPPED")
		AssertNoError(t, err)

		// Verify initial state
		container, err := manager.Get(containerName)
		AssertNoError(t, err)
		AssertEqual(t, "STOPPED", container.State)

		// Add a small delay to ensure state is properly saved
		time.Sleep(100 * time.Millisecond)

		// Start container
		err = manager.Start(containerName)
		AssertNoError(t, err)

		// Verify final state
		container, err = manager.Get(containerName)
		AssertNoError(t, err)
		AssertEqual(t, "RUNNING", container.State)
	})

	t.Run("start_running_container", func(t *testing.T) {
		mock.AddContainer(containerName, "RUNNING")
		err = manager.state.SaveContainerState(containerName, &config.Container{}, "RUNNING")
		AssertNoError(t, err)

		// Verify initial state
		container, err := manager.Get(containerName)
		AssertNoError(t, err)
		AssertEqual(t, "RUNNING", container.State)

		// Try to start an already running container
		err = manager.Start(containerName)
		AssertError(t, err)

		// Verify state hasn't changed
		container, err = manager.Get(containerName)
		AssertNoError(t, err)
		AssertEqual(t, "RUNNING", container.State)
	})

	t.Run("start_frozen_container", func(t *testing.T) {
		mock.AddContainer(containerName, "FROZEN")
		err = manager.state.SaveContainerState(containerName, &config.Container{}, "FROZEN")
		AssertNoError(t, err)

		// Verify initial state
		container, err := manager.Get(containerName)
		AssertNoError(t, err)
		AssertEqual(t, "FROZEN", container.State)

		// Try to start a frozen container
		err = manager.Start(containerName)
		AssertError(t, err)

		// Verify state hasn't changed
		container, err = manager.Get(containerName)
		AssertNoError(t, err)
		AssertEqual(t, "FROZEN", container.State)
	})

	t.Run("stop_running_container", func(t *testing.T) {
		mock.AddContainer(containerName, "RUNNING")
		err = manager.state.SaveContainerState(containerName, &config.Container{}, "RUNNING")
		AssertNoError(t, err)

		// Verify initial state
		container, err := manager.Get(containerName)
		AssertNoError(t, err)
		AssertEqual(t, "RUNNING", container.State)

		// Stop container
		err = manager.Stop(containerName)
		AssertNoError(t, err)

		// Verify final state
		container, err = manager.Get(containerName)
		AssertNoError(t, err)
		AssertEqual(t, "STOPPED", container.State)
	})

	t.Run("stop_stopped_container", func(t *testing.T) {
		mock.AddContainer(containerName, "STOPPED")
		err = manager.state.SaveContainerState(containerName, &config.Container{}, "STOPPED")
		AssertNoError(t, err)

		// Verify initial state
		container, err := manager.Get(containerName)
		AssertNoError(t, err)
		AssertEqual(t, "STOPPED", container.State)

		// Try to stop an already stopped container
		err = manager.Stop(containerName)
		AssertError(t, err)

		// Verify state hasn't changed
		container, err = manager.Get(containerName)
		AssertNoError(t, err)
		AssertEqual(t, "STOPPED", container.State)
	})

	t.Run("stop_nonexistent_container", func(t *testing.T) {
		err := manager.Stop("nonexistent")
		AssertError(t, err)
	})

	t.Run("start_nonexistent_container", func(t *testing.T) {
		err := manager.Start("nonexistent")
		AssertError(t, err)
	})
}

func TestCreateRemove(t *testing.T) {
	containerName := "test-container-create"
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

	t.Run("create_new_container", func(t *testing.T) {
		cfg := &config.Container{
			Image: "ubuntu:20.04",
			Network: &config.NetworkConfig{
				Type: "bridge",
			},
		}

		// Create container
		err := manager.Create(containerName, cfg)
		AssertNoError(t, err)

		// Verify container exists
		exists := manager.ContainerExists(containerName)
		AssertEqual(t, true, exists)

		// Verify container state
		container, err := manager.Get(containerName)
		AssertNoError(t, err)
		AssertEqual(t, "STOPPED", container.State)
		AssertEqual(t, cfg, container.Config)
	})

	t.Run("create_existing_container", func(t *testing.T) {
		cfg := &config.Container{
			Image: "ubuntu:20.04",
		}

		// Try to create container with same name
		err := manager.Create(containerName, cfg)
		AssertError(t, err)
	})

	t.Run("remove_stopped_container", func(t *testing.T) {
		// Verify container exists and is stopped
		container, err := manager.Get(containerName)
		AssertNoError(t, err)
		AssertEqual(t, "STOPPED", container.State)

		// Remove container
		err = manager.Remove(containerName)
		AssertNoError(t, err)

		// Verify container no longer exists
		exists := manager.ContainerExists(containerName)
		AssertEqual(t, false, exists)

		// Verify state is removed
		_, err = manager.state.GetContainerState(containerName)
		AssertError(t, err)
	})

	t.Run("remove_running_container", func(t *testing.T) {
		// Create new container
		cfg := &config.Container{Image: "ubuntu:20.04"}
		err := manager.Create(containerName, cfg)
		AssertNoError(t, err)

		// Set container state to running
		mock.AddContainer(containerName, "RUNNING")
		err = manager.state.SaveContainerState(containerName, cfg, "RUNNING")
		AssertNoError(t, err)

		// Try to remove running container
		err = manager.Remove(containerName)
		AssertError(t, err)

		// Verify container still exists
		exists := manager.ContainerExists(containerName)
		AssertEqual(t, true, exists)
	})

	t.Run("remove_nonexistent_container", func(t *testing.T) {
		err := manager.Remove("nonexistent")
		AssertError(t, err)
	})
}
