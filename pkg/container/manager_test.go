package container_test

import (
	"os"
	"path/filepath"

	"github.com/larkinwc/proxmox-lxc-compose/pkg/config"
	"github.com/larkinwc/proxmox-lxc-compose/pkg/container"
	"github.com/larkinwc/proxmox-lxc-compose/pkg/internal/mock"
	testing_internal "github.com/larkinwc/proxmox-lxc-compose/pkg/internal/testing"
	"github.com/larkinwc/proxmox-lxc-compose/pkg/logging"
	"testing"
)

func init() {
	// Initialize logger for tests
	if err := logging.Init(logging.Config{
		Level:       "debug",
		Development: true,
	}); err != nil {
		panic("Failed to initialize logger for tests: " + err.Error())
	}
}

// TestBasicContainerLifecycle tests the basic container lifecycle operations
// without relying on CLI commands
func TestBasicContainerLifecycle(t *testing.T) {
	containerName := "test-container-lifecycle"
	configPath := t.TempDir()

	// Create state directory
	statePath := filepath.Join(configPath, "state")
	err := os.MkdirAll(statePath, 0755)
	testing_internal.AssertNoError(t, err)

	// Set CONTAINER_CONFIG_PATH for mock
	os.Setenv("CONTAINER_CONFIG_PATH", configPath)
	defer os.Unsetenv("CONTAINER_CONFIG_PATH")

	mockCmd, cleanup := mock.SetupMockCommand(&container.ExecCommand)
	defer cleanup()

	// Enable debug mode for mock
	mockCmd.SetDebug(true)

	manager, err := container.NewLXCManager(configPath)
	testing_internal.AssertNoError(t, err)

	// Test container creation
	t.Run("create_container", func(t *testing.T) {
		cfg := &config.Container{
			Image: "ubuntu:20.04",
			Network: &config.NetworkConfig{
				Type: "bridge",
			},
		}
		commonCfg := cfg.ToCommonContainer()

		// Ensure container doesn't exist in mock state
		mockCmd.RemoveContainer(containerName)

		err := manager.Create(containerName, commonCfg)
		testing_internal.AssertNoError(t, err)

		// Add container to mock state after creation
		err = mockCmd.AddContainer(containerName, "STOPPED")
		testing_internal.AssertNoError(t, err)

		exists := manager.ContainerExists(containerName)
		testing_internal.AssertEqual(t, true, exists)
	})

	// Test container state transitions
	t.Run("state_transitions", func(t *testing.T) {
		// Test start
		err = manager.Start(containerName)
		testing_internal.AssertNoError(t, err)

		err = mockCmd.SetContainerState(containerName, "RUNNING")
		testing_internal.AssertNoError(t, err)

		container, err := manager.Get(containerName)
		testing_internal.AssertNoError(t, err)
		testing_internal.AssertEqual(t, "RUNNING", container.State)

		// Test pause
		err = manager.Pause(containerName)
		testing_internal.AssertNoError(t, err)

		err = mockCmd.SetContainerState(containerName, "FROZEN")
		testing_internal.AssertNoError(t, err)

		container, err = manager.Get(containerName)
		testing_internal.AssertNoError(t, err)
		testing_internal.AssertEqual(t, "FROZEN", container.State)

		// Test resume
		err = manager.Resume(containerName)
		testing_internal.AssertNoError(t, err)

		err = mockCmd.SetContainerState(containerName, "RUNNING")
		testing_internal.AssertNoError(t, err)

		container, err = manager.Get(containerName)
		testing_internal.AssertNoError(t, err)
		testing_internal.AssertEqual(t, "RUNNING", container.State)

		// Test stop
		err = manager.Stop(containerName)
		testing_internal.AssertNoError(t, err)

		err = mockCmd.SetContainerState(containerName, "STOPPED")
		testing_internal.AssertNoError(t, err)

		container, err = manager.Get(containerName)
		testing_internal.AssertNoError(t, err)
		testing_internal.AssertEqual(t, "STOPPED", container.State)
	})

	// Test container removal
	t.Run("remove_container", func(t *testing.T) {
		err = mockCmd.SetContainerState(containerName, "STOPPED")
		testing_internal.AssertNoError(t, err)

		err = manager.Remove(containerName)
		testing_internal.AssertNoError(t, err)

		err = mockCmd.RemoveContainer(containerName)
		testing_internal.AssertNoError(t, err)

		exists := manager.ContainerExists(containerName)
		testing_internal.AssertEqual(t, false, exists)
	})
}

// Move the following tests to integration_test.go when ready:
// TestPauseResume
// TestRestart
// TestUpdate
// TestStartStop
// TestCreateRemove
