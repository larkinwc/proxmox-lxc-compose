package testutil

import (
	"os"
	"os/exec"
)

// MockCommandState tracks the state of mock commands
type MockCommandState struct {
	Name string
	Args []string
}

// SetupMockCommand sets up command mocking and returns a cleanup function
func SetupMockCommand(execCommand *func(string, ...string) *exec.Cmd) (*MockCommandState, func()) {
	mock := &MockCommandState{}
	oldExec := *execCommand

	*execCommand = func(name string, args ...string) *exec.Cmd {
		mock.Name = name
		mock.Args = args

		// For lxc-info, return a mock response based on the container name
		if name == "lxc-info" {
			script := `#!/bin/sh
echo "State: RUNNING"
exit 0
`
			tmpfile, err := os.CreateTemp("", "mock-lxc-info")
			if err != nil {
				panic(err)
			}
			if err := os.WriteFile(tmpfile.Name(), []byte(script), 0755); err != nil {
				panic(err)
			}
			return exec.Command("sh", tmpfile.Name())
		}

		// For lxc-attach, simulate log following
		if name == "lxc-attach" {
			script := `#!/bin/sh
echo "Log line 1"
echo "Log line 2"
sleep 0.1
`
			tmpfile, err := os.CreateTemp("", "mock-lxc-attach")
			if err != nil {
				panic(err)
			}
			if err := os.WriteFile(tmpfile.Name(), []byte(script), 0755); err != nil {
				panic(err)
			}
			return exec.Command("sh", tmpfile.Name())
		}

		return exec.Command("echo", "mock")
	}

	return mock, func() {
		*execCommand = oldExec
	}
}
