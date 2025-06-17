package container

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/larkinwc/proxmox-lxc-compose/pkg/config"
	"github.com/larkinwc/proxmox-lxc-compose/pkg/internal/recovery"
	"github.com/larkinwc/proxmox-lxc-compose/pkg/logging"
	"github.com/larkinwc/proxmox-lxc-compose/pkg/oci"
	"github.com/larkinwc/proxmox-lxc-compose/pkg/proxmox"
)

// Manager defines the interface for managing LXC containers
type Manager interface {
	// Create creates a new container from the given configuration
	Create(name string, cfg *config.Container) error
	// Start starts a container
	Start(name string) error
	// Stop stops a container
	Stop(name string) error
	// Remove removes a container
	Remove(name string) error
	// List returns a list of all containers
	List() ([]Container, error)
	// Get returns information about a specific container
	Get(name string) (*Container, error)
	// Pause freezes a running container
	Pause(name string) error
	// Resume unfreezes a paused container
	Resume(name string) error
	// Restart stops and then starts a container
	Restart(name string) error
	// Update updates a container's configuration
	Update(name string, cfg *config.Container) error
	// GetLogs returns container logs with given options
	GetLogs(name string, opts LogOptions) (io.ReadCloser, error)
	// FollowLogs streams logs to writer
	FollowLogs(name string, w io.Writer) error
}

// LXCManager implements the Manager interface for LXC containers
type LXCManager struct {
	configPath string
	state      *StateManager
	client     proxmox.Client
}

// NewLXCManager creates a new LXC container manager
func NewLXCManager(configPath string) (*LXCManager, error) {
	logging.Debug("Initializing LXC manager", "configPath", configPath)

	stateManager, err := NewStateManager(filepath.Join(configPath, "state"))
	if err != nil {
		return nil, fmt.Errorf("failed to create state manager: %w", err)
	}

	// Initialize the Proxmox client
	client, err := proxmox.NewClient()
	if err != nil {
		return nil, fmt.Errorf("failed to create proxmox client: %w", err)
	}

	return &LXCManager{
		configPath: configPath,
		state:      stateManager,
		client:     client,
	}, nil
}

func (m *LXCManager) execLXCCommand(name string, args ...string) error {
	logging.Debug("Executing LXC command",
		"command", name,
		"args", args,
		"container", args[1], // args[1] is usually the container name
	)

	// Create a context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Use retry with backoff for commands that might fail temporarily
	return recovery.RetryWithBackoff(ctx, recovery.DefaultRetryConfig, func() error {
		cmd := ExecCommand(name, args...)
		output, err := cmd.CombinedOutput()

		// Check if the command timed out
		if ctx.Err() == context.DeadlineExceeded {
			return fmt.Errorf("command timed out after 5 seconds")
		}

		if err != nil {
			// Use debug level for expected failures like container non-existence
			if name == "lxc-info" && strings.Contains(string(output), "doesn't exist") {
				logging.Debug("Container does not exist (expected)",
					"command", name,
					"args", args,
					"output", string(output),
				)
			} else {
				logging.Error("Command failed",
					"command", name,
					"args", args,
					"output", string(output),
					"error", err,
				)
			}
			return fmt.Errorf("command failed: %w", err)
		}
		return nil
	})
}

// ContainerExists checks if a container exists
func (m *LXCManager) ContainerExists(name string) bool {
	logging.Debug("Checking if container exists", "name", name)

	// Only check state first - directory existence is not enough
	if _, err := m.state.GetContainerState(name); err == nil {
		logging.Debug("Container found in state", "name", name)
		return true
	}

	// If not in state, check LXC
	if err := m.execLXCCommand("lxc-info", "-n", name); err == nil {
		logging.Debug("Container found in LXC", "name", name)
		return true
	}

	logging.Debug("Container does not exist", "name", name)
	return false
}

// Create implements Manager.Create
func (m *LXCManager) Create(name string, cfg *config.Container) error {
	if cfg == nil {
		return fmt.Errorf("container configuration is required")
	}

	if m.ContainerExists(name) {
		return fmt.Errorf("container %s already exists", name)
	}

	// Create container directory structure
	containerDir := filepath.Join(m.configPath, name)
	dirs := []string{
		containerDir,
		filepath.Join(containerDir, "rootfs"),
		filepath.Join(containerDir, "logs"),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create container directory %s: %w", dir, err)
		}
	}

	// Pull and convert the OCI image
	if cfg.Image != "" {
		// Convert the image to LXC format
		templatePath := filepath.Join(m.configPath, "templates", fmt.Sprintf("%s.tar.gz", cfg.Image))
		if err := oci.ConvertOCIToLXC(cfg.Image, templatePath); err != nil {
			return fmt.Errorf("failed to convert image: %w", err)
		}

		// Extract the template to the rootfs
		rootfsPath := filepath.Join(containerDir, "rootfs")

		// Use tar with proper flags for container rootfs extraction
		cmd := ExecCommand("tar", "-xzf", templatePath, "-C", rootfsPath, "--numeric-owner", "--preserve-permissions")
		if output, err := cmd.CombinedOutput(); err != nil {
			logging.Error("Failed to extract rootfs", "templatePath", templatePath, "rootfsPath", rootfsPath, "output", string(output))
			return fmt.Errorf("failed to extract rootfs: %w (output: %s)", err, string(output))
		}

		logging.Debug("Rootfs extraction completed", "templatePath", templatePath, "rootfsPath", rootfsPath)

		// Setup container for proper initialization
		if err := m.setupContainerInit(rootfsPath); err != nil {
			logging.Error("Failed to setup container init", "container", name, "error", err)
		}
	}

	// Generate the main LXC configuration file
	if err := m.generateLXCConfig(name, cfg); err != nil {
		return fmt.Errorf("failed to generate LXC config: %w", err)
	}

	// Apply container configuration
	if cfg.Resources != nil {
		if err := m.applyCPUConfig(name, &config.CPUConfig{Cores: &cfg.Resources.Cores}); err != nil {
			return fmt.Errorf("failed to apply CPU configuration: %w", err)
		}
		if err := m.applyMemoryConfig(name, &config.MemoryConfig{Limit: cfg.Resources.Memory}); err != nil {
			return fmt.Errorf("failed to apply memory configuration: %w", err)
		}
	}

	// Configure network if specified
	if cfg.Network != nil {
		if err := m.configureNetwork(name, cfg.Network); err != nil {
			return fmt.Errorf("failed to configure network: %w", err)
		}
	}

	// Save initial state
	if err := m.state.SaveContainerState(name, cfg, "STOPPED"); err != nil {
		return fmt.Errorf("failed to save container state: %w", err)
	}

	logging.Debug("Container created and state saved", "name", name)

	return nil
}

// Start implements Manager.Start
func (m *LXCManager) Start(name string) error {
	container, err := m.Get(name)
	if err != nil {
		return fmt.Errorf("failed to get container: %w", err)
	}

	if container.State == "RUNNING" {
		return fmt.Errorf("container '%s' is already running", name)
	}

	if container.State != "STOPPED" {
		return fmt.Errorf("container '%s' is not in a valid state for starting (current state: %s)", name, container.State)
	}

	// Start the container with debugging enabled
	logFile := filepath.Join(m.configPath, name, "start.log")
	if err := m.execLXCCommand("lxc-start", "-n", name, "-F", "-o", logFile, "-l", "DEBUG"); err != nil {
		// Try to read the log file for more details
		if logContent, readErr := os.ReadFile(logFile); readErr == nil {
			logging.Error("Container start failed", "container", name, "logContent", string(logContent))
		}
		return fmt.Errorf("failed to start container: %w", err)
	}

	// Update state - container.Config is already *config.Container
	if err := m.state.SaveContainerState(name, container.Config, "RUNNING"); err != nil {
		return fmt.Errorf("failed to update container state: %w", err)
	}

	return nil
}

// Stop implements Manager.Stop
func (m *LXCManager) Stop(name string) error {
	container, err := m.Get(name)
	if err != nil {
		return fmt.Errorf("failed to get container: %w", err)
	}

	if container.State == "STOPPED" {
		return fmt.Errorf("container '%s' is already stopped", name)
	}

	if container.State != "RUNNING" && container.State != "FROZEN" {
		return fmt.Errorf("container '%s' is not in a valid state for stopping (current state: %s)", name, container.State)
	}

	// Stop the container using execLXCCommand
	if err := m.execLXCCommand("lxc-stop", "-n", name); err != nil {
		return fmt.Errorf("failed to stop container: %w", err)
	}

	// Update state - container.Config is already *config.Container
	if err := m.state.SaveContainerState(name, container.Config, "STOPPED"); err != nil {
		return fmt.Errorf("failed to update container state: %w", err)
	}

	return nil
}

// Remove implements Manager.Remove
func (m *LXCManager) Remove(name string) error {
	container, err := m.Get(name)
	if err != nil {
		return fmt.Errorf("failed to get container: %w", err)
	}

	if container.State != "STOPPED" {
		return fmt.Errorf("container '%s' must be stopped before removal", name)
	}

	// Try to destroy container in LXC - use force flag to handle corrupted configs
	if err := m.execLXCCommand("lxc-destroy", "-n", name, "-f"); err != nil {
		// If lxc-destroy fails, log the error but continue with manual cleanup
		logging.Error("LXC destroy failed, attempting manual cleanup", "container", name, "error", err)
	}

	// Remove container directory (this cleans up corrupted containers)
	containerPath := filepath.Join(m.configPath, name)
	if err := os.RemoveAll(containerPath); err != nil {
		return fmt.Errorf("failed to remove container directory: %w", err)
	}

	// Remove state
	if err := m.state.RemoveContainerState(name); err != nil {
		return fmt.Errorf("failed to remove container state: %w", err)
	}

	return nil
}

// List implements Manager.List
func (m *LXCManager) List() ([]Container, error) {
	entries, err := os.ReadDir(m.configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config directory: %w", err)
	}

	var containers []Container
	for _, entry := range entries {
		if !entry.IsDir() || entry.Name() == "state" || entry.Name() == "templates" {
			continue
		}

		container, err := m.Get(entry.Name())
		if err != nil {
			continue
		}

		containers = append(containers, *container)
	}

	return containers, nil
}

// Get implements Manager.Get
func (m *LXCManager) Get(name string) (*Container, error) {
	if !m.ContainerExists(name) {
		return nil, fmt.Errorf("container %s does not exist", name)
	}

	// First check if we have state info
	state, err := m.state.GetContainerState(name)
	if err != nil {
		// Create default state if none exists
		state = &State{
			Name:   name,
			Status: "STOPPED",
		}
	}

	// Try up to 3 times to get a stable state
	for i := 0; i < 3; i++ {
		cmd := ExecCommand("lxc-info", "-n", name)
		output, err := cmd.CombinedOutput()
		if err == nil {
			currentState := ""
			// Parse lxc-info output to get state
			for _, line := range strings.Split(string(output), "\n") {
				if strings.HasPrefix(line, "State:") {
					lxcState := strings.TrimSpace(strings.TrimPrefix(line, "State:"))
					switch strings.ToUpper(lxcState) {
					case "RUNNING":
						currentState = "RUNNING"
					case "STOPPED":
						currentState = "STOPPED"
					case "FROZEN":
						currentState = "FROZEN"
					}
					break
				}
			}

			// If we got a valid state that matches our saved state or we've tried 3 times,
			// use this state
			if currentState != "" && (currentState == state.Status || i == 2) {
				state.Status = currentState
				break
			}
		}

		// Wait a short time before retrying
		if i < 2 {
			time.Sleep(100 * time.Millisecond)
		}
	}

	return &Container{
		Name:   name,
		State:  state.Status,
		Config: state.Config,
	}, nil
}

// Pause implements Manager.Pause
func (m *LXCManager) Pause(name string) error {
	container, err := m.Get(name)
	if err != nil {
		return fmt.Errorf("failed to get container: %w", err)
	}

	if container.State == "FROZEN" {
		return fmt.Errorf("container '%s' is already frozen", name)
	}

	if container.State != "RUNNING" {
		return fmt.Errorf("container '%s' is not in a valid state for pausing (current state: %s)", name, container.State)
	}

	// Freeze the container
	if err := m.execLXCCommand("lxc-freeze", "-n", name); err != nil {
		return fmt.Errorf("failed to pause container: %w", err)
	}

	// Update state - container.Config is already *config.Container
	if err := m.state.SaveContainerState(name, container.Config, "FROZEN"); err != nil {
		return fmt.Errorf("failed to update container state: %w", err)
	}

	return nil
}

// Resume implements Manager.Resume
func (m *LXCManager) Resume(name string) error {
	container, err := m.Get(name)
	if err != nil {
		return fmt.Errorf("failed to get container: %w", err)
	}

	if container.State == "RUNNING" {
		return fmt.Errorf("container '%s' is already running", name)
	}

	if container.State != "FROZEN" {
		return fmt.Errorf("container '%s' is not in a valid state for resuming (current state: %s)", name, container.State)
	}

	// Unfreeze the container
	if err := m.execLXCCommand("lxc-unfreeze", "-n", name); err != nil {
		return fmt.Errorf("failed to resume container: %w", err)
	}

	// Update state - container.Config is already *config.Container
	if err := m.state.SaveContainerState(name, container.Config, "RUNNING"); err != nil {
		return fmt.Errorf("failed to update container state: %w", err)
	}

	return nil
}

// Restart implements Manager.Restart
func (m *LXCManager) Restart(name string) error {
	container, err := m.Get(name)
	if err != nil {
		return fmt.Errorf("failed to get container: %w", err)
	}

	// If container is running or frozen, stop it first
	if container.State == "RUNNING" || container.State == "FROZEN" {
		if err := m.execLXCCommand("lxc-stop", "-n", name); err != nil {
			return fmt.Errorf("failed to stop container: %w", err)
		}

		// Update state to STOPPED - container.Config is already *config.Container
		if err := m.state.SaveContainerState(name, container.Config, "STOPPED"); err != nil {
			return fmt.Errorf("failed to update container state: %w", err)
		}

		// Verify state update
		container, err = m.Get(name)
		if err != nil {
			return fmt.Errorf("failed to get container state: %w", err)
		}
	}

	// Start the container
	if err := m.execLXCCommand("lxc-start", "-n", name); err != nil {
		return fmt.Errorf("failed to start container: %w", err)
	}

	// Update state to RUNNING - container.Config is already *config.Container
	if err := m.state.SaveContainerState(name, container.Config, "RUNNING"); err != nil {
		return fmt.Errorf("failed to update container state: %w", err)
	}

	return nil
}

// Update implements Manager.Update
func (m *LXCManager) Update(name string, cfg *config.Container) error {
	if cfg == nil {
		return fmt.Errorf("container configuration is required")
	}

	// Check if container exists
	if !m.ContainerExists(name) {
		return fmt.Errorf("container %s does not exist", name)
	}

	container, err := m.Get(name)
	if err != nil {
		return fmt.Errorf("failed to get container: %w", err)
	}

	// Save initial state
	if err := m.state.SaveContainerState(name, cfg, container.State); err != nil {
		return fmt.Errorf("failed to save container state: %w", err)
	}

	// Apply new configuration
	if cfg.Resources != nil {
		if err := m.applyCPUConfig(name, &config.CPUConfig{Cores: &cfg.Resources.Cores}); err != nil {
			return fmt.Errorf("failed to apply CPU configuration: %w", err)
		}
		if err := m.applyMemoryConfig(name, &config.MemoryConfig{Limit: cfg.Resources.Memory}); err != nil {
			return fmt.Errorf("failed to apply memory configuration: %w", err)
		}
	}
	if err := m.configureNetwork(name, cfg.Network); err != nil {
		return fmt.Errorf("failed to apply network configuration: %w", err)
	}
	// Note: storage, security, env, and entrypoint are not yet implemented

	return nil
}

// generateLXCConfig creates the main LXC configuration file for a container
func (m *LXCManager) generateLXCConfig(name string, cfg *config.Container) error {
	containerDir := filepath.Join(m.configPath, name)
	configPath := filepath.Join(containerDir, "config")

	var lines []string

	// Basic container configuration
	lines = append(lines, fmt.Sprintf("lxc.uts.name = %s", name))
	lines = append(lines, fmt.Sprintf("lxc.rootfs.path = dir:%s/rootfs", containerDir))

	// Security settings
	if cfg.Security != nil {
		if cfg.Security.Privileged {
			lines = append(lines, "lxc.seccomp.profile =")
		} else {
			lines = append(lines, "lxc.apparmor.profile = generated")
			lines = append(lines, "lxc.seccomp.profile = /usr/share/lxc/config/common.seccomp")
		}
	} else {
		// Default to unprivileged
		lines = append(lines, "lxc.apparmor.profile = generated")
		lines = append(lines, "lxc.seccomp.profile = /usr/share/lxc/config/common.seccomp")
	}

	// Include common configuration
	lines = append(lines, "lxc.include = /usr/share/lxc/config/common.conf")

	// Include distribution-specific config if available
	lines = append(lines, "lxc.include = /usr/share/lxc/config/ubuntu.common.conf")

	// Add init system configuration - detect what's available
	initCmd := m.detectInitCommand(containerDir)
	if initCmd != "" {
		lines = append(lines, fmt.Sprintf("lxc.init.cmd = %s", initCmd))
		if initCmd == "/sbin/init" || strings.Contains(initCmd, "systemd") {
			lines = append(lines, "lxc.signal.halt = SIGRTMIN+3")
			lines = append(lines, "lxc.signal.reboot = SIGTERM")
		}
	}

	// Basic system configuration
	lines = append(lines, "lxc.arch = amd64")
	lines = append(lines, "lxc.tty.max = 4")
	lines = append(lines, "lxc.pty.max = 1024")

	// Add network configuration from external file
	networkConfigPath := filepath.Join(containerDir, "network.conf")
	if _, err := os.Stat(networkConfigPath); err == nil {
		lines = append(lines, fmt.Sprintf("lxc.include = %s", networkConfigPath))
	}

	// Network configuration with bridge validation
	if cfg.Network != nil {
		// Handle legacy network configuration directly
		if cfg.Network.Type != "" || cfg.Network.Bridge != "" || cfg.Network.IP != "" {
			// Determine bridge name
			bridgeName := cfg.Network.Bridge
			if bridgeName == "" {
				bridgeName = "lxcbr0" // default
			}

			// Check if bridge exists before trying to use it
			bridgeCmd := ExecCommand("ip", "link", "show", bridgeName)
			if err := bridgeCmd.Run(); err != nil {
				// Bridge doesn't exist, use host networking as fallback
				logging.Debug("Bridge not found, using host networking", "bridge", bridgeName)
				lines = append(lines, "lxc.net.0.type = none")
			} else {
				// Bridge exists, use it
				logging.Debug("Using bridge for networking", "bridge", bridgeName)
				lines = append(lines, "lxc.net.0.type = veth")
				lines = append(lines, fmt.Sprintf("lxc.net.0.link = %s", bridgeName))
				lines = append(lines, "lxc.net.0.flags = up")

				if cfg.Network.IP != "" {
					lines = append(lines, fmt.Sprintf("lxc.net.0.ipv4.address = %s", cfg.Network.IP))
					if cfg.Network.Gateway != "" {
						lines = append(lines, fmt.Sprintf("lxc.net.0.ipv4.gateway = %s", cfg.Network.Gateway))
					}
				}
			}
			// DNS configuration is handled via resolv.conf instead of LXC network config
			// LXC doesn't support lxc.net.0.ipv4.nameserver.X syntax in modern versions
		}
	} else {
		// No network configuration specified, use host networking
		logging.Debug("No network configuration, using host networking")
		lines = append(lines, "lxc.net.0.type = none")
	}

	// Resource limits - use cgroup v1 syntax for broader compatibility
	if cfg.Resources != nil {
		if cfg.Resources.Memory != "" {
			// Try cgroup v2 first, fallback handled by LXC
			lines = append(lines, fmt.Sprintf("lxc.cgroup.memory.limit_in_bytes = %s", cfg.Resources.Memory))
		}
		if cfg.Resources.Cores > 0 {
			// Set CPU limits using cgroup v1 syntax for compatibility
			lines = append(lines, fmt.Sprintf("lxc.cgroup.cpuset.cpus = 0-%d", cfg.Resources.Cores-1))
		}
	}

	// Environment variables
	for key, value := range cfg.Environment {
		lines = append(lines, fmt.Sprintf("lxc.environment = %s=%s", key, value))
	}

	// Autostart
	lines = append(lines, "lxc.start.auto = 0")

	// DNS configuration via resolv.conf bind mount if DNS servers are specified
	if cfg.Network != nil && len(cfg.Network.DNS) > 0 {
		if err := m.configureDNS(containerDir, cfg.Network.DNS); err != nil {
			logging.Error("Failed to configure DNS", "container", name, "error", err)
		} else {
			// Bind mount the custom resolv.conf
			resolvConfPath := filepath.Join(containerDir, "resolv.conf")
			lines = append(lines, fmt.Sprintf("lxc.mount.entry = %s etc/resolv.conf none bind,ro 0 0", resolvConfPath))
		}
	}

	// Write the configuration file
	content := strings.Join(lines, "\n") + "\n"
	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write LXC config file: %w", err)
	}

	logging.Debug("Generated LXC config file", "container", name, "path", configPath)
	return nil
}

// detectInitCommand determines the best init command for the container
func (m *LXCManager) detectInitCommand(containerDir string) string {
	rootfsPath := filepath.Join(containerDir, "rootfs")

	// Check for available init commands in order of preference
	initCandidates := []string{
		"/sbin/init",
		"/usr/sbin/init",
		"/bin/systemd",
		"/usr/bin/systemd",
		"/bin/bash",
		"/bin/sh",
	}

	for _, candidate := range initCandidates {
		// Remove leading slash to avoid double slash in path
		candidateRelPath := strings.TrimPrefix(candidate, "/")
		candidatePath := filepath.Join(rootfsPath, candidateRelPath)
		if info, err := os.Stat(candidatePath); err == nil && !info.IsDir() {
			// Check if it's executable
			if info.Mode()&0111 != 0 {
				logging.Debug("Found init command", "container", filepath.Base(containerDir), "init", candidate)
				return candidate
			}
		}
	}

	// If no init found, try to install systemd for Ubuntu containers
	if m.installSystemdIfNeeded(rootfsPath) {
		// Check again for /sbin/init after installation
		if _, err := os.Stat(filepath.Join(rootfsPath, "sbin/init")); err == nil {
			logging.Debug("Installed systemd, using /sbin/init", "container", filepath.Base(containerDir))
			return "/sbin/init"
		}
	}

	logging.Warn("No suitable init command found", "container", filepath.Base(containerDir))
	return "" // Let LXC use its default
}

// installSystemdIfNeeded attempts to install systemd in Ubuntu containers
func (m *LXCManager) installSystemdIfNeeded(rootfsPath string) bool {
	// Check if this is an Ubuntu system
	osReleasePath := filepath.Join(rootfsPath, "etc", "os-release")
	if data, err := os.ReadFile(osReleasePath); err == nil {
		content := string(data)
		if strings.Contains(content, "Ubuntu") {
			logging.Debug("Detected Ubuntu container, attempting to install systemd")

			// Use chroot to install systemd
			cmd := ExecCommand("chroot", rootfsPath, "sh", "-c",
				"export DEBIAN_FRONTEND=noninteractive && "+
					"apt-get update -qq >/dev/null 2>&1 && "+
					"apt-get install -y systemd systemd-sysv >/dev/null 2>&1")

			if err := cmd.Run(); err != nil {
				logging.Debug("Failed to install systemd via chroot", "error", err)
				return false
			}

			logging.Debug("Successfully installed systemd")
			return true
		}
	}

	return false
}

// configureDNS creates a custom resolv.conf for the container
func (m *LXCManager) configureDNS(containerDir string, dnsServers []string) error {
	resolvConfPath := filepath.Join(containerDir, "resolv.conf")

	var lines []string
	for _, dns := range dnsServers {
		lines = append(lines, fmt.Sprintf("nameserver %s", dns))
	}

	// Add default search domain
	lines = append(lines, "search localdomain")

	content := strings.Join(lines, "\n") + "\n"
	if err := os.WriteFile(resolvConfPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write resolv.conf: %w", err)
	}

	return nil
}

// setupContainerInit sets up the container for proper initialization
func (m *LXCManager) setupContainerInit(rootfsPath string) error {
	// Create necessary directories for container operation
	dirs := []string{
		"dev", "proc", "sys", "tmp", "var/run", "var/lock", "var/log", "run", "run/lock",
	}

	for _, dir := range dirs {
		dirPath := filepath.Join(rootfsPath, dir)
		if err := os.MkdirAll(dirPath, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	// Check if this looks like an Ubuntu rootfs and handle accordingly
	if _, err := os.Stat(filepath.Join(rootfsPath, "usr", "bin", "systemctl")); err == nil {
		// This appears to be a systemd-based system (Ubuntu 20.04+)
		logging.Debug("Detected systemd-based rootfs, setting up for systemd init")

		// Create systemd directories
		systemdDirs := []string{
			"run/systemd", "var/lib/systemd", "etc/systemd/system",
		}
		for _, dir := range systemdDirs {
			dirPath := filepath.Join(rootfsPath, dir)
			if err := os.MkdirAll(dirPath, 0755); err != nil {
				logging.Debug("Could not create systemd directory", "dir", dir, "error", err)
			}
		}
	} else {
		// Fallback to traditional init system
		logging.Debug("Setting up traditional init system")

		// Create minimal /etc/inittab for simple init
		inittabPath := filepath.Join(rootfsPath, "etc", "inittab")
		inittabContent := `# /etc/inittab: init(8) configuration.
id:3:initdefault:
si::sysinit:/etc/init.d/rcS
l0:0:wait:/etc/init.d/rc 0
l1:1:wait:/etc/init.d/rc 1
l2:2:wait:/etc/init.d/rc 2
l3:3:wait:/etc/init.d/rc 3
l4:4:wait:/etc/init.d/rc 4
l5:5:wait:/etc/init.d/rc 5
l6:6:wait:/etc/init.d/rc 6
1:2345:respawn:/sbin/getty 38400 console
c1:12345:respawn:/sbin/getty 38400 tty1 linux
`
		if err := os.WriteFile(inittabPath, []byte(inittabContent), 0644); err != nil {
			logging.Debug("Could not create inittab", "error", err)
		}
	}

	// Ensure proper permissions on key directories
	keyDirs := map[string]os.FileMode{
		"tmp":     0777,
		"var/run": 0755,
		"var/log": 0755,
		"run":     0755,
	}

	for dir, mode := range keyDirs {
		dirPath := filepath.Join(rootfsPath, dir)
		if err := os.Chmod(dirPath, mode); err != nil {
			logging.Debug("Could not set permissions", "dir", dir, "error", err)
		}
	}

	// Verify rootfs structure
	logging.Debug("Rootfs setup completed", "rootfsPath", rootfsPath)
	if entries, err := os.ReadDir(rootfsPath); err == nil {
		var dirNames []string
		for _, entry := range entries {
			if entry.IsDir() {
				dirNames = append(dirNames, entry.Name())
			}
		}
		logging.Debug("Rootfs directories", "directories", strings.Join(dirNames, ", "))
	}

	return nil
}
