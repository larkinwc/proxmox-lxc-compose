// Package mock provides mocking functionality for testing
package mock

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// CommandState tracks the state of mock commands
type CommandState struct {
	mu             sync.RWMutex
	Name           string
	Args           []string
	tmpFiles       []string
	debug          bool
	commandHistory []struct {
		name string
		args []string
	}
}

type containerState struct {
	Name          string     `json:"name"`
	CreatedAt     time.Time  `json:"created_at"`
	LastStartedAt *time.Time `json:"last_started_at,omitempty"`
	Status        string     `json:"status"`
}

// SetDebug enables or disables debug logging
func (m *CommandState) SetDebug(enabled bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.debug = enabled
}

// getStateFromFile reads the container state from its state file
func (m *CommandState) getStateFromFile(containerName string) (string, error) {
	if configPath, ok := os.LookupEnv("CONTAINER_CONFIG_PATH"); ok {
		stateFilePath := filepath.Join(configPath, "state", containerName+".json")
		data, err := os.ReadFile(stateFilePath)
		if err != nil {
			return "", err
		}

		var stateData struct {
			Status string `json:"status"`
		}
		if err := json.Unmarshal(data, &stateData); err != nil {
			return "", err
		}

		return stateData.Status, nil
	}
	return "", fmt.Errorf("CONTAINER_CONFIG_PATH not set")
}

// SetContainerState allows tests to set the state of a container
func (m *CommandState) SetContainerState(containerName, state string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	state = strings.ToUpper(state)

	if configPath, ok := os.LookupEnv("CONTAINER_CONFIG_PATH"); ok {
		// Create state directory if it doesn't exist
		statePath := filepath.Join(configPath, "state")
		if err := os.MkdirAll(statePath, 0755); err != nil {
			return fmt.Errorf("failed to create state directory: %w", err)
		}

		// Create state file
		stateFilePath := filepath.Join(statePath, containerName+".json")
		stateData := containerState{
			Name:      containerName,
			CreatedAt: time.Now(),
			Status:    state,
		}

		data, err := json.MarshalIndent(stateData, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal state data: %w", err)
		}

		if err := os.WriteFile(stateFilePath, data, 0644); err != nil {
			return fmt.Errorf("failed to write state file: %w", err)
		}
	}
	return nil
}

// AddContainer adds a container to the mock state and ensures its state file exists
func (m *CommandState) AddContainer(containerName, state string) error {
	if m.debug {
		fmt.Printf("DEBUG: Adding container %s with state %s\n", containerName, state)
	}

	if configPath, ok := os.LookupEnv("CONTAINER_CONFIG_PATH"); ok {
		// Create container directory
		containerPath := filepath.Join(configPath, containerName)
		if err := os.MkdirAll(containerPath, 0755); err != nil {
			return fmt.Errorf("failed to create container directory: %w", err)
		}

		// Create a dummy config file to mark this as a valid container
		configFile := filepath.Join(containerPath, "config")
		if err := os.WriteFile(configFile, []byte("# LXC Config\n"), 0644); err != nil {
			return fmt.Errorf("failed to create config file: %w", err)
		}
	}

	// Set the container state
	return m.SetContainerState(containerName, state)
}

// ContainerExists checks if a container exists
func (m *CommandState) ContainerExists(containerName string) bool {
	if m.debug {
		fmt.Printf("DEBUG: ContainerExists check for %s\n", containerName)
	}
	if containerName == "nonexistent" {
		if m.debug {
			fmt.Printf("DEBUG: Container %s is nonexistent\n", containerName)
		}
		return false
	}
	if configPath, ok := os.LookupEnv("CONTAINER_CONFIG_PATH"); ok {
		// Check container directory first
		containerPath := filepath.Join(configPath, containerName)
		dirExists := false
		if _, err := os.Stat(containerPath); err == nil {
			dirExists = true
		}
		// Then check state file
		stateFilePath := filepath.Join(configPath, "state", containerName+".json")
		stateExists := false
		if _, err := os.Stat(stateFilePath); err == nil {
			stateExists = true
		}
		if m.debug {
			fmt.Printf("DEBUG: Container %s directory exists: %v, state exists: %v\n", containerName, dirExists, stateExists)
		}
		return dirExists || stateExists
	}
	if m.debug {
		fmt.Printf("DEBUG: CONTAINER_CONFIG_PATH not set\n")
	}
	return false
}

// GetContainerState returns the current state of a container
func (m *CommandState) GetContainerState(containerName string) string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	state, err := m.getStateFromFile(containerName)
	if err != nil {
		return ""
	}
	return state
}

func (m *CommandState) execLXCCommand(containerName string, command string, _ ...string) error {
	if m.debug {
		fmt.Printf("DEBUG: Executing command %s for container %s\n", command, containerName)
	}

	// First verify container exists
	if !m.ContainerExists(containerName) {
		if m.debug {
			fmt.Printf("DEBUG: Container %s does not exist\n", containerName)
		}
		return fmt.Errorf("container %s does not exist", containerName)
	}

	// Get current state with retries to ensure stability
	var state string
	var err error
	for i := 0; i < 3; i++ {
		state, err = m.getStateFromFile(containerName)
		if err == nil {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if err != nil {
		if m.debug {
			fmt.Printf("DEBUG: Failed to get state for %s: %v\n", containerName, err)
		}
		return fmt.Errorf("failed to get container state: %w", err)
	}
	if m.debug {
		fmt.Printf("DEBUG: Current state for %s: %s\n", containerName, state)
	}

	// Handle command
	newState := state
	switch command {
	case "lxc-start":
		if strings.ToUpper(state) != "STOPPED" {
			if m.debug {
				fmt.Printf("DEBUG: Cannot start container %s in state %s\n", containerName, state)
			}
			return fmt.Errorf("container is not in a valid state for starting (current state: %s)", state)
		}
		newState = "RUNNING"
	case "lxc-stop":
		if strings.ToUpper(state) != "RUNNING" && strings.ToUpper(state) != "FROZEN" {
			if m.debug {
				fmt.Printf("DEBUG: Cannot stop container %s in state %s\n", containerName, state)
			}
			return fmt.Errorf("container is not in a valid state for stopping (current state: %s)", state)
		}
		newState = "STOPPED"
	case "lxc-freeze":
		if strings.ToUpper(state) != "RUNNING" {
			return fmt.Errorf("container is not in a valid state for freezing (current state: %s)", state)
		}
		newState = "FROZEN"
	case "lxc-unfreeze":
		if strings.ToUpper(state) != "FROZEN" {
			return fmt.Errorf("container is not in a valid state for unfreezing (current state: %s)", state)
		}
		newState = "RUNNING"
	}

	if m.debug && newState != state {
		fmt.Printf("DEBUG: Transitioning container %s from %s to %s\n", containerName, state, newState)
	}

	// Update state if changed with retries to ensure stability
	if newState != state {
		for i := 0; i < 3; i++ {
			if err := m.SetContainerState(containerName, newState); err == nil {
				break
			}
			if i < 2 {
				time.Sleep(50 * time.Millisecond)
			}
		}
	}

	return nil
}

// SetupMockCommand sets up command mocking and returns a cleanup function
func SetupMockCommand(execCommand *func(string, ...string) *exec.Cmd) (*CommandState, func()) {
	mock := &CommandState{
		tmpFiles: make([]string, 0),
		debug:    true,
		commandHistory: make([]struct {
			name string
			args []string
		}, 0),
	}
	oldExec := *execCommand
	*execCommand = func(name string, args ...string) *exec.Cmd {
		if mock.debug {
			fmt.Printf("DEBUG: Mock command called: %s %s\n", name, strings.Join(args, " "))
		}

		mock.mu.Lock()
		mock.Name = name
		mock.Args = args
		mock.commandHistory = append(mock.commandHistory, struct {
			name string
			args []string
		}{name, args})
		mock.mu.Unlock()

		// For all commands, first check if args are valid
		if len(args) < 2 || args[0] != "-n" {
			return exec.Command("sh", "-c", "echo 'Invalid arguments' >&2; exit 2")
		}

		containerName := args[1]
		if containerName == "nonexistent" {
			return exec.Command("sh", "-c", "echo 'Container does not exist' >&2; exit 2")
		}

		switch name {
		case "lxc-start", "lxc-stop", "lxc-freeze", "lxc-unfreeze":
			if err := mock.execLXCCommand(containerName, name); err != nil {
				return exec.Command("sh", "-c", fmt.Sprintf("echo '%s' >&2; exit 1", err))
			}
			return exec.Command("sh", "-c", "true")

		case "lxc-info":
			if !mock.ContainerExists(containerName) {
				return exec.Command("sh", "-c", "echo 'Container does not exist' >&2; exit 2")
			}
			state, err := mock.getStateFromFile(containerName)
			if err != nil {
				return exec.Command("sh", "-c", fmt.Sprintf("echo '%s' >&2; exit 1", err))
			}
			return exec.Command("sh", "-c", fmt.Sprintf("echo 'State: %s'", state))

		case "lxc-attach":
			if !mock.ContainerExists(containerName) {
				return exec.Command("sh", "-c", "echo 'Container does not exist' >&2; exit 2")
			}
			// Check if this is a tail -f command for console.log
			if strings.Contains(strings.Join(args, " "), "tail -f /var/log/console.log") {
				// Create a script that properly streams logs one at a time with line buffering
				return exec.Command("sh", "-c", `
					stdbuf -oL bash -c '
						echo "[$(date -Iseconds)] Container started"
						sleep 0.1
						echo "[$(date -Iseconds)] Service initialized"
						sleep 0.1
						echo "[$(date -Iseconds)] Ready to accept connections"
						# Keep streaming
						while true; do sleep 1; done
					'
				`)
			}
			return exec.Command("sh", "-c", "true")
		}

		return exec.Command("sh", "-c", fmt.Sprintf("echo 'Unknown command: %s' >&2; exit 1", name))
	}

	return mock, func() {
		if mock.debug {
			fmt.Printf("DEBUG: Cleaning up %d temporary files\n", len(mock.tmpFiles))
		}
		mock.mu.Lock()
		defer mock.mu.Unlock()
		for _, file := range mock.tmpFiles {
			os.Remove(file)
		}
		*execCommand = oldExec
	}
}
