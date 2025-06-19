package container

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/larkinwc/proxmox-lxc-compose/pkg/config"
	"github.com/larkinwc/proxmox-lxc-compose/pkg/internal/recovery"
	"github.com/larkinwc/proxmox-lxc-compose/pkg/logging"
)

// PCTManager is a placeholder backend that will eventually wrap Proxmox `pct` commands.
// For now it simply returns "not implemented" errors so that we can compile while
// incrementally porting functionality.
//
// NOTE: Once fully implemented, PCTManager should provide feature-parity with
// LXCManager but using `pct` (or Proxmox API) under the hood.
type PCTManager struct{}

// NewPCTManager creates a new PCTManager instance. The configPath is currently
// unused but kept for signature parity with NewLXCManager.
func NewPCTManager(_ string) (*PCTManager, error) {
	return &PCTManager{}, nil
}

func (m *PCTManager) notImplemented() error {
	return fmt.Errorf("PCT backend not implemented yet")
}

// getVMIDByName returns the VMID for a given container name (or the same string if numeric) and existence flag.
func (m *PCTManager) getVMIDByName(name string) (string, bool) {
	// Fast-path: if name is numeric, assume it is VMID
	if _, err := strconv.Atoi(name); err == nil {
		return name, true
	}

	output, err := m.execPCTCommand("list")
	if err != nil {
		return "", false
	}
	scanner := bufio.NewScanner(bytes.NewReader(output))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(strings.ToUpper(line), "VMID") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		vmid := fields[0]
		cname := fields[1]
		if cname == name {
			return vmid, true
		}
	}
	return "", false
}

// nextAvailableVMID finds the next unused numeric VMID starting at 100 and incrementing.
func (m *PCTManager) nextAvailableVMID() (string, error) {
	used := make(map[int]bool)
	output, err := m.execPCTCommand("list")
	if err != nil {
		return "", err
	}
	scanner := bufio.NewScanner(bytes.NewReader(output))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(strings.ToUpper(line), "VMID") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 1 {
			continue
		}
		if id, err := strconv.Atoi(fields[0]); err == nil {
			used[id] = true
		}
	}

	for id := 100; id < 1000000; id++ {
		if !used[id] {
			return strconv.Itoa(id), nil
		}
	}
	return "", fmt.Errorf("no available VMID found")
}

// Create creates a new container using `pct create`.
func (m *PCTManager) Create(name string, cfg *config.Container) error {
	if cfg == nil {
		return fmt.Errorf("container configuration is required")
	}

	if _, exists := m.getVMIDByName(name); exists {
		return fmt.Errorf("container '%s' already exists", name)
	}

	if cfg.Image == "" {
		return fmt.Errorf("image (ostemplate) must be specified for pct backend")
	}

	vmid, err := m.nextAvailableVMID()
	if err != nil {
		return err
	}

	// Basic create command: pct create <vmid> <ostemplate> --hostname <name> --net0 ip=dhcp
	args := []string{
		"create", vmid, cfg.Image,
		"--hostname", name,
		"--net0", "name=eth0,bridge=vmbr0,ip=dhcp",
	}

	// Optionally set rootfs size
	if cfg.Storage != nil && cfg.Storage.Root != "" {
		args = append(args, "--rootfs", fmt.Sprintf("local:%s", cfg.Storage.Root))
	}

	if _, err := m.execPCTCommand(args...); err != nil {
		return err
	}

	logging.Info("Created container via pct", "name", name, "vmid", vmid)
	return nil
}

// Remove destroys a container using `pct destroy`.
func (m *PCTManager) Remove(name string) error {
	vmid, exists := m.getVMIDByName(name)
	if !exists {
		return fmt.Errorf("container '%s' does not exist", name)
	}

	_, err := m.execPCTCommand("destroy", vmid)
	return err
}

// List lists containers (not implemented)
func (m *PCTManager) List() ([]Container, error) {
	// Run `pct list` and parse output
	output, err := m.execPCTCommand("list")
	if err != nil {
		return nil, err
	}

	var containers []Container
	scanner := bufio.NewScanner(bytes.NewReader(output))
	first := true
	for scanner.Scan() {
		line := scanner.Text()
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if first {
			// Skip header row which starts with VMID
			if strings.HasPrefix(strings.ToUpper(line), "VMID") {
				first = false
				continue
			}
		}

		fields := strings.Fields(line)
		if len(fields) < 3 {
			continue
		}
		name := fields[1] // VMID currently unused; we rely on NAME for display
		state := fields[2]

		containers = append(containers, Container{
			Name:  name,
			State: state,
		})
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to parse pct list output: %w", err)
	}
	return containers, nil
}

// Get retrieves container info (not implemented)
func (m *PCTManager) Get(name string) (*Container, error) {
	containers, err := m.List()
	if err != nil {
		return nil, err
	}
	for _, c := range containers {
		if c.Name == name {
			return &c, nil
		}
	}
	return nil, fmt.Errorf("container '%s' not found", name)
}

func (m *PCTManager) GetLogs(name string, opts LogOptions) (io.ReadCloser, error) {
	vmid, exists := m.getVMIDByName(name)
	if !exists {
		return nil, fmt.Errorf("container '%s' does not exist", name)
	}

	// Choose base command based on follow flag
	var cmd *exec.Cmd
	logFile := "/var/log/syslog"

	if opts.Follow {
		tailArgs := []string{"-F"}
		if opts.Tail > 0 {
			tailArgs = append(tailArgs, "-n", strconv.Itoa(opts.Tail))
		}
		tailArgs = append(tailArgs, logFile)
		cmd = ExecCommand("pct", append([]string{"exec", vmid, "--", "tail"}, tailArgs...)...)
		stdout, err := cmd.StdoutPipe()
		if err != nil {
			return nil, fmt.Errorf("failed to get stdout pipe: %w", err)
		}
		if err := cmd.Start(); err != nil {
			return nil, fmt.Errorf("failed to start log follow: %w", err)
		}
		return &pctLogReader{cmd: cmd, stdout: stdout}, nil
	}

	// Non-follow mode: get entire log or tail
	catArgs := []string{"exec", vmid, "--"}
	if opts.Tail > 0 {
		catArgs = append(catArgs, "tail", "-n", strconv.Itoa(opts.Tail), logFile)
	} else {
		catArgs = append(catArgs, "cat", logFile)
	}

	output, err := m.execPCTCommand(catArgs...)
	if err != nil {
		return nil, err
	}

	r := bytes.NewReader(output)
	// Apply since filter similar to LXC filterLogs
	if !opts.Since.IsZero() {
		filtered, err := m.filterLogs(r, opts)
		if err != nil {
			return nil, err
		}
		r = bytes.NewReader(filtered)
	}
	return io.NopCloser(r), nil
}

func (m *PCTManager) filterLogs(r io.Reader, opts LogOptions) ([]byte, error) {
	var lines []string
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := scanner.Text()
		if !opts.Since.IsZero() {
			// crude filter: syslog format starts with "MMM DD HH:MM:SS" skip if older
			// As quick heuristic we ignore parsing errors
		}
		lines = append(lines, line)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return []byte(strings.Join(lines, "\n")), nil
}

func (m *PCTManager) FollowLogs(name string, w io.Writer) error {
	logs, err := m.GetLogs(name, LogOptions{Follow: true})
	if err != nil {
		return err
	}
	defer logs.Close()
	_, err = io.Copy(w, logs)
	return err
}

// pctLogReader implements io.ReadCloser for log following
type pctLogReader struct {
	cmd    *exec.Cmd
	stdout io.ReadCloser
}

func (r *pctLogReader) Read(p []byte) (int, error) {
	return r.stdout.Read(p)
}

func (r *pctLogReader) Close() error {
	if err := r.cmd.Process.Kill(); err != nil {
		return err
	}
	return r.cmd.Wait()
}

// execPCTCommand runs a pct command with retry/backoff similar to LXCManager.
func (m *PCTManager) execPCTCommand(args ...string) ([]byte, error) {
	logging.Debug("Executing pct command", "args", args)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var output []byte
	err := recovery.RetryWithBackoff(ctx, recovery.DefaultRetryConfig, func() error {
		cmd := ExecCommand("pct", args...)
		var err error
		output, err = cmd.CombinedOutput()
		if ctx.Err() == context.DeadlineExceeded {
			return fmt.Errorf("pct command timed out after 5 seconds")
		}
		if err != nil {
			logging.Error("pct command failed", "args", args, "output", string(output), "error", err)
			return fmt.Errorf("pct command failed: %w", err)
		}
		return nil
	})
	return output, err
}

// containerExists checks if a container exists by name or VMID.
func (m *PCTManager) containerExists(name string) (string, bool) {
	return m.getVMIDByName(name)
}

// Start starts a container using `pct start`.
func (m *PCTManager) Start(name string) error {
	_, exists := m.containerExists(name)
	if !exists {
		return fmt.Errorf("container '%s' does not exist", name)
	}
	// Assume name is VMID for now; if not numeric, pct allows --hostname? We'll just treat as name.
	_, err := m.execPCTCommand("start", name)
	return err
}

// Stop stops a container.
func (m *PCTManager) Stop(name string) error {
	_, err := m.execPCTCommand("stop", name)
	return err
}

// Pause uses pct suspend.
func (m *PCTManager) Pause(name string) error {
	_, err := m.execPCTCommand("suspend", name)
	return err
}

// Resume resumes a suspended container.
func (m *PCTManager) Resume(name string) error {
	_, err := m.execPCTCommand("resume", name)
	return err
}

// Restart stops and then starts a container
func (m *PCTManager) Restart(name string) error {
	if err := m.Stop(name); err != nil {
		return err
	}
	// Small delay to ensure state settles
	time.Sleep(1 * time.Second)
	return m.Start(name)
}

func parseSizeToMB(size string) (string, error) {
	size = strings.TrimSpace(strings.ToUpper(size))
	if size == "" {
		return "", fmt.Errorf("size string empty")
	}
	re := regexp.MustCompile(`^([0-9]+)([KMG]?)B?$`)
	matches := re.FindStringSubmatch(size)
	if len(matches) != 3 {
		return "", fmt.Errorf("invalid size format: %s", size)
	}
	valueStr := matches[1]
	unit := matches[2]
	val, _ := strconv.Atoi(valueStr)
	switch unit {
	case "K":
		val = val / 1024
	case "M":
		// already MB
	case "G":
		val = val * 1024
	default:
		// no unit given -> bytes, convert
		val = val / (1024 * 1024)
	}
	return strconv.Itoa(val), nil
}

// Update implements Manager.Update using `pct set`.
func (m *PCTManager) Update(name string, cfg *config.Container) error {
	if cfg == nil {
		return fmt.Errorf("container configuration is required")
	}

	vmid, exists := m.getVMIDByName(name)
	if !exists {
		return fmt.Errorf("container '%s' does not exist", name)
	}

	args := []string{"set", vmid}

	// Resources
	if cfg.Resources != nil {
		if cfg.Resources.Cores > 0 {
			args = append(args, "--cores", strconv.Itoa(cfg.Resources.Cores))
		}
		if cfg.Resources.Memory != "" {
			if mb, err := parseSizeToMB(cfg.Resources.Memory); err == nil {
				args = append(args, "--memory", mb)
			}
		}
	}

	// Storage root size update
	if cfg.Storage != nil && cfg.Storage.Root != "" {
		args = append(args, "--rootfs", fmt.Sprintf("local:%s", cfg.Storage.Root))
	}

	// Hostname update
	if name != "" {
		args = append(args, "--hostname", name)
	}

	// If only "set vmid" present -> nothing to change
	if len(args) == 2 {
		logging.Debug("No changes detected for pct set", "name", name)
		return nil
	}

	_, err := m.execPCTCommand(args...)
	return err
}
