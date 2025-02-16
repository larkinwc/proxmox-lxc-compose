// Package testing_internal provides test utilities for container tests
package testing_internal

import (
	"os/exec"
)

// ExecCommand executes a command and returns a Command struct
func ExecCommand(name string, args ...string) *exec.Cmd {
	return exec.Command(name, args...)
}
