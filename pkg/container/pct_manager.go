package container

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
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

// Create creates a new container (not implemented)
func (m *PCTManager) Create(name string, cfg *config.Container) error {
	return m.notImplemented()
}

// Remove removes a container (not implemented)
func (m *PCTManager) Remove(name string) error {
	return m.notImplemented()
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
	return nil, m.notImplemented()
}

func (m *PCTManager) FollowLogs(name string, w io.Writer) error {
	return m.notImplemented()
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

// ContainerExists checks if a container exists by name or VMID.
func (m *PCTManager) containerExists(name string) (string, bool) {
	containers, err := m.List()
	if err != nil {
		return "", false
	}
	for _, c := range containers {
		if c.Name == name {
			return name, true
		}
		// Also allow numeric ID provided as name
		if c.State != "" && c.Config == nil { // just ignore config
		}
	}
	return "", false
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

// Update updates container configuration. For now, not implemented.
func (m *PCTManager) Update(name string, cfg *config.Container) error {
	return m.notImplemented()
}
