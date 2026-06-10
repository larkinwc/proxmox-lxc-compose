package proxmox

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

// execCommand is the indirection point used to run external commands. Tests
// replace it to avoid invoking the real pct binary.
var execCommand = exec.Command

// defaultConfDir is where Proxmox stores per-container LXC configuration.
const defaultConfDir = "/etc/pve/lxc"

// PCTBackend implements Backend using the Proxmox `pct` CLI. It must run on a
// Proxmox node (pct talks to the local cluster filesystem).
type PCTBackend struct {
	// binary is the pct executable name/path (default "pct").
	binary string
	// confDir is the directory holding <vmid>.conf files. Overridable in tests.
	confDir string
}

// NewPCTBackend returns a Backend backed by the pct CLI.
func NewPCTBackend() *PCTBackend {
	return &PCTBackend{binary: "pct", confDir: defaultConfDir}
}

// run executes a pct subcommand and returns combined output.
func (b *PCTBackend) run(args ...string) ([]byte, error) {
	cmd := execCommand(b.binary, args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return out, fmt.Errorf("pct %s failed: %w: %s",
			strings.Join(args, " "), err, strings.TrimSpace(string(out)))
	}
	return out, nil
}

// createArgs builds the argument list for `pct create`.
func createArgs(vmid int, o CreateOptions) []string {
	args := []string{"create", strconv.Itoa(vmid), o.OSTemplate}

	if o.Hostname != "" {
		args = append(args, "--hostname", o.Hostname)
	}

	// rootfs: storage:sizeInGB
	if o.Storage != "" {
		size := o.RootFSSize
		if size <= 0 {
			size = 8
		}
		args = append(args, "--rootfs", fmt.Sprintf("%s:%d", o.Storage, size))
	}

	if o.Cores > 0 {
		args = append(args, "--cores", strconv.Itoa(o.Cores))
	}
	if o.CPULimit > 0 {
		args = append(args, "--cpulimit", strconv.Itoa(o.CPULimit))
	}
	if o.CPUUnits > 0 {
		args = append(args, "--cpuunits", strconv.Itoa(o.CPUUnits))
	}
	if o.MemoryMB > 0 {
		args = append(args, "--memory", strconv.Itoa(o.MemoryMB))
	}
	if o.SwapMB > 0 {
		args = append(args, "--swap", strconv.Itoa(o.SwapMB))
	}

	// Unprivileged flag is always emitted explicitly for determinism.
	if o.Unprivileged {
		args = append(args, "--unprivileged", "1")
	} else {
		args = append(args, "--unprivileged", "0")
	}

	if o.Features != "" {
		args = append(args, "--features", o.Features)
	}
	if len(o.Nameservers) > 0 {
		args = append(args, "--nameserver", strings.Join(o.Nameservers, " "))
	}
	if o.SearchDomain != "" {
		args = append(args, "--searchdomain", o.SearchDomain)
	}

	for i, net := range o.Nets {
		args = append(args, fmt.Sprintf("--net%d", i), net)
	}
	for i, mp := range o.Mounts {
		args = append(args, fmt.Sprintf("--mp%d", i), mp)
	}

	for k, v := range o.Extra {
		args = append(args, "--"+k, v)
	}

	if o.Start {
		args = append(args, "--start", "1")
	}

	return args
}

// Create provisions a new container.
func (b *PCTBackend) Create(vmid int, opts CreateOptions) error {
	if opts.OSTemplate == "" {
		return fmt.Errorf("OS template is required to create a container")
	}
	_, err := b.run(createArgs(vmid, opts)...)
	return err
}

// Start boots a stopped container.
func (b *PCTBackend) Start(vmid int) error {
	_, err := b.run("start", strconv.Itoa(vmid))
	return err
}

// Stop forcibly stops a container.
func (b *PCTBackend) Stop(vmid int) error {
	_, err := b.run("stop", strconv.Itoa(vmid))
	return err
}

// Shutdown gracefully shuts down a container, falling back to a hard stop if
// it doesn't exit within the timeout. The fallback matters for converted OCI
// images whose PID 1 is the application process (e.g. nginx) rather than a real
// init that handles shutdown signals.
func (b *PCTBackend) Shutdown(vmid int) error {
	_, err := b.run("shutdown", strconv.Itoa(vmid), "--forceStop", "1", "--timeout", "10")
	return err
}

// Suspend pauses a running container.
func (b *PCTBackend) Suspend(vmid int) error {
	_, err := b.run("suspend", strconv.Itoa(vmid))
	return err
}

// Resume unfreezes a suspended container.
func (b *PCTBackend) Resume(vmid int) error {
	_, err := b.run("resume", strconv.Itoa(vmid))
	return err
}

// Destroy removes a container.
func (b *PCTBackend) Destroy(vmid int) error {
	_, err := b.run("destroy", strconv.Itoa(vmid))
	return err
}

// SetInitCommand sets the container's init command by appending a raw
// `lxc.init.cmd` line to its Proxmox config file. `pct` has no native flag for
// this, so the key is written directly into /etc/pve/lxc/<vmid>.conf. This is
// how a converted OCI image's entrypoint/command is reproduced under LXC.
//
// Passing an empty path is a no-op (the distro's normal init is used).
func (b *PCTBackend) SetInitCommand(vmid int, initPath string) error {
	if initPath == "" {
		return nil
	}
	confPath := filepath.Join(b.confDir, strconv.Itoa(vmid)+".conf")
	data, err := os.ReadFile(confPath)
	if err != nil {
		return fmt.Errorf("failed to read container config %s: %w", confPath, err)
	}

	line := "lxc.init.cmd: " + initPath
	// Replace an existing lxc.init.cmd line (which may have a different value)
	// rather than appending a second, conflicting key.
	lines := strings.Split(string(data), "\n")
	replaced := false
	for i, existing := range lines {
		trimmed := strings.TrimSpace(existing)
		if trimmed == line {
			return nil
		}
		if strings.HasPrefix(trimmed, "lxc.init.cmd:") {
			lines[i] = line
			replaced = true
		}
	}
	if !replaced {
		lines = append(lines, line)
	}

	content := strings.Join(lines, "\n")
	if !strings.HasSuffix(content, "\n") {
		content += "\n"
	}
	if err := os.WriteFile(confPath, []byte(content), 0640); err != nil {
		return fmt.Errorf("failed to write init command to %s: %w", confPath, err)
	}
	return nil
}

// Status returns the runtime status of a container by parsing `pct status`.
func (b *PCTBackend) Status(vmid int) (Status, error) {
	out, err := b.run("status", strconv.Itoa(vmid))
	if err != nil {
		return StatusUnknown, err
	}
	return parseStatus(string(out)), nil
}

// parseStatus interprets the output of `pct status <vmid>` ("status: running").
func parseStatus(out string) Status {
	line := strings.TrimSpace(out)
	line = strings.TrimPrefix(line, "status:")
	line = strings.TrimSpace(line)
	switch strings.ToLower(line) {
	case "running":
		return StatusRunning
	case "stopped":
		return StatusStopped
	case "suspended", "paused", "frozen":
		return StatusPaused
	default:
		return StatusUnknown
	}
}

// List returns all containers by parsing `pct list`.
func (b *PCTBackend) List() ([]ContainerInfo, error) {
	out, err := b.run("list")
	if err != nil {
		return nil, err
	}
	return parseList(string(out)), nil
}

// parseList parses the columnar output of `pct list`:
//
//	VMID       Status     Lock         Name
//	100        running                 web
func parseList(out string) []ContainerInfo {
	var infos []ContainerInfo
	lines := strings.Split(strings.TrimSpace(out), "\n")
	for i, line := range lines {
		if i == 0 || strings.TrimSpace(line) == "" {
			continue // header / blank
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		vmid, err := strconv.Atoi(fields[0])
		if err != nil {
			continue
		}
		info := ContainerInfo{VMID: vmid, Status: parseStatusWord(fields[1])}
		// Name is the last field when present (Lock column may be empty).
		if len(fields) >= 3 {
			info.Name = fields[len(fields)-1]
		}
		infos = append(infos, info)
	}
	return infos
}

// VMIDs returns the numeric IDs of all containers currently on the node.
func (b *PCTBackend) VMIDs() ([]int, error) {
	infos, err := b.List()
	if err != nil {
		return nil, err
	}
	ids := make([]int, 0, len(infos))
	for _, info := range infos {
		ids = append(ids, info.VMID)
	}
	return ids, nil
}

func parseStatusWord(w string) Status {
	switch strings.ToLower(w) {
	case "running":
		return StatusRunning
	case "stopped":
		return StatusStopped
	case "suspended", "paused", "frozen":
		return StatusPaused
	default:
		return StatusUnknown
	}
}
