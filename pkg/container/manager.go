package container

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/larkinwc/proxmox-lxc-compose/pkg/common"
	"github.com/larkinwc/proxmox-lxc-compose/pkg/config"
	"github.com/larkinwc/proxmox-lxc-compose/pkg/internal/recovery"
	"github.com/larkinwc/proxmox-lxc-compose/pkg/logging"
)

// Manager defines the interface for managing LXC containers
type Manager interface {
	// Create creates a new container from the given configuration
	Create(name string, cfg *common.Container) error
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
	Update(name string, cfg *common.Container) error
}

// LXCManager implements the Manager interface for LXC containers
type LXCManager struct {
	configPath string
	state      *StateManager
}

// NewLXCManager creates a new LXC container manager
func NewLXCManager(configPath string) (*LXCManager, error) {
	logging.Debug("Initializing LXC manager", "configPath", configPath)

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
	logging.Debug("Executing LXC command",
		"command", name,
		"args", args,
		"container", args[1], // args[1] is usually the container name
	)

	// Create a context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Use retry with backoff for commands that might fail temporarily
	return recovery.RetryWithBackoff(ctx, recovery.DefaultRetryConfig, func() error {
		cmd := ExecCommand(name, args...)
		output, err := cmd.CombinedOutput()

		// Check if the command timed out
		if ctx.Err() == context.DeadlineExceeded {
			return fmt.Errorf("command timed out after 5 seconds")
		}

		if err != nil {
			logging.Error("Command failed",
				"command", name,
				"args", args,
				"output", string(output),
				"error", err,
			)
			return fmt.Errorf("command failed: %w", err)
		}
		return nil
	})
}

// ContainerExists checks if a container exists
func (m *LXCManager) ContainerExists(name string) bool {
	logging.Debug("Checking if container exists", "name", name)

	// Only check state first - directory existence is not enough
	if _, err := m.state.GetContainerState(name); err == nil {
		logging.Debug("Container found in state", "name", name)
		return true
	}

	// If not in state, check LXC
	if err := m.execLXCCommand("lxc-info", "-n", name); err == nil {
		logging.Debug("Container found in LXC", "name", name)
		return true
	}

	logging.Debug("Container does not exist", "name", name)
	return false
}

// Create implements Manager.Create
func (m *LXCManager) Create(name string, cfg *common.Container) error {
	if cfg == nil {
		return fmt.Errorf("container configuration is required")
	}

	// Validate container configuration
	if err := validateContainerConfig(cfg); err != nil {
		return fmt.Errorf("invalid container configuration: %w", err)
	}

	if m.ContainerExists(name) {
		return fmt.Errorf("container %s already exists", name)
	}

	// Create container directory structure
	containerDir := filepath.Join(m.configPath, name)
	dirs := []string{
		containerDir,
		filepath.Join(containerDir, "rootfs"),
		filepath.Join(containerDir, "logs"),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create container directory %s: %w", dir, err)
		}
	}

	// Convert common.Container to config.Container for state saving
	configContainer := config.FromCommonContainer(cfg)

	// Configure network if specified
	if cfg.Network != nil {
		networkCfg := config.FromCommonNetworkConfig(cfg.Network)
		if err := m.configureNetwork(name, networkCfg); err != nil {
			return fmt.Errorf("failed to configure network: %w", err)
		}
	}

	// Save initial state
	if err := m.state.SaveContainerState(name, configContainer, "STOPPED"); err != nil {
		return fmt.Errorf("failed to save container state: %w", err)
	}

	logging.Debug("Container created and state saved", "name", name)

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

	// Update state - container.Config is already *config.Container
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

	// Update state - container.Config is already *config.Container
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

	// Destroy container in LXC
	if err := m.execLXCCommand("lxc-destroy", "-n", name); err != nil {
		return fmt.Errorf("failed to destroy container: %w", err)
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
		cmd := ExecCommand("lxc-info", "-n", name)
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

	// Update state - container.Config is already *config.Container
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

	// Update state - container.Config is already *config.Container
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

		// Update state to STOPPED - container.Config is already *config.Container
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

	// Update state to RUNNING - container.Config is already *config.Container
	if err := m.state.SaveContainerState(name, container.Config, "RUNNING"); err != nil {
		return fmt.Errorf("failed to update container state: %w", err)
	}

	return nil
}

// Update implements Manager.Update
func (m *LXCManager) Update(name string, cfg *common.Container) error {
	if cfg == nil {
		return fmt.Errorf("container configuration is required")
	}

	container, err := m.Get(name)
	if err != nil {
		return fmt.Errorf("failed to get container: %w", err)
	}

	// Convert common.Container to config.Container for state saving
	configContainer := config.FromCommonContainer(cfg)
	if err := m.state.SaveContainerState(name, configContainer, container.State); err != nil {
		return fmt.Errorf("failed to save container state: %w", err)
	}
	return nil
}
