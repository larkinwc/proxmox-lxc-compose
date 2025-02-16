package container_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/larkinwc/proxmox-lxc-compose/pkg/config"
	"github.com/larkinwc/proxmox-lxc-compose/pkg/container"
	testing_internal "github.com/larkinwc/proxmox-lxc-compose/pkg/internal/testing"
)

func TestStateManager(t *testing.T) {
	dir := t.TempDir()
	statePath := filepath.Join(dir, "state")
	err := os.MkdirAll(statePath, 0755)
	testing_internal.AssertNoError(t, err)

	manager, err := container.NewStateManager(statePath)
	testing_internal.AssertNoError(t, err)

	containerName := "test-container"
	containerConfig := &config.Container{
		Image: "ubuntu:20.04",
	}

	t.Run("save_and_get_state", func(t *testing.T) {
		// Save initial state
		err := manager.SaveContainerState(containerName, containerConfig, "STOPPED")
		testing_internal.AssertNoError(t, err)

		// Get state
		state, err := manager.GetContainerState(containerName)
		testing_internal.AssertNoError(t, err)
		testing_internal.AssertEqual(t, "STOPPED", state.Status)
		testing_internal.AssertEqual(t, containerConfig.Image, state.Config.Image)
	})

	t.Run("update_state", func(t *testing.T) {
		// Update state
		err := manager.SaveContainerState(containerName, containerConfig, "RUNNING")
		testing_internal.AssertNoError(t, err)

		// Verify update
		state, err := manager.GetContainerState(containerName)
		testing_internal.AssertNoError(t, err)
		testing_internal.AssertEqual(t, "RUNNING", state.Status)
	})

	t.Run("remove_state", func(t *testing.T) {
		// Remove state
		err := manager.RemoveContainerState(containerName)
		testing_internal.AssertNoError(t, err)

		// Verify removal
		_, err = manager.GetContainerState(containerName)
		testing_internal.AssertError(t, err)
	})

	t.Run("invalid_state_file", func(t *testing.T) {
		// Create invalid state file
		statePath := filepath.Join(dir, "state", containerName+".json")
		err := os.WriteFile(statePath, []byte("invalid json"), 0644)
		testing_internal.AssertNoError(t, err)

		// Attempt to read invalid state
		_, err = manager.GetContainerState(containerName)
		testing_internal.AssertError(t, err)
	})

	t.Run("state_file_permissions", func(t *testing.T) {
		cfg := &config.Container{
			Image: "ubuntu:20.04",
		}

		// Save state
		err := manager.SaveContainerState("secure-container", cfg, "STOPPED")
		testing_internal.AssertNoError(t, err)

		// Check file permissions
		statePath := filepath.Join(dir, "state", "secure-container.json")
		info, err := os.Stat(statePath)
		testing_internal.AssertNoError(t, err)
		testing_internal.AssertEqual(t, os.FileMode(0600), info.Mode().Perm())
	})
}
