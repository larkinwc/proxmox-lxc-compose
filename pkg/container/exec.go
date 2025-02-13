package container

import "os/exec"

// execCommand is a variable that holds the exec.Command function.
// This allows us to replace it with a mock during testing.
var execCommand = exec.Command
