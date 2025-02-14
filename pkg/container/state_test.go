package container

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"proxmox-lxc-compose/pkg/config"
	. "proxmox-lxc-compose/pkg/internal/testing"
)

func TestStateManager(t *testing.T) {
	t.Run("NewStateManager creates state directory", func(t *testing.T) {
		dir, cleanup := TempDir(t)
		defer cleanup()

		statePath := filepath.Join(dir, "state")
		sm, err := NewStateManager(statePath)
		AssertNoError(t, err)
		AssertFileExists(t, statePath)
		AssertEqual(t, statePath, sm.statePath)
	})

	t.Run("SaveContainerState creates and updates state", func(t *testing.T) {
		dir, cleanup := TempDir(t)
		defer cleanup()

		sm, err := NewStateManager(filepath.Join(dir, "state"))
		AssertNoError(t, err)

		cfg := &config.Container{
			Image: "ubuntu:20.04",
		}

		// Save initial state
		err = sm.SaveContainerState("test", cfg, "STOPPED")
		AssertNoError(t, err)

		// Verify state file exists
		statePath := filepath.Join(dir, "state", "test.json")
		AssertFileExists(t, statePath)

		// Read and verify state content
		data, err := os.ReadFile(statePath)
		AssertNoError(t, err)

		var state State
		err = json.Unmarshal(data, &state)
		AssertNoError(t, err)
		AssertEqual(t, "test", state.Name)
		AssertEqual(t, "STOPPED", state.Status)
		AssertEqual(t, cfg.Image, state.Config.Image)

		// Update state
		time.Sleep(time.Millisecond) // Ensure time difference
		err = sm.SaveContainerState("test", cfg, "RUNNING")
		AssertNoError(t, err)

		// Verify updated state
		state2, err := sm.loadState("test")
		AssertNoError(t, err)
		AssertEqual(t, "RUNNING", state2.Status)
		if state2.LastStartedAt == nil {
			t.Fatal("LastStartedAt should not be nil")
		}
	})

	t.Run("GetContainerState returns correct state", func(t *testing.T) {
		dir, cleanup := TempDir(t)
		defer cleanup()

		sm, err := NewStateManager(filepath.Join(dir, "state"))
		AssertNoError(t, err)

		// Try to get non-existent state
		_, err = sm.GetContainerState("nonexistent")
		AssertError(t, err)

		// Save and get state
		cfg := &config.Container{Image: "ubuntu:20.04"}
		err = sm.SaveContainerState("test", cfg, "STOPPED")
		AssertNoError(t, err)

		state, err := sm.GetContainerState("test")
		AssertNoError(t, err)
		AssertEqual(t, "test", state.Name)
		AssertEqual(t, "STOPPED", state.Status)
	})

	t.Run("RemoveContainerState removes state", func(t *testing.T) {
		dir, cleanup := TempDir(t)
		defer cleanup()

		sm, err := NewStateManager(filepath.Join(dir, "state"))
		AssertNoError(t, err)

		// Save state
		cfg := &config.Container{Image: "ubuntu:20.04"}
		err = sm.SaveContainerState("test", cfg, "STOPPED")
		AssertNoError(t, err)

		// Remove state
		err = sm.RemoveContainerState("test")
		AssertNoError(t, err)

		// Verify state is removed
		_, err = sm.GetContainerState("test")
		AssertError(t, err)

		// Verify state file is removed
		statePath := filepath.Join(dir, "state", "test.json")
		if _, err := os.Stat(statePath); err == nil {
			t.Fatal("state file should not exist")
		}
	})

	t.Run("loadStates loads existing states", func(t *testing.T) {
		dir, cleanup := TempDir(t)
		defer cleanup()

		statePath := filepath.Join(dir, "state")
		err := os.MkdirAll(statePath, 0755)
		AssertNoError(t, err)

		// Create test state files
		states := map[string]State{
			"test1": {
				Name:   "test1",
				Status: "RUNNING",
				Config: &config.Container{Image: "ubuntu:20.04"},
			},
			"test2": {
				Name:   "test2",
				Status: "STOPPED",
				Config: &config.Container{Image: "alpine:latest"},
			},
		}

		for name, state := range states {
			data, err := json.MarshalIndent(state, "", "  ")
			AssertNoError(t, err)
			err = os.WriteFile(filepath.Join(statePath, name+".json"), data, 0644)
			AssertNoError(t, err)
		}

		// Create new state manager and verify states are loaded
		sm, err := NewStateManager(statePath)
		AssertNoError(t, err)

		for name, expected := range states {
			state, err := sm.GetContainerState(name)
			AssertNoError(t, err)
			AssertEqual(t, expected.Name, state.Name)
			AssertEqual(t, expected.Status, state.Status)
			AssertEqual(t, expected.Config.Image, state.Config.Image)
		}
	})
}
