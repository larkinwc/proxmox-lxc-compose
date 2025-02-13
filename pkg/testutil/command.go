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
}

// SetupMockCommand sets up command mocking and returns a cleanup function
func SetupMockCommand(execCommand *func(string, ...string) *exec.Cmd) (*MockCommandState, func()) {
	mock := &MockCommandState{
		ContainerStates: make(map[string]string),
		tmpFiles:        make([]string, 0),
		debug:           true, // Enable debug by default
	}
	oldExec := *execCommand

	*execCommand = func(name string, args ...string) *exec.Cmd {
		if mock.debug {
			fmt.Printf("DEBUG: Mock command called: %s %s\n", name, strings.Join(args, " "))
			fmt.Printf("DEBUG: Current container states: %v\n", mock.ContainerStates)
		}

		mock.Name = name
		mock.Args = args

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
				return exec.Command("false")
			}

			containerName := args[1]
			if mock.debug {
				fmt.Printf("DEBUG: Looking up container: %s\n", containerName)
			}

			if containerName == "nonexistent" {
				if mock.debug {
					fmt.Printf("DEBUG: lxc-info failed: nonexistent container: %s\n", containerName)
				}
				return exec.Command("false")
			}

			state, exists := mock.ContainerStates[containerName]
			if !exists {
				if mock.debug {
					fmt.Printf("DEBUG: lxc-info failed: container not found in state map: %s\n", containerName)
					fmt.Printf("DEBUG: Available containers: %v\n", mock.ContainerStates)
				}
				return exec.Command("false")
			}

			if mock.debug {
				fmt.Printf("DEBUG: Creating mock script for container %s with state %s\n", containerName, state)
			}

			script := fmt.Sprintf(`#!/bin/sh
echo "Name: %s"
echo "State: %s"
echo "PID: 12345"
`, containerName, state)

			tmpfile, err := os.CreateTemp("", "mock-lxc-info-*.sh")
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
				return exec.Command("false")
			}

			containerName := args[1]
			if containerName == "nonexistent" {
				if mock.debug {
					fmt.Printf("DEBUG: lxc-attach failed: nonexistent container: %s\n", containerName)
				}
				return exec.Command("false")
			}

			if _, exists := mock.ContainerStates[containerName]; !exists {
				if mock.debug {
					fmt.Printf("DEBUG: lxc-attach failed: container not found in state map: %s\n", containerName)
				}
				return exec.Command("false")
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
				return exec.Command("false")
			}

			containerName := args[1]
			if containerName == "nonexistent" {
				if mock.debug {
					fmt.Printf("DEBUG: %s failed: nonexistent container: %s\n", name, containerName)
				}
				return exec.Command("false")
			}

			if _, exists := mock.ContainerStates[containerName]; !exists {
				if mock.debug {
					fmt.Printf("DEBUG: %s failed: container not found in state map: %s\n", name, containerName)
				}
				return exec.Command("false")
			}

			// Update container state
			if name == "lxc-freeze" {
				mock.ContainerStates[containerName] = "FROZEN"
			} else {
				mock.ContainerStates[containerName] = "RUNNING"
			}

			if mock.debug {
				fmt.Printf("DEBUG: %s succeeded: new state for %s: %s\n", name, containerName, mock.ContainerStates[containerName])
			}

			return exec.Command("true")
		}

		// For any other command, just echo mock
		if mock.debug {
			fmt.Printf("DEBUG: Unhandled command: %s %s\n", name, strings.Join(args, " "))
		}
		return exec.Command("echo", strings.Join(append([]string{name}, args...), " "))
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

// AddContainer adds a new container with the given state
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
