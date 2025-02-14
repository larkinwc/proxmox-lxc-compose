package container

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

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
	// Restart stops and then starts a container
	Restart(name string) error
	// Update updates a container's configuration
	Update(name string, cfg *config.Container) error
}

// LXCManager implements the Manager interface for LXC containers
type LXCManager struct {
	configPath string
	state      *StateManager
}

// NewLXCManager creates a new LXC container manager
func NewLXCManager(configPath string) (*LXCManager, error) {
	if err := os.MkdirAll(configPath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create config directory: %w", err)
	}

	stateManager, err := NewStateManager(filepath.Join(configPath, "state"))
	if err != nil {
		return nil, fmt.Errorf("failed to create state manager: %w", err)
	}

	return &LXCManager{
		configPath: configPath,
		state:      stateManager,
	}, nil
}

func (m *LXCManager) execLXCCommand(name string, args ...string) error {
	cmd := execCommand(name, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		outputStr := strings.TrimSpace(string(output))
		if len(outputStr) > 0 {
			// Parse common LXC error messages
			if strings.Contains(outputStr, "not found") {
				return fmt.Errorf("container not found")
			}
			if strings.Contains(outputStr, "already running") {
				return fmt.Errorf("container is already running")
			}
			if strings.Contains(outputStr, "not running") {
				return fmt.Errorf("container is not running")
			}
			return fmt.Errorf("%s", outputStr)
		}
		return err
	}
	return nil
}

// ContainerExists checks if a container exists
func (m *LXCManager) ContainerExists(name string) bool {
	// Check if the container exists in our state
	if _, err := m.state.GetContainerState(name); err == nil {
		return true
	}

	// Check if the container directory exists
	containerPath := filepath.Join(m.configPath, name)
	if _, err := os.Stat(containerPath); err == nil {
		return true
	}

	// Check if the container exists in LXC
	if err := m.execLXCCommand("lxc-info", "-n", name); err == nil {
		return true
	}

	return false
}

// Create implements Manager.Create
func (m *LXCManager) Create(name string, cfg *config.Container) error {
	if m.ContainerExists(name) {
		return fmt.Errorf("container %s already exists", name)
	}

	// Create container directory
	containerPath := filepath.Join(m.configPath, name)
	if err := os.MkdirAll(containerPath, 0755); err != nil {
		return fmt.Errorf("failed to create container directory: %w", err)
	}

	// Save initial state
	if err := m.state.SaveContainerState(name, cfg, "STOPPED"); err != nil {
		return fmt.Errorf("failed to save container state: %w", err)
	}

	return nil
}

// Start implements Manager.Start
func (m *LXCManager) Start(name string) error {
	container, err := m.Get(name)
	if err != nil {
		return fmt.Errorf("failed to get container: %w", err)
	}

	if container.State == "RUNNING" {
		return fmt.Errorf("container '%s' is already running", name)
	}

	if container.State != "STOPPED" {
		return fmt.Errorf("container '%s' is not in a valid state for starting (current state: %s)", name, container.State)
	}

	// Start the container
	if err := m.execLXCCommand("lxc-start", "-n", name); err != nil {
		return fmt.Errorf("failed to start container: %w", err)
	}

	// Update state
	if err := m.state.SaveContainerState(name, container.Config, "RUNNING"); err != nil {
		return fmt.Errorf("failed to update container state: %w", err)
	}

	return nil
}

// Stop implements Manager.Stop
func (m *LXCManager) Stop(name string) error {
	container, err := m.Get(name)
	if err != nil {
		return fmt.Errorf("failed to get container: %w", err)
	}

	if container.State == "STOPPED" {
		return fmt.Errorf("container '%s' is already stopped", name)
	}

	if container.State != "RUNNING" && container.State != "FROZEN" {
		return fmt.Errorf("container '%s' is not in a valid state for stopping (current state: %s)", name, container.State)
	}

	// Stop the container using execLXCCommand
	if err := m.execLXCCommand("lxc-stop", "-n", name); err != nil {
		return fmt.Errorf("failed to stop container: %w", err)
	}

	// Update state
	if err := m.state.SaveContainerState(name, container.Config, "STOPPED"); err != nil {
		return fmt.Errorf("failed to update container state: %w", err)
	}

	return nil
}

// Remove implements Manager.Remove
func (m *LXCManager) Remove(name string) error {
	container, err := m.Get(name)
	if err != nil {
		return fmt.Errorf("failed to get container: %w", err)
	}

	if container.State != "STOPPED" {
		return fmt.Errorf("container '%s' must be stopped before removal", name)
	}

	// Remove container directory
	containerPath := filepath.Join(m.configPath, name)
	if err := os.RemoveAll(containerPath); err != nil {
		return fmt.Errorf("failed to remove container directory: %w", err)
	}

	// Remove state
	if err := m.state.RemoveContainerState(name); err != nil {
		return fmt.Errorf("failed to remove container state: %w", err)
	}

	return nil
}

// List implements Manager.List
func (m *LXCManager) List() ([]Container, error) {
	entries, err := os.ReadDir(m.configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config directory: %w", err)
	}

	var containers []Container
	for _, entry := range entries {
		if !entry.IsDir() || entry.Name() == "state" || entry.Name() == "templates" {
			continue
		}

		container, err := m.Get(entry.Name())
		if err != nil {
			continue
		}

		containers = append(containers, *container)
	}

	return containers, nil
}

// Get implements Manager.Get
func (m *LXCManager) Get(name string) (*Container, error) {
	if !m.ContainerExists(name) {
		return nil, fmt.Errorf("container %s does not exist", name)
	}

	// First check if we have state info
	state, err := m.state.GetContainerState(name)
	if err != nil {
		// Create default state if none exists
		state = &State{
			Name:   name,
			Status: "STOPPED",
		}
	}

	// Try up to 3 times to get a stable state
	for i := 0; i < 3; i++ {
		cmd := execCommand("lxc-info", "-n", name)
		output, err := cmd.CombinedOutput()
		if err == nil {
			currentState := ""
			// Parse lxc-info output to get state
			for _, line := range strings.Split(string(output), "\n") {
				if strings.HasPrefix(line, "State:") {
					lxcState := strings.TrimSpace(strings.TrimPrefix(line, "State:"))
					switch strings.ToUpper(lxcState) {
					case "RUNNING":
						currentState = "RUNNING"
					case "STOPPED":
						currentState = "STOPPED"
					case "FROZEN":
						currentState = "FROZEN"
					}
					break
				}
			}

			// If we got a valid state that matches our saved state or we've tried 3 times,
			// use this state
			if currentState != "" && (currentState == state.Status || i == 2) {
				state.Status = currentState
				break
			}
		}

		// Wait a short time before retrying
		if i < 2 {
			time.Sleep(100 * time.Millisecond)
		}
	}

	return &Container{
		Name:   name,
		State:  state.Status,
		Config: state.Config,
	}, nil
}

// Pause implements Manager.Pause
func (m *LXCManager) Pause(name string) error {
	container, err := m.Get(name)
	if err != nil {
		return fmt.Errorf("failed to get container: %w", err)
	}

	if container.State == "FROZEN" {
		return fmt.Errorf("container '%s' is already frozen", name)
	}

	if container.State != "RUNNING" {
		return fmt.Errorf("container '%s' is not in a valid state for pausing (current state: %s)", name, container.State)
	}

	// Freeze the container
	if err := m.execLXCCommand("lxc-freeze", "-n", name); err != nil {
		return fmt.Errorf("failed to pause container: %w", err)
	}

	// Update state
	if err := m.state.SaveContainerState(name, container.Config, "FROZEN"); err != nil {
		return fmt.Errorf("failed to update container state: %w", err)
	}

	return nil
}

// Resume implements Manager.Resume
func (m *LXCManager) Resume(name string) error {
	container, err := m.Get(name)
	if err != nil {
		return fmt.Errorf("failed to get container: %w", err)
	}

	if container.State == "RUNNING" {
		return fmt.Errorf("container '%s' is already running", name)
	}

	if container.State != "FROZEN" {
		return fmt.Errorf("container '%s' is not in a valid state for resuming (current state: %s)", name, container.State)
	}

	// Unfreeze the container
	if err := m.execLXCCommand("lxc-unfreeze", "-n", name); err != nil {
		return fmt.Errorf("failed to resume container: %w", err)
	}

	// Update state
	if err := m.state.SaveContainerState(name, container.Config, "RUNNING"); err != nil {
		return fmt.Errorf("failed to update container state: %w", err)
	}

	return nil
}

// Restart implements Manager.Restart
func (m *LXCManager) Restart(name string) error {
	container, err := m.Get(name)
	if err != nil {
		return fmt.Errorf("failed to get container: %w", err)
	}

	// If container is running or frozen, stop it first
	if container.State == "RUNNING" || container.State == "FROZEN" {
		if err := m.execLXCCommand("lxc-stop", "-n", name); err != nil {
			return fmt.Errorf("failed to stop container: %w", err)
		}

		// Update state to STOPPED
		if err := m.state.SaveContainerState(name, container.Config, "STOPPED"); err != nil {
			return fmt.Errorf("failed to update container state: %w", err)
		}

		// Verify state update
		container, err = m.Get(name)
		if err != nil {
			return fmt.Errorf("failed to get container state: %w", err)
		}
	}

	// Start the container
	if err := m.execLXCCommand("lxc-start", "-n", name); err != nil {
		return fmt.Errorf("failed to start container: %w", err)
	}

	// Update state to RUNNING
	if err := m.state.SaveContainerState(name, container.Config, "RUNNING"); err != nil {
		return fmt.Errorf("failed to update container state: %w", err)
	}

	return nil
}

// Update implements Manager.Update
func (m *LXCManager) Update(name string, cfg *config.Container) error {
	container, err := m.Get(name)
	if err != nil {
		return fmt.Errorf("failed to get container: %w", err)
	}

	// Save new configuration
	if err := m.state.SaveContainerState(name, cfg, container.State); err != nil {
		return fmt.Errorf("failed to update container configuration: %w", err)
	}

	return nil
}
