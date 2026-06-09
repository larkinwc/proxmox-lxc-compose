package proxmox

import "os/exec"

// recordedCommand captures a single invocation of execCommand.
type recordedCommand struct {
	name string
	args []string
}

// mockExec replaces execCommand with one that records invocations and returns
// the supplied stdout/exit behavior. It returns the recorder and a restore fn.
func mockExec(stdout string, fail bool) (*[]recordedCommand, func()) {
	orig := execCommand
	var recorded []recordedCommand
	execCommand = func(name string, args ...string) *exec.Cmd {
		recorded = append(recorded, recordedCommand{name: name, args: args})
		if fail {
			// `false` exits non-zero with no output.
			return exec.Command("false")
		}
		// `printf` emits the canned stdout without a trailing newline issue.
		return exec.Command("printf", "%s", stdout)
	}
	return &recorded, func() { execCommand = orig }
}
