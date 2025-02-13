package container

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
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
	mu        sync.RWMutex
	states    map[string]*State
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

	// Load existing states
	if err := sm.loadStates(); err != nil {
		return nil, err
	}

	return sm, nil
}

// SaveContainerState saves the state of a container
func (sm *StateManager) SaveContainerState(name string, cfg *config.Container, status string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	state, exists := sm.states[name]
	if !exists {
		state = &State{
			Name:      name,
			CreatedAt: time.Now(),
			Config:    cfg,
		}
		sm.states[name] = state
	}

	now := time.Now()
	if status == "RUNNING" {
		state.LastStartedAt = &now
	} else if status == "STOPPED" {
		state.LastStoppedAt = &now
	}
	state.Status = status

	return sm.saveState(name, state)
}

// GetContainerState retrieves the state of a container
func (sm *StateManager) GetContainerState(name string) (*State, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	state, exists := sm.states[name]
	if !exists {
		return nil, fmt.Errorf("no state found for container %s", name)
	}

	return state, nil
}

// RemoveContainerState removes the state of a container
func (sm *StateManager) RemoveContainerState(name string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	delete(sm.states, name)
	return os.Remove(sm.getStatePath(name))
}

// loadStates loads all container states from disk
func (sm *StateManager) loadStates() error {
	files, err := os.ReadDir(sm.statePath)
	if err != nil {
		return fmt.Errorf("failed to read state directory: %w", err)
	}

	for _, file := range files {
		if file.IsDir() || filepath.Ext(file.Name()) != ".json" {
			continue
		}

		name := file.Name()[:len(file.Name())-5] // remove .json extension
		state, err := sm.loadState(name)
		if err != nil {
			return err
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
		return nil, fmt.Errorf("failed to parse state file: %w", err)
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
