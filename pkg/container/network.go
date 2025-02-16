package container

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
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

// GetNetworkConfig reads network configuration from a container's config file
func (m *LXCManager) GetNetworkConfig(name string) (*config.NetworkConfig, error) {
	logging.Debug("Reading network configuration", "container", name)

	configPath := filepath.Join(m.configPath, name, "network", "config")
	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read network config: %w", err)
	}

	cfg := &config.NetworkConfig{
		Interfaces: make([]config.NetworkInterface, 0),
	}
	var currentIface *config.NetworkInterface
	var currentIndex = -1

	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		// Parse port forwarding
		if strings.HasPrefix(key, "lxc.net.port_forward") {
			parts := strings.Split(value, ":")
			if len(parts) == 3 {
				hostPort, _ := strconv.Atoi(parts[1])
				guestPort, _ := strconv.Atoi(parts[2])
				cfg.PortForwards = append(cfg.PortForwards, config.PortForward{
					Protocol: parts[0],
					Host:     hostPort,
					Guest:    guestPort,
				})
			}
			continue
		}

		// Parse interface configuration
		if strings.HasPrefix(key, "lxc.net.") {
			index := -1
			if n, err := fmt.Sscanf(key, "lxc.net.%d", &index); err == nil && n == 1 {
				if index != currentIndex {
					currentIndex = index
					currentIface = &config.NetworkInterface{}
					cfg.Interfaces = append(cfg.Interfaces, *currentIface)
				}
			}

			if currentIface != nil {
				switch {
				case strings.HasSuffix(key, ".type"):
					currentIface.Type = value
				case strings.HasSuffix(key, ".link"):
					currentIface.Bridge = value
				case strings.HasSuffix(key, ".name"):
					currentIface.Interface = value
				case strings.HasSuffix(key, ".ipv4.method"):
					currentIface.DHCP = value == "dhcp"
				case strings.HasSuffix(key, ".ipv4.address"):
					currentIface.IP = value
				case strings.HasSuffix(key, ".ipv4.gateway"):
					currentIface.Gateway = value
				case strings.HasSuffix(key, ".hostname"):
					currentIface.Hostname = value
				case strings.HasSuffix(key, ".mtu"):
					if mtu, err := strconv.Atoi(value); err == nil {
						currentIface.MTU = mtu
					}
				case strings.HasSuffix(key, ".hwaddr"):
					currentIface.MAC = value
				case strings.HasPrefix(key, "lxc.net."+strconv.Itoa(currentIndex)+".ipv4.nameserver."):
					currentIface.DNS = append(currentIface.DNS, value)
				}
			}
			continue
		}

		// Parse global DNS settings
		if strings.HasPrefix(key, "lxc.net.dns.") {
			cfg.DNSServers = append(cfg.DNSServers, value)
			continue
		}

		// Parse search domains
		if key == "lxc.net.search_domains" {
			cfg.SearchDomains = strings.Fields(value)
		}

		// Check for network isolation
		if key == "lxc.net.0.flags" && value == "down" {
			cfg.Isolated = true
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to parse network config: %w", err)
	}

	return cfg, nil
}
