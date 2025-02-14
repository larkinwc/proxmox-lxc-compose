package container

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"proxmox-lxc-compose/pkg/config"
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

// SaveContainerState saves the state of a container
func (sm *StateManager) SaveContainerState(name string, cfg *config.Container, status string) error {
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
			state.LastStartedAt = &time.Time{}
			*state.LastStartedAt = time.Now()
		} else if status == "STOPPED" && existing.Status == "RUNNING" {
			state.LastStoppedAt = &time.Time{}
			*state.LastStoppedAt = time.Now()
		}
	}

	sm.states[name] = state

	if err := sm.saveState(name, state); err != nil {
		return fmt.Errorf("failed to save state: %w", err)
	}

	return nil
}

// GetContainerState retrieves the state of a container
func (sm *StateManager) GetContainerState(name string) (*State, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	state, ok := sm.states[name]
	if !ok {
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

	if err := os.WriteFile(sm.getStatePath(name), data, 0644); err != nil {
		return fmt.Errorf("failed to write state file: %w", err)
	}

	return nil
}

// getStatePath returns the path to a container's state file
func (sm *StateManager) getStatePath(name string) string {
	return filepath.Join(sm.statePath, name+".json")
}
