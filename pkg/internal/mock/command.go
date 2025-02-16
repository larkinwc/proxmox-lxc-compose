// Package mock provides mocking functionality for testing
package mock

import (
	"fmt"
	"os/exec"
	"strings"
	"sync"
	"time"
)

// CommandState tracks the state of mock commands
type CommandState struct {
	mu              sync.RWMutex
	debug           bool
	ContainerStates map[string]string
	mockOutput      map[string][]byte
	CalledCommands  map[string]bool
	commandHistory  []struct {
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

// SetDebug enables or disables debug mode
func (m *CommandState) SetDebug(enabled bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.debug = enabled
}

// getContainerState gets the container state from memory
func (m *CommandState) getContainerState(containerName string) (string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if state, exists := m.ContainerStates[containerName]; exists {
		if m.debug {
			fmt.Printf("DEBUG: Found container state for %s: %s\n", containerName, state)
		}
		return state, nil
	}
	return "", fmt.Errorf("container state not found")
}

// SetContainerState allows tests to set the state of a container
func (m *CommandState) SetContainerState(containerName, state string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	state = strings.ToUpper(state)
	m.ContainerStates[containerName] = state

	if m.debug {
		fmt.Printf("DEBUG: Set container state for %s to %s\n", containerName, state)
	}
	return nil
}

// RemoveContainer removes a container's state
func (m *CommandState) RemoveContainer(containerName string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.ContainerStates, containerName)
	if m.debug {
		fmt.Printf("DEBUG: Removed container state for %s\n", containerName)
	}
	return nil
}

// AddContainer adds a container to the mock state
func (m *CommandState) AddContainer(containerName, state string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.debug {
		fmt.Printf("DEBUG: Adding container %s with state %s\n", containerName, state)
	}

	// Set the container state
	m.ContainerStates[containerName] = strings.ToUpper(state)
	return nil
}

// ContainerExists checks if a container exists
func (m *CommandState) ContainerExists(containerName string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.debug {
		fmt.Printf("DEBUG: ContainerExists check for %s\n", containerName)
	}

	_, exists := m.ContainerStates[containerName]
	if m.debug {
		fmt.Printf("DEBUG: Container %s exists: %v\n", containerName, exists)
	}
	return exists
}

// GetContainerState returns the current state of a container
func (m *CommandState) GetContainerState(containerName string) string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	state, _ := m.getContainerState(containerName)
	return state
}

// execLXCCommand handles LXC command execution and state transitions
func (m *CommandState) execLXCCommand(name string, args ...string) ([]byte, error) {
	if m.debug {
		fmt.Printf("DEBUG: Mock command called: %s %s\n", name, strings.Join(args, " "))
	}

	// Record command execution
	m.commandHistory = append(m.commandHistory, struct {
		name string
		args []string
	}{name, args})

	// Handle lxc-info command
	if name == "lxc-info" && len(args) >= 2 && args[0] == "-n" {
		containerName := args[1]
		if state, exists := m.ContainerStates[containerName]; exists {
			if m.debug {
				fmt.Printf("DEBUG: Found container state for %s: %s\n", containerName, state)
			}
			return []byte(fmt.Sprintf("Name: %s\nState: %s\n", containerName, state)), nil
		}
		return []byte("container does not exist\n"), fmt.Errorf("exit status 1")
	}

	// Handle lxc-destroy command
	if name == "lxc-destroy" && len(args) >= 2 && args[0] == "-n" {
		containerName := args[1]
		delete(m.ContainerStates, containerName)
		if m.debug {
			fmt.Printf("DEBUG: Removed container state for %s\n", containerName)
		}
		return nil, nil
	}

	// Handle lxc-start command
	if name == "lxc-start" && len(args) >= 2 && args[0] == "-n" {
		containerName := args[1]
		if _, exists := m.ContainerStates[containerName]; exists {
			m.ContainerStates[containerName] = "RUNNING"
			if m.debug {
				fmt.Printf("DEBUG: Started container %s\n", containerName)
			}
			return nil, nil
		}
		return []byte("container does not exist\n"), fmt.Errorf("exit status 1")
	}

	// Handle lxc-stop command
	if name == "lxc-stop" && len(args) >= 2 && args[0] == "-n" {
		containerName := args[1]
		if _, exists := m.ContainerStates[containerName]; exists {
			m.ContainerStates[containerName] = "STOPPED"
			if m.debug {
				fmt.Printf("DEBUG: Stopped container %s\n", containerName)
			}
			return nil, nil
		}
		return []byte("container does not exist\n"), fmt.Errorf("exit status 1")
	}

	return nil, nil
}

// mockCmd is a custom command type for mocking
type mockCmd struct {
	*exec.Cmd
	runErr error
	output []byte
}

// SetupMockCommand sets up a mock command executor
func SetupMockCommand(execCommand *func(string, ...string) *exec.Cmd) (MockCommand, func()) {
	mockState := NewCommandState()
	originalExecCommand := *execCommand

	*execCommand = func(command string, args ...string) *exec.Cmd {
		// Create a new command that will never actually execute
		cmd := exec.Command("/bin/true")
		cmd.Args = append([]string{command}, args...)

		// Handle mock output based on command
		output, err := mockState.execLXCCommand(command, args...)
		if err != nil {
			// For error cases, return a command that will fail
			failCmd := exec.Command("/bin/false")
			failCmd.Args = append([]string{command}, args...)
			return failCmd
		}

		// For success cases with output, store it in the environment
		if output != nil {
			cmd.Env = append(cmd.Env, "MOCK_OUTPUT="+string(output))
		}

		return cmd
	}

	// Return cleanup function
	cleanup := func() {
		*execCommand = originalExecCommand
	}

	return mockState, cleanup
}

// AddMockOutput adds a mock output for a specific command
func (m *CommandState) AddMockOutput(command string, output []byte) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.debug {
		fmt.Printf("DEBUG: Adding mock output for command: '%s'\n", command)
	}
	m.mockOutput[command] = output
	m.CalledCommands[command] = false
}

// WasCalled checks if a command was called
func (m *CommandState) WasCalled(cmd string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	called, exists := m.CalledCommands[cmd]
	return exists && called
}

// Command executes a mock command
func (m *CommandState) Command(cmd string, args ...string) ([]byte, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	fullCmd := cmd
	if len(args) > 0 {
		fullCmd = cmd + " " + strings.Join(args, " ")
	}

	m.CalledCommands[fullCmd] = true

	if output, exists := m.mockOutput[fullCmd]; exists {
		return output, nil
	}

	return []byte{}, nil
}

// MockCommand represents a mock command executor
type MockCommand interface {
	Run(name string, args ...string) error
	Output(name string, args ...string) ([]byte, error)
	CombinedOutput(name string, args ...string) ([]byte, error)
	SetDebug(enabled bool)
	ContainerExists(containerName string) bool
	SetContainerState(containerName, state string) error
	RemoveContainer(containerName string) error
	AddContainer(containerName, state string) error
	AddMockOutput(command string, output []byte)
	WasCalled(cmd string) bool
	Command(cmd string, args ...string) ([]byte, error)
}

// Run executes a command with the given arguments
func (m *CommandState) Run(name string, args ...string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.debug {
		fmt.Printf("DEBUG: Mock command called: %s %s\n", name, strings.Join(args, " "))
	}

	// Record command execution
	m.commandHistory = append(m.commandHistory, struct {
		name string
		args []string
	}{name, args})

	// Handle lxc commands
	switch name {
	case "lxc-info":
		if len(args) >= 2 && args[0] == "-n" {
			containerName := args[1]
			if _, exists := m.ContainerStates[containerName]; exists {
				return nil
			}
			return fmt.Errorf("container does not exist")
		}
		return fmt.Errorf("invalid arguments")

	case "lxc-destroy":
		if len(args) >= 2 && args[0] == "-n" {
			containerName := args[1]
			delete(m.ContainerStates, containerName)
			return nil
		}
		return fmt.Errorf("invalid arguments")

	case "lxc-start":
		if len(args) >= 2 && args[0] == "-n" {
			containerName := args[1]
			if _, exists := m.ContainerStates[containerName]; exists {
				m.ContainerStates[containerName] = "RUNNING"
				return nil
			}
			return fmt.Errorf("container does not exist")
		}
		return fmt.Errorf("invalid arguments")

	case "lxc-stop":
		if len(args) >= 2 && args[0] == "-n" {
			containerName := args[1]
			if _, exists := m.ContainerStates[containerName]; exists {
				m.ContainerStates[containerName] = "STOPPED"
				return nil
			}
			return fmt.Errorf("container does not exist")
		}
		return fmt.Errorf("invalid arguments")
	}

	return nil
}

// Output executes a command and returns its output
func (m *CommandState) Output(name string, args ...string) ([]byte, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.debug {
		fmt.Printf("DEBUG: Mock command called: %s %s\n", name, strings.Join(args, " "))
	}

	// Record command execution
	m.commandHistory = append(m.commandHistory, struct {
		name string
		args []string
	}{name, args})

	// Handle lxc commands
	switch name {
	case "lxc-info":
		if len(args) >= 2 && args[0] == "-n" {
			containerName := args[1]
			if state, exists := m.ContainerStates[containerName]; exists {
				return []byte(fmt.Sprintf("Name: %s\nState: %s\n", containerName, state)), nil
			}
			return []byte("container does not exist\n"), fmt.Errorf("exit status 1")
		}
		return []byte{}, fmt.Errorf("invalid arguments")
	}

	return []byte{}, nil
}

// CombinedOutput executes a command and returns its combined output
func (m *CommandState) CombinedOutput(name string, args ...string) ([]byte, error) {
	return m.Output(name, args...)
}

// NewCommandState creates a new CommandState
func NewCommandState() *CommandState {
	return &CommandState{
		ContainerStates: make(map[string]string),
		mockOutput:      make(map[string][]byte),
		CalledCommands:  make(map[string]bool),
	}
}
