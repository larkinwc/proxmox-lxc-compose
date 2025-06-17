package container

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/larkinwc/proxmox-lxc-compose/pkg/config"
	"github.com/larkinwc/proxmox-lxc-compose/pkg/logging"
)

// configureNetwork configures network settings for a container
func (m *LXCManager) configureNetwork(name string, cfg *config.NetworkConfig) error {
	if cfg == nil {
		return nil
	}

	logging.Debug("Configuring network", "container", name)
	containerDir := filepath.Join(m.configPath, name)
	if err := os.MkdirAll(containerDir, 0755); err != nil {
		return fmt.Errorf("failed to create container directory: %w", err)
	}
	configPath := filepath.Join(containerDir, "network.conf")

	// If the file exists, remove it first to avoid any issues
	if err := os.Remove(configPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove existing network config: %w", err)
	}

	var lines []string

	// Handle legacy configuration
	if cfg.Type != "" {
		logging.Debug("Using legacy network configuration", "container", name)
		legacy := config.NetworkInterface{
			Type:      cfg.Type,
			Bridge:    cfg.Bridge,
			Interface: cfg.Interface,
			IP:        cfg.IP,
			Gateway:   cfg.Gateway,
			DNS:       cfg.DNS,
			DHCP:      cfg.DHCP,
			Hostname:  cfg.Hostname,
			MTU:       cfg.MTU,
			MAC:       cfg.MAC,
		}
		cfg.Interfaces = append([]config.NetworkInterface{legacy}, cfg.Interfaces...)
	}

	// Configure network isolation if enabled
	if cfg.Isolated {
		lines = append(lines, "lxc.net.0.flags = down")
		return os.WriteFile(configPath, []byte(strings.Join(lines, "\n")+"\n"), 0644)
	}

	// Configure each network interface
	for i, iface := range cfg.Interfaces {
		// Skip empty interfaces
		if iface.Type == "" && iface.Interface == "" && iface.Bridge == "" {
			continue
		}

		// Default to veth type if not specified
		if iface.Type == "" {
			iface.Type = "veth"
		}

		prefix := fmt.Sprintf("lxc.net.%d", i)

		// Basic interface configuration
		lines = append(lines, fmt.Sprintf("%s.type = %s", prefix, iface.Type))

		if iface.Bridge != "" {
			lines = append(lines, fmt.Sprintf("%s.link = %s", prefix, iface.Bridge))
		}
		if iface.Interface != "" {
			lines = append(lines, fmt.Sprintf("%s.name = %s", prefix, iface.Interface))
		}

		// Add default flags
		lines = append(lines, fmt.Sprintf("%s.flags = up", prefix))

		// IP configuration
		if iface.DHCP {
			lines = append(lines, fmt.Sprintf("%s.ipv4.method = dhcp", prefix))
			lines = append(lines, fmt.Sprintf("%s.ipv6.method = dhcp", prefix))
		} else if iface.IP != "" {
			lines = append(lines, fmt.Sprintf("%s.ipv4.address = %s", prefix, iface.IP))
			if iface.Gateway != "" {
				lines = append(lines, fmt.Sprintf("%s.ipv4.gateway = %s", prefix, iface.Gateway))
			}
		}

		// DNS configuration
		for j, dns := range iface.DNS {
			lines = append(lines, fmt.Sprintf("%s.ipv4.nameserver.%d = %s", prefix, j, dns))
		}

		// Additional settings
		if iface.Hostname != "" {
			lines = append(lines, fmt.Sprintf("%s.hostname = %s", prefix, iface.Hostname))
		}
		if iface.MTU > 0 {
			lines = append(lines, fmt.Sprintf("%s.mtu = %d", prefix, iface.MTU))
		}
		if iface.MAC != "" {
			lines = append(lines, fmt.Sprintf("%s.hwaddr = %s", prefix, iface.MAC))
		}
	}

	// Configure port forwarding
	if len(cfg.PortForwards) > 0 {
		// Get the primary interface's IP (first interface with static IP)
		var containerIP string
		for _, iface := range cfg.Interfaces {
			if !iface.DHCP && iface.IP != "" {
				containerIP = strings.Split(iface.IP, "/")[0]
				break
			}
		}

		if containerIP == "" {
			return fmt.Errorf("port forwarding requires at least one interface with static IP")
		}

		// Add iptables rules for port forwarding
		for _, pf := range cfg.PortForwards {
			// Pre-start hook to set up forwarding
			preStartRule := fmt.Sprintf("lxc.hook.pre-start = iptables -t nat -A PREROUTING -p %s --dport %d -j DNAT --to %s:%d",
				pf.Protocol, pf.Host, containerIP, pf.Guest)
			lines = append(lines, preStartRule)

			// Post-stop hook to clean up forwarding
			postStopRule := fmt.Sprintf("lxc.hook.post-stop = iptables -t nat -D PREROUTING -p %s --dport %d -j DNAT --to %s:%d",
				pf.Protocol, pf.Host, containerIP, pf.Guest)
			lines = append(lines, postStopRule)
		}
	}

	if len(lines) == 0 {
		return nil
	}

	return os.WriteFile(configPath, []byte(strings.Join(lines, "\n")+"\n"), 0644)
}
