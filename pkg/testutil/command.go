package testutil

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"proxmox-lxc-compose/pkg/config"
	"strings"
	"time"
)

// MockCommandState tracks the state of mock commands
type MockCommandState struct {
	Name            string
	Args            []string
	ContainerStates map[string]string
	tmpFiles        []string
	debug           bool
	commandHistory  []struct {
		name string
		args []string
	}
}

// CommandWasCalled checks if a command was called with the given name and arguments
func (m *MockCommandState) CommandWasCalled(name string, args ...string) bool {
	for _, cmd := range m.commandHistory {
		if cmd.name != name {
			continue
		}
		if len(cmd.args) != len(args) {
			continue
		}
		match := true
		for i, arg := range args {
			if cmd.args[i] != arg {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}

// SetupMockCommand sets up command mocking and returns a cleanup function
func SetupMockCommand(execCommand *func(string, ...string) *exec.Cmd) (*MockCommandState, func()) {
	mock := &MockCommandState{
		ContainerStates: make(map[string]string),
		tmpFiles:        make([]string, 0),
		debug:           true,
		commandHistory: make([]struct {
			name string
			args []string
		}, 0),
	}
	oldExec := *execCommand

	*execCommand = func(name string, args ...string) *exec.Cmd {
		if mock.debug {
			fmt.Printf("DEBUG: Mock command called: %s %s\n", name, strings.Join(args, " "))
			fmt.Printf("DEBUG: Current container states: %v\n", mock.ContainerStates)
			fmt.Printf("DEBUG: Command history: %v\n", mock.commandHistory)
		}

		mock.Name = name
		mock.Args = args
		mock.commandHistory = append(mock.commandHistory, struct {
			name string
			args []string
		}{name, args})

		// For all commands, first check if container exists
		if len(args) < 2 || args[0] != "-n" {
			if mock.debug {
				fmt.Printf("DEBUG: Invalid args: %v\n", args)
			}
			return exec.Command("sh", "-c", fmt.Sprintf("echo 'Invalid arguments' >&2; exit 2"))
		}

		containerName := args[1]
		if containerName == "nonexistent" {
			if mock.debug {
				fmt.Printf("DEBUG: Nonexistent container: %s\n", containerName)
			}
			return exec.Command("sh", "-c", fmt.Sprintf("echo 'Container does not exist' >&2; exit 2"))
		}

		state, exists := mock.ContainerStates[containerName]
		if !exists {
			// Check if container directory exists in configPath
			if configPath, ok := os.LookupEnv("CONTAINER_CONFIG_PATH"); ok {
				containerPath := filepath.Join(configPath, containerName)
				if _, err := os.Stat(containerPath); err == nil {
					state = "STOPPED"
					exists = true
					mock.ContainerStates[containerName] = state
				}
			}
		}

		if !exists {
			if mock.debug {
				fmt.Printf("DEBUG: Container not found: %s\n", containerName)
			}
			return exec.Command("sh", "-c", fmt.Sprintf("echo 'Container does not exist' >&2; exit 2"))
		}

		if mock.debug {
			fmt.Printf("DEBUG: Current state for container %s: %s\n", containerName, state)
		}

		switch name {
		case "lxc-freeze":
			if strings.ToUpper(state) != "RUNNING" {
				if mock.debug {
					fmt.Printf("DEBUG: Cannot freeze non-running container (state: %s)\n", state)
				}
				return exec.Command("sh", "-c", fmt.Sprintf("echo 'Container is not in a valid state for pausing (current state: %s)' >&2; exit 1", state))
			}
			mock.ContainerStates[containerName] = "FROZEN"
			if err := mock.SetContainerState(containerName, "FROZEN"); err != nil {
				if mock.debug {
					fmt.Printf("DEBUG: Failed to update state: %v\n", err)
				}
				return exec.Command("sh", "-c", fmt.Sprintf("echo 'Failed to update state: %v' >&2; exit 1", err))
			}

		case "lxc-unfreeze":
			if strings.ToUpper(state) != "FROZEN" {
				if mock.debug {
					fmt.Printf("DEBUG: Cannot unfreeze non-frozen container (state: %s)\n", state)
				}
				return exec.Command("sh", "-c", fmt.Sprintf("echo 'Container is not in a valid state for resuming (current state: %s)' >&2; exit 1", state))
			}
			mock.ContainerStates[containerName] = "RUNNING"
			if err := mock.SetContainerState(containerName, "RUNNING"); err != nil {
				if mock.debug {
					fmt.Printf("DEBUG: Failed to update state: %v\n", err)
				}
				return exec.Command("sh", "-c", fmt.Sprintf("echo 'Failed to update state: %v' >&2; exit 1", err))
			}

		case "lxc-start":
			if strings.ToUpper(state) != "STOPPED" {
				if mock.debug {
					fmt.Printf("DEBUG: Cannot start non-stopped container (state: %s)\n", state)
				}
				return exec.Command("sh", "-c", fmt.Sprintf("echo 'Container is not in a valid state for starting (current state: %s)' >&2; exit 1", state))
			}
			mock.ContainerStates[containerName] = "RUNNING"
			if err := mock.SetContainerState(containerName, "RUNNING"); err != nil {
				if mock.debug {
					fmt.Printf("DEBUG: Failed to update state: %v\n", err)
				}
				return exec.Command("sh", "-c", fmt.Sprintf("echo 'Failed to update state: %v' >&2; exit 1", err))
			}

		case "lxc-stop":
			if strings.ToUpper(state) != "RUNNING" && strings.ToUpper(state) != "FROZEN" {
				if mock.debug {
					fmt.Printf("DEBUG: Cannot stop non-running/frozen container (state: %s)\n", state)
				}
				return exec.Command("sh", "-c", fmt.Sprintf("echo 'Container is not in a valid state for stopping (current state: %s)' >&2; exit 1", state))
			}
			mock.ContainerStates[containerName] = "STOPPED"
			if err := mock.SetContainerState(containerName, "STOPPED"); err != nil {
				if mock.debug {
					fmt.Printf("DEBUG: Failed to update state: %v\n", err)
				}
				return exec.Command("sh", "-c", fmt.Sprintf("echo 'Failed to update state: %v' >&2; exit 1", err))
			}
		}

		// Return success for any command at this point
		return exec.Command("sh", "-c", "true")
	}

	return mock, func() {
		if mock.debug {
			fmt.Printf("DEBUG: Cleaning up %d temporary files\n", len(mock.tmpFiles))
		}
		// Clean up all temporary files
		for _, file := range mock.tmpFiles {
			if mock.debug {
				fmt.Printf("DEBUG: Removing temporary file: %s\n", file)
			}
			os.Remove(file)
		}
		*execCommand = oldExec
	}
}

// SetContainerState allows tests to set the state of a container
func (m *MockCommandState) SetContainerState(containerName, state string) error {
	if m.debug {
		fmt.Printf("DEBUG: SetContainerState called for %s with state %s\n", containerName, state)
		fmt.Printf("DEBUG: Current container states before update: %v\n", m.ContainerStates)
	}

	if containerName == "nonexistent" {
		return fmt.Errorf("container %s does not exist", containerName)
	}

	// Update in-memory state
	m.ContainerStates[containerName] = strings.ToUpper(state)
	if m.debug {
		fmt.Printf("DEBUG: Updated container states: %v\n", m.ContainerStates)
	}

	// Update state file
	if configPath, ok := os.LookupEnv("CONTAINER_CONFIG_PATH"); ok {
		if m.debug {
			fmt.Printf("DEBUG: Found config path: %s\n", configPath)
		}

		// Create container directory if it doesn't exist
		containerPath := filepath.Join(configPath, containerName)
		if err := os.MkdirAll(containerPath, 0755); err != nil {
			if m.debug {
				fmt.Printf("DEBUG: Failed to create container directory: %v\n", err)
			}
			return err
		}

		// Create state directory if it doesn't exist
		statePath := filepath.Join(configPath, "state")
		if err := os.MkdirAll(statePath, 0755); err != nil {
			if m.debug {
				fmt.Printf("DEBUG: Failed to create state directory: %v\n", err)
			}
			return err
		}

		// Create state file
		stateFilePath := filepath.Join(statePath, containerName+".json")
		stateData := struct {
			Name          string            `json:"name"`
			CreatedAt     string            `json:"created_at"`
			LastStartedAt string            `json:"last_started_at,omitempty"`
			LastStoppedAt string            `json:"last_stopped_at,omitempty"`
			Config        *config.Container `json:"config"`
			Status        string            `json:"status"`
		}{
			Name:      containerName,
			CreatedAt: time.Now().Format(time.RFC3339),
			Status:    strings.ToUpper(state),
			Config:    &config.Container{},
		}

		data, err := json.MarshalIndent(stateData, "", "  ")
		if err != nil {
			if m.debug {
				fmt.Printf("DEBUG: Failed to marshal state data: %v\n", err)
			}
			return err
		}

		// Write state file
		if err := os.WriteFile(stateFilePath, data, 0644); err != nil {
			if m.debug {
				fmt.Printf("DEBUG: Failed to write state file: %v\n", err)
			}
			return err
		}

		if m.debug {
			fmt.Printf("DEBUG: Successfully wrote state file: %s\n", stateFilePath)
			fmt.Printf("DEBUG: State file contents: %s\n", string(data))
		}
	} else if m.debug {
		fmt.Printf("DEBUG: CONTAINER_CONFIG_PATH not set\n")
	}

	return nil
}

// AddContainer adds a container to the mock state
func (m *MockCommandState) AddContainer(containerName, state string) {
	if m.debug {
		fmt.Printf("DEBUG: Adding container %s with state %s\n", containerName, state)
	}
	m.ContainerStates[containerName] = strings.ToUpper(state)

	// Create container directory in configPath if it doesn't exist
	if configPath, ok := os.LookupEnv("CONTAINER_CONFIG_PATH"); ok {
		containerPath := filepath.Join(configPath, containerName)
		if err := os.MkdirAll(containerPath, 0755); err != nil {
			if m.debug {
				fmt.Printf("DEBUG: Failed to create container directory: %v\n", err)
				return
			}
		}

		// Create state directory if it doesn't exist
		statePath := filepath.Join(configPath, "state")
		if err := os.MkdirAll(statePath, 0755); err != nil {
			if m.debug {
				fmt.Printf("DEBUG: Failed to create state directory: %v\n", err)
				return
			}
		}

		// Create or update state file
		stateData := struct {
			Name          string            `json:"name"`
			CreatedAt     string            `json:"created_at"`
			LastStartedAt string            `json:"last_started_at,omitempty"`
			LastStoppedAt string            `json:"last_stopped_at,omitempty"`
			Config        *config.Container `json:"config"`
			Status        string            `json:"status"`
		}{
			Name:      containerName,
			CreatedAt: time.Now().Format(time.RFC3339),
			Status:    strings.ToUpper(state),
			Config:    &config.Container{},
		}

		// Update timestamps based on state
		now := time.Now().Format(time.RFC3339)
		if strings.ToUpper(state) == "RUNNING" {
			stateData.LastStartedAt = now
		} else if strings.ToUpper(state) == "STOPPED" {
			stateData.LastStoppedAt = now
		}

		data, err := json.MarshalIndent(stateData, "", "  ")
		if err != nil {
			if m.debug {
				fmt.Printf("DEBUG: Failed to marshal state data: %v\n", err)
			}
			return
		}

		stateFile := filepath.Join(statePath, containerName+".json")
		if err := os.WriteFile(stateFile, data, 0644); err != nil {
			if m.debug {
				fmt.Printf("DEBUG: Failed to write state file: %v\n", err)
			}
			return
		}

		if m.debug {
			fmt.Printf("DEBUG: Successfully wrote state file: %s\n", stateFile)
		}
	}
}

// SetDebug enables or disables debug logging
func (m *MockCommandState) SetDebug(enabled bool) {
	m.debug = enabled
}

// GetContainerState returns the state of a container
func (m *MockCommandState) GetContainerState(containerName string) (string, bool) {
	state, exists := m.ContainerStates[containerName]
	return state, exists
}

// RemoveContainer removes a container from the mock state
func (m *MockCommandState) RemoveContainer(containerName string) {
	delete(m.ContainerStates, containerName)
}

// ContainerExists checks if a container exists in the mock state
func (m *MockCommandState) ContainerExists(containerName string) bool {
	if containerName == "nonexistent" {
		return false
	}

	_, exists := m.ContainerStates[containerName]
	return exists
}

// MockCommandExecutor mocks command execution for testing
type MockCommandExecutor struct {
	commands   map[string][]byte
	errorCmds  map[string]error
	actualExec bool
}

// NewMockCommandExecutor creates a new mock command executor
func NewMockCommandExecutor() *MockCommandExecutor {
	return &MockCommandExecutor{
		commands:  make(map[string][]byte),
		errorCmds: make(map[string]error),
	}
}

// AddMockCommand adds a mock command response
func (m *MockCommandExecutor) AddMockCommand(cmd string, output []byte) {
	m.commands[cmd] = output
}

// AddMockError adds a mock command error
func (m *MockCommandExecutor) AddMockError(cmd string, err error) {
	m.errorCmds[cmd] = err
}

// SetActualExecution enables actual command execution for unmocked commands
func (m *MockCommandExecutor) SetActualExecution(enabled bool) {
	m.actualExec = enabled
}

// Command creates a mocked exec.Command
func (m *MockCommandExecutor) Command(name string, args ...string) *exec.Cmd {
	cmdStr := name
	if len(args) > 0 {
		cmdStr = name + " " + strings.Join(args, " ")
	}

	if err, ok := m.errorCmds[cmdStr]; ok {
		// Create a failing command that writes error to stderr
		cmd := exec.Command("sh", "-c", "exit 1")
		cmd.Stderr = bytes.NewBuffer([]byte(err.Error()))
		return cmd
	}

	// For successful commands, create a shell command that outputs the mock data
	if output, ok := m.commands[cmdStr]; ok {
		cmd := exec.Command("sh", "-c", "cat")
		stdin, _ := cmd.StdinPipe()
		go func() {
			defer stdin.Close()
			stdin.Write(output)
		}()
		return cmd
	}

	if m.actualExec {
		return exec.Command(name, args...)
	}

	// Default: success with no output
	return exec.Command("true")
}
