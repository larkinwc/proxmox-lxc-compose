package container

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"proxmox-lxc-compose/pkg/config"
	"proxmox-lxc-compose/pkg/internal/recovery"
	"proxmox-lxc-compose/pkg/logging"
)

// State represents the persistent state of a container
type State struct {
	Name          string            `json:"name"`
	CreatedAt     time.Time         `json:"created_at"`
	LastStartedAt *time.Time        `json:"last_started_at,omitempty"`
	LastStoppedAt *time.Time        `json:"last_stopped_at,omitempty"`
	Config        *config.Container `json:"config"`
	Status        string            `json:"status"`
}

// StateManager handles container state persistence
type StateManager struct {
	statePath string
	states    map[string]*State
	mu        sync.RWMutex
}

// NewStateManager creates a new state manager
func NewStateManager(statePath string) (*StateManager, error) {
	logging.Debug("Initializing state manager", "path", statePath)

	if err := os.MkdirAll(statePath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create state directory: %w", err)
	}

	sm := &StateManager{
		statePath: statePath,
		states:    make(map[string]*State),
	}

	if err := sm.loadStates(); err != nil {
		return nil, fmt.Errorf("failed to load states: %w", err)
	}

	return sm, nil
}

// SaveContainerState saves the state of a container with retries
func (sm *StateManager) SaveContainerState(name string, cfg *config.Container, status string) error {
	logging.Debug("Saving container state",
		"name", name,
		"status", status,
		"config", fmt.Sprintf("%+v", cfg),
	)

	ctx := context.Background()
	return recovery.RetryWithBackoff(ctx, recovery.DefaultRetryConfig, func() error {
		sm.mu.Lock()
		defer sm.mu.Unlock()

		state := &State{
			Name:      name,
			Config:    cfg,
			Status:    status,
			CreatedAt: time.Now(),
		}

		if existing, ok := sm.states[name]; ok {
			state.CreatedAt = existing.CreatedAt
			if status == "RUNNING" && (existing.Status == "STOPPED" || existing.Status == "FROZEN") {
				now := time.Now()
				state.LastStartedAt = &now
				logging.Debug("Container started", "name", name, "time", now)
			} else if status == "STOPPED" && existing.Status == "RUNNING" {
				now := time.Now()
				state.LastStoppedAt = &now
				logging.Debug("Container stopped", "name", name, "time", now)
			}
		}

		sm.states[name] = state

		if err := sm.saveState(name, state); err != nil {
			logging.Error("Failed to save state",
				"name", name,
				"error", err,
			)
			return fmt.Errorf("failed to save state: %w", err)
		}

		return nil
	})
}

// GetContainerState retrieves the state of a container
func (sm *StateManager) GetContainerState(name string) (*State, error) {
	logging.Debug("Getting container state", "name", name)

	sm.mu.RLock()
	defer sm.mu.RUnlock()

	state, ok := sm.states[name]
	if !ok {
		logging.Debug("Container state not found", "name", name)
		return nil, fmt.Errorf("container %s does not exist", name)
	}

	return state, nil
}

// RemoveContainerState removes the state of a container
func (sm *StateManager) RemoveContainerState(name string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if _, ok := sm.states[name]; !ok {
		return fmt.Errorf("container %s does not exist", name)
	}

	delete(sm.states, name)

	stateFile := sm.getStatePath(name)
	if err := os.Remove(stateFile); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove state file: %w", err)
	}

	return nil
}

// loadStates loads all container states from disk
func (sm *StateManager) loadStates() error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	entries, err := os.ReadDir(sm.statePath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to read state directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}

		name := strings.TrimSuffix(entry.Name(), ".json")
		state, err := sm.loadState(name)
		if err != nil {
			continue // Skip invalid state files
		}

		sm.states[name] = state
	}

	return nil
}

// loadState loads a single container state from disk
func (sm *StateManager) loadState(name string) (*State, error) {
	data, err := os.ReadFile(sm.getStatePath(name))
	if err != nil {
		return nil, fmt.Errorf("failed to read state file: %w", err)
	}

	var state State
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("failed to unmarshal state: %w", err)
	}

	return &state, nil
}

// saveState saves a single container state to disk
func (sm *StateManager) saveState(name string, state *State) error {
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal state: %w", err)
	}

	if err := os.WriteFile(sm.getStatePath(name), data, 0600); err != nil {
		return fmt.Errorf("failed to write state file: %w", err)
	}

	return nil
}

// getStatePath returns the path to a container's state file
func (sm *StateManager) getStatePath(name string) string {
	return filepath.Join(sm.statePath, name+".json")
}

// GetStatePath returns the path to the state directory
func (sm *StateManager) GetStatePath() string {
	return sm.statePath
}

// GetStates returns all container states
func (sm *StateManager) GetStates() map[string]*State {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	states := make(map[string]*State)
	for k, v := range sm.states {
		states[k] = v
	}
	return states
}

// LoadStateFromDisk loads a container state from disk
func (sm *StateManager) LoadStateFromDisk(name string) (*State, error) {
	return sm.loadState(name)
}

func (sm *StateManager) GetState(name string) (*State, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	state, ok := sm.states[name]
	if !ok {
		return nil, fmt.Errorf("state not found for container: %s", name)
	}
	return state, nil
}

func (sm *StateManager) SaveState(state *State) error {
	if state == nil || state.Name == "" {
		return fmt.Errorf("invalid state: name is required")
	}

	// Create state directory if it doesn't exist
	if err := os.MkdirAll(sm.statePath, 0700); err != nil {
		return fmt.Errorf("failed to create state directory: %w", err)
	}

	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.states[state.Name] = state

	// Marshal state to JSON
	data, err := json.Marshal(state)
	if err != nil {
		return fmt.Errorf("failed to marshal state: %w", err)
	}

	// Write state file with restricted permissions
	stateFile := sm.getStatePath(state.Name)
	if err := os.WriteFile(stateFile, data, 0600); err != nil {
		return fmt.Errorf("failed to write state file: %w", err)
	}

	return nil
}
