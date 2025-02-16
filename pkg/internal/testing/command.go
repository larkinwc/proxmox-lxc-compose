// Package testutil provides test utilities for container tests
package testutil

import (
	"os/exec"
)

// ExecCommand executes a command and returns a Command struct
func ExecCommand(name string, args ...string) *exec.Cmd {
	return exec.Command(name, args...)
}
