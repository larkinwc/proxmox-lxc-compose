// Package mock provides mocking functionality for testing
package mock

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"proxmox-lxc-compose/pkg/config"
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
			return exec.Command("sh", "-c", fmt.Sprintf("echo 'Container does not exist' >&2; exit 2"))
		}

		state, exists := mock.ContainerStates[containerName]
		if !exists {
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
			return exec.Command("sh", "-c", fmt.Sprintf("echo 'Container does not exist' >&2; exit 2"))
		}

		if mock.debug {
			fmt.Printf("DEBUG: Current state for container %s: %s\n", containerName, state)
		}

		switch name {
		case "lxc-info":
			return exec.Command("sh", "-c", fmt.Sprintf("echo 'State: %s'", state))
		case "lxc-freeze":
			if strings.ToUpper(state) != "RUNNING" {
				return exec.Command("sh", "-c", fmt.Sprintf("echo 'Container is not in a valid state for pausing (current state: %s)' >&2; exit 1", state))
			}
			if err := mock.SetContainerState(containerName, "FROZEN"); err != nil {
				return exec.Command("sh", "-c", fmt.Sprintf("echo 'Failed to update state: %v' >&2; exit 1", err))
			}
			return exec.Command("sh", "-c", "true")

		case "lxc-unfreeze":
			if strings.ToUpper(state) != "FROZEN" {
				return exec.Command("sh", "-c", fmt.Sprintf("echo 'Container is not in a valid state for resuming (current state: %s)' >&2; exit 1", state))
			}
			if err := mock.SetContainerState(containerName, "RUNNING"); err != nil {
				return exec.Command("sh", "-c", fmt.Sprintf("echo 'Failed to update state: %v' >&2; exit 1", err))
			}
			return exec.Command("sh", "-c", "true")

		case "lxc-start":
			if strings.ToUpper(state) != "STOPPED" {
				return exec.Command("sh", "-c", fmt.Sprintf("echo 'Container is not in a valid state for starting (current state: %s)' >&2; exit 1", state))
			}
			if err := mock.SetContainerState(containerName, "RUNNING"); err != nil {
				return exec.Command("sh", "-c", fmt.Sprintf("echo 'Failed to update state: %v' >&2; exit 1", err))
			}
			return exec.Command("sh", "-c", "true")

		case "lxc-stop":
			if strings.ToUpper(state) != "RUNNING" && strings.ToUpper(state) != "FROZEN" {
				return exec.Command("sh", "-c", fmt.Sprintf("echo 'Container is not in a valid state for stopping (current state: %s)' >&2; exit 1", state))
			}
			if err := mock.SetContainerState(containerName, "STOPPED"); err != nil {
				return exec.Command("sh", "-c", fmt.Sprintf("echo 'Failed to update state: %v' >&2; exit 1", err))
			}
			return exec.Command("sh", "-c", "true")

		case "lxc-attach":
			if strings.Contains(strings.Join(args, " "), "tail -f") {
				// Mock streaming log output
				return exec.Command("sh", "-c", `
					echo "2025-02-13 17:02:36.123 Container started"
					echo "2025-02-13 17:02:37.456 Service initialized"
					sleep 0.1
					echo "2025-02-13 17:02:38.789 Ready to accept connections"
					sleep 0.1
				`)
			}
		}

		// Return success for any command at this point
		return exec.Command("sh", "-c", "true")
	}

	return mock, func() {
		if mock.debug {
			fmt.Printf("DEBUG: Cleaning up %d temporary files\n", len(mock.tmpFiles))
		}
		for _, file := range mock.tmpFiles {
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

	// Update state file
	if configPath, ok := os.LookupEnv("CONTAINER_CONFIG_PATH"); ok {
		containerPath := filepath.Join(configPath, containerName)
		if err := os.MkdirAll(containerPath, 0755); err != nil {
			return err
		}

		statePath := filepath.Join(configPath, "state")
		if err := os.MkdirAll(statePath, 0755); err != nil {
			return err
		}

		stateFilePath := filepath.Join(statePath, containerName+".json")

		// Try to read existing state file
		var stateData struct {
			Name          string            `json:"name"`
			CreatedAt     time.Time         `json:"created_at"`
			LastStartedAt *time.Time        `json:"last_started_at,omitempty"`
			LastStoppedAt *time.Time        `json:"last_stopped_at,omitempty"`
			Config        *config.Container `json:"config"`
			Status        string            `json:"status"`
		}

		if data, err := os.ReadFile(stateFilePath); err == nil {
			if err := json.Unmarshal(data, &stateData); err == nil {
				// Preserve existing data but update status
				stateData.Status = strings.ToUpper(state)
				now := time.Now()
				switch strings.ToUpper(state) {
				case "RUNNING":
					stateData.LastStartedAt = &now
				case "STOPPED":
					stateData.LastStoppedAt = &now
				}
			}
		}

		// If no existing state or unmarshal failed, create new state
		if stateData.Name == "" {
			now := time.Now()
			stateData = struct {
				Name          string            `json:"name"`
				CreatedAt     time.Time         `json:"created_at"`
				LastStartedAt *time.Time        `json:"last_started_at,omitempty"`
				LastStoppedAt *time.Time        `json:"last_stopped_at,omitempty"`
				Config        *config.Container `json:"config"`
				Status        string            `json:"status"`
			}{
				Name:      containerName,
				CreatedAt: now,
				Status:    strings.ToUpper(state),
				Config:    &config.Container{},
			}

			switch strings.ToUpper(state) {
			case "RUNNING":
				stateData.LastStartedAt = &now
			case "STOPPED":
				stateData.LastStoppedAt = &now
			}
		}

		data, err := json.MarshalIndent(stateData, "", "  ")
		if err != nil {
			return err
		}

		if err := os.WriteFile(stateFilePath, data, 0644); err != nil {
			return err
		}
	}

	if m.debug {
		fmt.Printf("DEBUG: State updated successfully to %s\n", state)
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
			}
			return
		}

		// Create empty config.yaml file
		configFile := filepath.Join(containerPath, "config.yaml")
		if err := os.WriteFile(configFile, []byte{}, 0644); err != nil {
			if m.debug {
				fmt.Printf("DEBUG: Failed to create config file: %v\n", err)
			}
		}

		// Create or update state file
		if err := m.SetContainerState(containerName, state); err != nil {
			if m.debug {
				fmt.Printf("DEBUG: Failed to set container state: %v\n", err)
			}
		}
	}
}
