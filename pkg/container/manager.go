package container

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"

	"proxmox-lxc-compose/pkg/config"
)

// Manager defines the interface for managing LXC containers
type Manager interface {
	// Create creates a new container from the given configuration
	Create(name string, cfg *config.Container) error
	// Start starts a container
	Start(name string) error
	// Stop stops a container
	Stop(name string) error
	// Remove removes a container
	Remove(name string) error
	// List returns a list of all containers
	List() ([]Container, error)
	// Get returns information about a specific container
	Get(name string) (*Container, error)
	// Pause freezes a running container
	Pause(name string) error
	// Resume unfreezes a paused container
	Resume(name string) error
}

// Container represents a running LXC container
type Container struct {
	Name   string
	State  string
	Config *config.Container
}

// LXCManager implements the Manager interface for LXC containers
type LXCManager struct {
	configPath string
	state      *StateManager
}

// NewLXCManager creates a new LXC container manager
func NewLXCManager(configPath string) (*LXCManager, error) {
	statePath := filepath.Join(configPath, "state")
	stateManager, err := NewStateManager(statePath)
	if err != nil {
		return nil, fmt.Errorf("failed to create state manager: %w", err)
	}

	return &LXCManager{
		configPath: configPath,
		state:      stateManager,
	}, nil
}

// Create implements Manager.Create
func (m *LXCManager) Create(name string, cfg *config.Container) error {
	// Check if lxc-create is available
	if _, err := exec.LookPath("lxc-create"); err != nil {
		return fmt.Errorf("lxc-create not found: %w", err)
	}

	// Create container with template
	createCmd := exec.Command("lxc-create", "-n", name, "-t", "none")
	if err := createCmd.Run(); err != nil {
		return fmt.Errorf("failed to create container: %w", err)
	}

	// Apply configuration
	if err := m.applyConfig(name, cfg); err != nil {
		return fmt.Errorf("failed to apply configuration: %w", err)
	}

	// Save initial state
	if err := m.state.SaveContainerState(name, cfg, "STOPPED"); err != nil {
		return fmt.Errorf("failed to save container state: %w", err)
	}

	return nil
}

// Start implements Manager.Start
func (m *LXCManager) Start(name string) error {
	cmd := exec.Command("lxc-start", "-n", name)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to start container: %w", err)
	}

	// Update state
	if err := m.state.SaveContainerState(name, nil, "RUNNING"); err != nil {
		return fmt.Errorf("failed to update container state: %w", err)
	}

	return nil
}

// Stop implements Manager.Stop
func (m *LXCManager) Stop(name string) error {
	cmd := exec.Command("lxc-stop", "-n", name)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to stop container: %w", err)
	}

	// Update state
	if err := m.state.SaveContainerState(name, nil, "STOPPED"); err != nil {
		return fmt.Errorf("failed to update container state: %w", err)
	}

	return nil
}

// Remove implements Manager.Remove
func (m *LXCManager) Remove(name string) error {
	cmd := exec.Command("lxc-destroy", "-n", name)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to remove container: %w", err)
	}

	// Remove state
	if err := m.state.RemoveContainerState(name); err != nil {
		return fmt.Errorf("failed to remove container state: %w", err)
	}

	return nil
}

// List implements Manager.List
func (m *LXCManager) List() ([]Container, error) {
	cmd := exec.Command("lxc-ls", "--fancy", "--fancy-format", "name,state")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list containers: %w", err)
	}

	var containers []Container
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	// Skip header line
	for _, line := range lines[1:] {
		fields := strings.Fields(line)
		if len(fields) >= 2 {
			container := Container{
				Name:  fields[0],
				State: fields[1],
			}

			// Try to get state from state manager
			if state, err := m.state.GetContainerState(container.Name); err == nil {
				container.Config = state.Config
			}

			containers = append(containers, container)
		}
	}

	return containers, nil
}

// Get implements Manager.Get
func (m *LXCManager) Get(name string) (*Container, error) {
	cmd := exec.Command("lxc-info", "-n", name)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get container info: %w", err)
	}

	container := &Container{
		Name: name,
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, line := range lines {
		parts := strings.Split(line, ":")
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		if key == "State" {
			container.State = value
		}
	}

	// Try to get state from state manager
	if state, err := m.state.GetContainerState(name); err == nil {
		container.Config = state.Config
	}

	return container, nil
}

// Pause implements Manager.Pause
func (m *LXCManager) Pause(name string) error {
	// Check if container exists and is running
	container, err := m.Get(name)
	if err != nil {
		return fmt.Errorf("failed to get container: %w", err)
	}

	if container.State != "RUNNING" {
		return fmt.Errorf("container '%s' is not running (current state: %s)", name, container.State)
	}

	// Freeze the container
	cmd := exec.Command("lxc-freeze", "-n", name)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to pause container: %w", err)
	}

	// Update state
	if err := m.state.SaveContainerState(name, nil, "FROZEN"); err != nil {
		return fmt.Errorf("failed to update container state: %w", err)
	}

	return nil
}

// Resume implements Manager.Resume
func (m *LXCManager) Resume(name string) error {
	// Check if container exists and is paused
	container, err := m.Get(name)
	if err != nil {
		return fmt.Errorf("failed to get container: %w", err)
	}

	if container.State != "FROZEN" {
		return fmt.Errorf("container '%s' is not paused (current state: %s)", name, container.State)
	}

	// Unfreeze the container
	cmd := exec.Command("lxc-unfreeze", "-n", name)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to resume container: %w", err)
	}

	// Update state
	if err := m.state.SaveContainerState(name, nil, "RUNNING"); err != nil {
		return fmt.Errorf("failed to update container state: %w", err)
	}

	return nil
}
