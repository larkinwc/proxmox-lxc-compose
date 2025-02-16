package container

import "os/exec"

// ExecCommand is a variable that holds the exec.Command function.
// This allows us to replace it with a mock during testing.
var ExecCommand = exec.Command
