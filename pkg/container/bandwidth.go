// Package container implements LXC container management
package container

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"proxmox-lxc-compose/pkg/common"
	"proxmox-lxc-compose/pkg/logging"
)

// NetworkBandwidthLimit represents bandwidth limits for a network interface
type NetworkBandwidthLimit struct {
	Interface string
	InRate    int64 // Ingress rate in bytes per second
	OutRate   int64 // Egress rate in bytes per second
}

// SetNetworkBandwidthLimit sets bandwidth limits for a container's network interface
func (m *LXCManager) SetNetworkBandwidthLimit(name string, limit NetworkBandwidthLimit) error {
	// Ensure container exists
	containerPath := filepath.Join(m.configPath, name)
	if _, err := os.Stat(containerPath); os.IsNotExist(err) {
		return fmt.Errorf("container %s does not exist", name)
	}

	// Generate traffic control commands
	lines := []string{
		fmt.Sprintf("lxc.hook.pre-start = tc qdisc add dev %s root handle 1: htb default 10", limit.Interface),
		fmt.Sprintf("lxc.hook.pre-start = tc class add dev %s parent 1: classid 1:10 htb rate %dbps", limit.Interface, limit.InRate),
		fmt.Sprintf("lxc.hook.pre-start = tc class add dev %s parent 1: classid 1:20 htb rate %dbps", limit.Interface, limit.OutRate),
		// Cleanup rules on container stop
		fmt.Sprintf("lxc.hook.post-stop = tc qdisc del dev %s root", limit.Interface),
	}

	// Update container config
	configPath := filepath.Join(containerPath, "config")
	f, err := os.OpenFile(configPath, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open config file: %w", err)
	}
	defer f.Close()

	for _, line := range lines {
		if _, err := f.WriteString(line + "\n"); err != nil {
			return fmt.Errorf("failed to write config: %w", err)
		}
	}

	logging.Debug("Set network bandwidth limits",
		"container", name,
		"interface", limit.Interface,
		"in_rate", limit.InRate,
		"out_rate", limit.OutRate,
	)

	return nil
}

// GetNetworkBandwidthLimits gets current bandwidth limits for a container's network interface
func (m *LXCManager) GetNetworkBandwidthLimits(name, iface string) (*common.BandwidthLimit, error) {
	// Read tc class info using lxc-attach
	args := []string{"-n", name, "--", "tc", "class", "show", "dev", iface}
	cmdStr := fmt.Sprintf("lxc-attach %s", strings.Join(args, " "))
	logging.Debug("Executing tc command",
		"command", cmdStr,
		"name", name,
		"args", strings.Join(args, " "),
		"full_command", cmdStr)

	cmd := ExecCommand("lxc-attach", args...)
	var output bytes.Buffer
	cmd.Stdout = &output
	cmd.Stderr = &output // Also capture stderr for debugging
	if err := cmd.Run(); err != nil {
		logging.Debug("tc command failed",
			"error", err,
			"output", output.String(),
			"command", cmdStr)
		return nil, fmt.Errorf("failed to get bandwidth limits: %w", err)
	}

	// Log the raw output for debugging
	logging.Debug("tc command output", "output", output.String(), "output_len", len(output.String()))

	// Parse tc output to get rate limits
	limit := &common.BandwidthLimit{}

	// Parse tc output
	for _, line := range strings.Split(output.String(), "\n") {
		logging.Debug("Processing line", "line", line)
		if strings.Contains(line, "rate") {
			fields := strings.Fields(line)
			logging.Debug("Found rate in line", "fields", fields)
			for i, field := range fields {
				if field == "rate" && i+1 < len(fields) {
					if strings.Contains(line, "1:10") {
						limit.IngressRate = strings.ToLower(fields[i+1])
						logging.Debug("Found ingress rate", "rate", limit.IngressRate)
						if i+3 < len(fields) && fields[i+2] == "burst" {
							limit.IngressBurst = strings.ToLower(fields[i+3])
							logging.Debug("Found ingress burst", "burst", limit.IngressBurst)
						}
					} else if strings.Contains(line, "1:20") {
						limit.EgressRate = strings.ToLower(fields[i+1])
						logging.Debug("Found egress rate", "rate", limit.EgressRate)
						if i+3 < len(fields) && fields[i+2] == "burst" {
							limit.EgressBurst = strings.ToLower(fields[i+3])
							logging.Debug("Found egress burst", "burst", limit.EgressBurst)
						}
					}
				}
			}
		}
	}

	if limit.IngressRate == "" && limit.EgressRate == "" {
		logging.Debug("No bandwidth limits found",
			"container", name,
			"interface", iface,
			"output", output.String())
		return nil, fmt.Errorf("no bandwidth limits found for container %s interface %s", name, iface)
	}

	logging.Debug("Get network bandwidth limits",
		"container", name,
		"interface", iface,
		"ingress_rate", limit.IngressRate,
		"ingress_burst", limit.IngressBurst,
		"egress_rate", limit.EgressRate,
		"egress_burst", limit.EgressBurst,
	)

	return limit, nil
}

// UpdateNetworkBandwidthLimits updates bandwidth limits for a container's network interface
func (m *LXCManager) UpdateNetworkBandwidthLimits(name, iface string, limits *common.BandwidthLimit) error {
	// Ensure container exists
	containerPath := filepath.Join(m.configPath, name)
	if _, err := os.Stat(containerPath); os.IsNotExist(err) {
		return fmt.Errorf("container %s does not exist", name)
	}

	// Generate traffic control commands to update limits
	cmd := ExecCommand("lxc-attach", "-n", name, "--",
		"tc", "class", "replace", "dev", iface,
		"parent", "1:", "classid", "1:10",
		"htb", "rate", limits.IngressRate,
		"burst", limits.IngressBurst)

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to update ingress bandwidth limit: %w", err)
	}

	cmd = ExecCommand("lxc-attach", "-n", name, "--",
		"tc", "class", "replace", "dev", iface,
		"parent", "1:", "classid", "1:20",
		"htb", "rate", limits.EgressRate,
		"burst", limits.EgressBurst)

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to update egress bandwidth limit: %w", err)
	}

	logging.Debug("Updated network bandwidth limits",
		"container", name,
		"interface", iface,
		"ingress_rate", limits.IngressRate,
		"ingress_burst", limits.IngressBurst,
		"egress_rate", limits.EgressRate,
		"egress_burst", limits.EgressBurst,
	)

	return nil
}
