package testutil

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
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
		debug:           true, // Enable debug by default
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
		}

		mock.Name = name
		mock.Args = args
		mock.commandHistory = append(mock.commandHistory, struct {
			name string
			args []string
		}{name, args})

		// For lxc-info, return a mock response based on the container name
		if name == "lxc-info" {
			if mock.debug {
				fmt.Printf("DEBUG: Processing lxc-info command\n")
			}

			// Check if container name is provided
			if len(args) < 2 || args[0] != "-n" {
				if mock.debug {
					fmt.Printf("DEBUG: lxc-info failed: invalid args: %v\n", args)
				}
				return exec.Command("sh", "-c", fmt.Sprintf("echo 'Invalid arguments' >&2; exit 2"))
			}

			containerName := args[1]
			if mock.debug {
				fmt.Printf("DEBUG: Looking up container: %s\n", containerName)
			}

			if containerName == "nonexistent" {
				if mock.debug {
					fmt.Printf("DEBUG: lxc-info failed: nonexistent container: %s\n", containerName)
				}
				return exec.Command("sh", "-c", fmt.Sprintf("echo 'Container does not exist' >&2; exit 2"))
			}

			state, exists := mock.ContainerStates[containerName]
			if !exists {
				if mock.debug {
					fmt.Printf("DEBUG: lxc-info failed: container not found in state map: %s\n", containerName)
				}
				return exec.Command("sh", "-c", fmt.Sprintf("echo 'Container does not exist' >&2; exit 2"))
			}

			if mock.debug {
				fmt.Printf("DEBUG: Creating mock script for container %s with state %s\n", containerName, state)
			}

			// Create a temporary file for the script
			tmpfile, err := os.CreateTemp("", "mock-lxc-info-*.sh")
			if err != nil {
				if mock.debug {
					fmt.Printf("DEBUG: Failed to create temp file: %v\n", err)
				}
				panic(err)
			}
			mock.tmpFiles = append(mock.tmpFiles, tmpfile.Name())

			// Write the script content
			script := fmt.Sprintf(`#!/bin/sh
cat << EOF
Name: %s
State: %s
PID: 12345
IP: 192.168.1.100
CPU use: 1.23
Memory use: 123.45 MiB
KMem use: 12.34 MiB
Link: vethXYZ123
TX bytes: 1234567 bytes
RX bytes: 7654321 bytes
Total bytes: 8888888 bytes
EOF
exit 0
`, containerName, state)

			if err := os.WriteFile(tmpfile.Name(), []byte(script), 0755); err != nil {
				if mock.debug {
					fmt.Printf("DEBUG: Failed to write script: %v\n", err)
				}
				panic(err)
			}

			if mock.debug {
				fmt.Printf("DEBUG: Created script at %s:\n%s\n", tmpfile.Name(), script)
			}

			return exec.Command("sh", tmpfile.Name())
		}

		// For lxc-attach, simulate log following
		if name == "lxc-attach" {
			if mock.debug {
				fmt.Printf("DEBUG: Processing lxc-attach command\n")
			}

			// Check if container name is provided
			if len(args) < 2 || args[0] != "-n" {
				if mock.debug {
					fmt.Printf("DEBUG: lxc-attach failed: invalid args: %v\n", args)
				}
				return exec.Command("sh", "-c", "exit 2")
			}

			containerName := args[1]
			if containerName == "nonexistent" {
				if mock.debug {
					fmt.Printf("DEBUG: lxc-attach failed: nonexistent container: %s\n", containerName)
				}
				return exec.Command("sh", "-c", "exit 2")
			}

			if _, exists := mock.ContainerStates[containerName]; !exists {
				if mock.debug {
					fmt.Printf("DEBUG: lxc-attach failed: container not found in state map: %s\n", containerName)
				}
				return exec.Command("sh", "-c", "exit 2")
			}

			script := `#!/bin/sh
echo "New log line 1"
echo "New log line 2"
sleep 0.1
`
			tmpfile, err := os.CreateTemp("", "mock-lxc-attach-*.sh")
			if err != nil {
				if mock.debug {
					fmt.Printf("DEBUG: Failed to create temp file: %v\n", err)
				}
				panic(err)
			}
			mock.tmpFiles = append(mock.tmpFiles, tmpfile.Name())

			if err := os.WriteFile(tmpfile.Name(), []byte(script), 0755); err != nil {
				if mock.debug {
					fmt.Printf("DEBUG: Failed to write script: %v\n", err)
				}
				panic(err)
			}

			if mock.debug {
				fmt.Printf("DEBUG: Created lxc-attach script at %s:\n%s\n", tmpfile.Name(), script)
			}

			return exec.Command("sh", tmpfile.Name())
		}

		// For lxc-freeze and lxc-unfreeze, check container name and update state
		if name == "lxc-freeze" || name == "lxc-unfreeze" {
			if mock.debug {
				fmt.Printf("DEBUG: Processing %s command\n", name)
			}

			if len(args) < 2 || args[0] != "-n" {
				if mock.debug {
					fmt.Printf("DEBUG: %s failed: invalid args: %v\n", name, args)
				}
				return exec.Command("sh", "-c", fmt.Sprintf("echo 'Invalid arguments' >&2; exit 2"))
			}

			containerName := args[1]
			if containerName == "nonexistent" {
				if mock.debug {
					fmt.Printf("DEBUG: %s failed: nonexistent container: %s\n", name, containerName)
				}
				return exec.Command("sh", "-c", fmt.Sprintf("echo 'Container does not exist' >&2; exit 2"))
			}

			state, exists := mock.ContainerStates[containerName]
			if !exists {
				if mock.debug {
					fmt.Printf("DEBUG: %s failed: container not found in state map: %s\n", name, containerName)
				}
				return exec.Command("sh", "-c", fmt.Sprintf("echo 'Container does not exist' >&2; exit 2"))
			}

			// Update container state
			if name == "lxc-freeze" {
				if state != "RUNNING" {
					if mock.debug {
						fmt.Printf("DEBUG: %s failed: container is not running (current state: %s)\n", name, state)
					}
					return exec.Command("sh", "-c", fmt.Sprintf("echo 'Container is not running (current state: %s)' >&2; exit 1", state))
				}
				mock.ContainerStates[containerName] = "FROZEN"
				if mock.debug {
					fmt.Printf("DEBUG: %s succeeded: new state for %s: %s\n", name, containerName, mock.ContainerStates[containerName])
				}
				return exec.Command("sh", "-c", "exit 0")
			} else {
				if state != "FROZEN" {
					if mock.debug {
						fmt.Printf("DEBUG: %s failed: container is not frozen (current state: %s)\n", name, state)
					}
					return exec.Command("sh", "-c", fmt.Sprintf("echo 'Container is not frozen (current state: %s)' >&2; exit 1", state))
				}
				mock.ContainerStates[containerName] = "RUNNING"
				if mock.debug {
					fmt.Printf("DEBUG: %s succeeded: new state for %s: %s\n", name, containerName, mock.ContainerStates[containerName])
				}
				return exec.Command("sh", "-c", "exit 0")
			}
		}

		// For lxc-start and lxc-stop, check container name and update state
		if name == "lxc-start" || name == "lxc-stop" {
			if mock.debug {
				fmt.Printf("DEBUG: Processing %s command\n", name)
			}

			if len(args) < 2 || args[0] != "-n" {
				if mock.debug {
					fmt.Printf("DEBUG: %s failed: invalid args: %v\n", name, args)
				}
				return exec.Command("sh", "-c", fmt.Sprintf("echo 'Invalid arguments' >&2; exit 2"))
			}

			containerName := args[1]
			if containerName == "nonexistent" {
				if mock.debug {
					fmt.Printf("DEBUG: %s failed: nonexistent container: %s\n", name, containerName)
				}
				return exec.Command("sh", "-c", fmt.Sprintf("echo 'Container does not exist' >&2; exit 2"))
			}

			state, exists := mock.ContainerStates[containerName]
			if !exists {
				if mock.debug {
					fmt.Printf("DEBUG: %s failed: container not found in state map: %s\n", name, containerName)
				}
				return exec.Command("sh", "-c", fmt.Sprintf("echo 'Container does not exist' >&2; exit 2"))
			}

			// Update container state
			if name == "lxc-start" {
				if state != "STOPPED" {
					if mock.debug {
						fmt.Printf("DEBUG: %s failed: container is not stopped (current state: %s)\n", name, state)
					}
					return exec.Command("sh", "-c", fmt.Sprintf("echo 'Container is not stopped (current state: %s)' >&2; exit 1", state))
				}
				mock.ContainerStates[containerName] = "RUNNING"
				if mock.debug {
					fmt.Printf("DEBUG: %s succeeded: new state for %s: %s\n", name, containerName, mock.ContainerStates[containerName])
				}
				return exec.Command("sh", "-c", "exit 0")
			} else {
				if state != "RUNNING" && state != "FROZEN" {
					if mock.debug {
						fmt.Printf("DEBUG: %s failed: container is not running or frozen (current state: %s)\n", name, state)
					}
					return exec.Command("sh", "-c", fmt.Sprintf("echo 'Container is not running or frozen (current state: %s)' >&2; exit 1", state))
				}
				mock.ContainerStates[containerName] = "STOPPED"
				if mock.debug {
					fmt.Printf("DEBUG: %s succeeded: new state for %s: %s\n", name, containerName, mock.ContainerStates[containerName])
				}
				return exec.Command("sh", "-c", "exit 0")
			}
		}

		// For any other command, just return success
		if mock.debug {
			fmt.Printf("DEBUG: Unhandled command: %s %s\n", name, strings.Join(args, " "))
		}
		return exec.Command("sh", "-c", "exit 0")
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
func (m *MockCommandState) SetContainerState(containerName, state string) {
	if m.debug {
		fmt.Printf("DEBUG: Setting state for %s to %s\n", containerName, state)
	}
	m.ContainerStates[containerName] = state
}

// AddContainer adds a container to the mock state with the given state
func (m *MockCommandState) AddContainer(containerName, state string) {
	if m.debug {
		fmt.Printf("DEBUG: Adding container %s with state %s\n", containerName, state)
	}
	m.ContainerStates[containerName] = state
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
