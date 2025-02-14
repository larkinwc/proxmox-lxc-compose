package container_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"proxmox-lxc-compose/pkg/config"
	"proxmox-lxc-compose/pkg/container"
	testing_internal "proxmox-lxc-compose/pkg/internal/testing"
)

func TestStateManager(t *testing.T) {
	// Initialize temp directory
	tmpDir := t.TempDir()

	// Create state manager
	sm, err := container.NewStateManager(tmpDir)
	testing_internal.AssertNoError(t, err)
	testing_internal.AssertFileExists(t, tmpDir)
	testing_internal.AssertEqual(t, 0, len(sm.GetStates()))

	statePath := sm.GetStatePath()
	testing_internal.AssertEqual(t, tmpDir, statePath)

	stateFile := filepath.Join(statePath, "test-container.state")
	testing_internal.AssertFileNotExists(t, stateFile)

	// Test loading non-existent state (should error)
	_, err = sm.LoadStateFromDisk("test-container")
	testing_internal.AssertError(t, err)

	// Test saving and loading state
	state := &container.State{
		Name:      "test-container",
		Status:    "running",
		CreatedAt: time.Now(),
		Config:    &config.Container{},
	}

	err = sm.SaveState(state)
	testing_internal.AssertNoError(t, err)
	testing_internal.AssertFileExists(t, filepath.Join(statePath, "test-container.json"))

	// Test loading the state back
	loadedState, err := sm.LoadStateFromDisk("test-container")
	testing_internal.AssertNoError(t, err)
	testing_internal.AssertEqual(t, state.Name, loadedState.Name)
	testing_internal.AssertEqual(t, state.Status, loadedState.Status)

	t.Run("NewStateManager creates state directory", func(t *testing.T) {
		dir := t.TempDir()

		sm, err := container.NewStateManager(dir)
		testing_internal.AssertNoError(t, err)
		testing_internal.AssertFileExists(t, dir)
		testing_internal.AssertEqual(t, dir, sm.GetStatePath())
	})

	t.Run("SaveContainerState creates and updates state", func(t *testing.T) {
		dir := t.TempDir()

		sm, err := container.NewStateManager(dir)
		testing_internal.AssertNoError(t, err)

		cfg := &config.Container{
			Image: "ubuntu:20.04",
		}

		// Save initial state
		err = sm.SaveContainerState("test", cfg, "STOPPED")
		testing_internal.AssertNoError(t, err)

		// Verify state file exists
		statePath := filepath.Join(dir, "test.json")
		testing_internal.AssertFileExists(t, statePath)

		// Read and verify state content
		data, err := os.ReadFile(statePath)
		testing_internal.AssertNoError(t, err)

		var state container.State
		err = json.Unmarshal(data, &state)
		testing_internal.AssertNoError(t, err)
		testing_internal.AssertEqual(t, "test", state.Name)
		testing_internal.AssertEqual(t, "STOPPED", state.Status)
		testing_internal.AssertEqual(t, cfg.Image, state.Config.Image)

		// Update state
		time.Sleep(time.Millisecond) // Ensure time difference
		err = sm.SaveContainerState("test", cfg, "RUNNING")
		testing_internal.AssertNoError(t, err)

		// Verify updated state
		state2, err := sm.LoadStateFromDisk("test")
		testing_internal.AssertNoError(t, err)
		testing_internal.AssertEqual(t, "RUNNING", state2.Status)
		if state2.LastStartedAt == nil {
			t.Fatal("LastStartedAt should not be nil")
		}
	})

	t.Run("GetContainerState returns correct state", func(t *testing.T) {
		dir := t.TempDir()

		sm, err := container.NewStateManager(dir)
		testing_internal.AssertNoError(t, err)

		// Try to get non-existent state
		_, err = sm.GetContainerState("nonexistent")
		testing_internal.AssertError(t, err)

		// Save and get state
		cfg := &config.Container{Image: "ubuntu:20.04"}
		err = sm.SaveContainerState("test", cfg, "STOPPED")
		testing_internal.AssertNoError(t, err)

		state, err := sm.GetContainerState("test")
		testing_internal.AssertNoError(t, err)
		testing_internal.AssertEqual(t, "test", state.Name)
		testing_internal.AssertEqual(t, "STOPPED", state.Status)
	})

	t.Run("RemoveContainerState removes state", func(t *testing.T) {
		dir := t.TempDir()

		sm, err := container.NewStateManager(dir)
		testing_internal.AssertNoError(t, err)

		// Save state
		cfg := &config.Container{Image: "ubuntu:20.04"}
		err = sm.SaveContainerState("test", cfg, "STOPPED")
		testing_internal.AssertNoError(t, err)

		// Remove state
		err = sm.RemoveContainerState("test")
		testing_internal.AssertNoError(t, err)

		// Verify state is removed
		_, err = sm.GetContainerState("test")
		testing_internal.AssertError(t, err)

		// Verify state file is removed
		statePath := filepath.Join(dir, "test.json")
		if _, err := os.Stat(statePath); err == nil {
			t.Fatal("state file should not exist")
		}
	})

	t.Run("loadStates loads existing states", func(t *testing.T) {
		dir := t.TempDir()

		err := os.MkdirAll(dir, 0755)
		testing_internal.AssertNoError(t, err)

		// Create test state files
		states := map[string]container.State{
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
			testing_internal.AssertNoError(t, err)
			err = os.WriteFile(filepath.Join(dir, name+".json"), data, 0644)
			testing_internal.AssertNoError(t, err)
		}

		// Create new state manager and verify states are loaded
		sm, err := container.NewStateManager(dir)
		testing_internal.AssertNoError(t, err)

		for name, expected := range states {
			state, err := sm.GetContainerState(name)
			testing_internal.AssertNoError(t, err)
			testing_internal.AssertEqual(t, expected.Name, state.Name)
			testing_internal.AssertEqual(t, expected.Status, state.Status)
			testing_internal.AssertEqual(t, expected.Config.Image, state.Config.Image)
		}
	})
}
