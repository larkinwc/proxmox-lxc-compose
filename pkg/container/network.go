package container

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"proxmox-lxc-compose/pkg/config"
	"proxmox-lxc-compose/pkg/logging"
)

// configureNetwork configures network settings for a container
func (m *LXCManager) configureNetwork(name string, cfg *config.NetworkConfig) error {
	if cfg == nil {
		return nil
	}

	logging.Debug("Configuring network", "container", name)
	configPath := filepath.Join(m.configPath, name, "network")
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
		cfg.Interfaces = append(cfg.Interfaces, legacy)
	}

	// Configure network isolation if enabled
	if cfg.Isolated {
		lines = append(lines, "lxc.net.0.flags = down")
		logging.Debug("Network isolation enabled", "container", name)
	}

	// Configure interfaces
	for i, iface := range cfg.Interfaces {
		prefix := fmt.Sprintf("lxc.net.%d", i)
		ifaceLines := []string{
			fmt.Sprintf("%s.type = %s", prefix, iface.Type),
		}

		if iface.Type == "bridge" {
			ifaceLines = append(ifaceLines, fmt.Sprintf("%s.link = %s", prefix, iface.Bridge))
		}

		if iface.Interface != "" {
			ifaceLines = append(ifaceLines, fmt.Sprintf("%s.name = %s", prefix, iface.Interface))
		}

		if iface.DHCP {
			ifaceLines = append(ifaceLines,
				fmt.Sprintf("%s.ipv4.method = dhcp", prefix),
				fmt.Sprintf("%s.ipv6.method = dhcp", prefix),
			)
		} else if iface.IP != "" {
			ifaceLines = append(ifaceLines, fmt.Sprintf("%s.ipv4.address = %s", prefix, iface.IP))
			if iface.Gateway != "" {
				ifaceLines = append(ifaceLines, fmt.Sprintf("%s.ipv4.gateway = %s", prefix, iface.Gateway))
			}
		}

		// Interface-specific DNS servers
		for j, dns := range iface.DNS {
			ifaceLines = append(ifaceLines, fmt.Sprintf("%s.ipv4.nameserver.%d = %s", prefix, j, dns))
		}

		if iface.Hostname != "" {
			ifaceLines = append(ifaceLines, fmt.Sprintf("%s.hostname = %s", prefix, iface.Hostname))
		}
		if iface.MTU > 0 {
			ifaceLines = append(ifaceLines, fmt.Sprintf("%s.mtu = %d", prefix, iface.MTU))
		}
		if iface.MAC != "" {
			ifaceLines = append(ifaceLines, fmt.Sprintf("%s.hwaddr = %s", prefix, iface.MAC))
		}

		lines = append(lines, ifaceLines...)
	}

	// Configure global DNS settings
	for i, dns := range cfg.DNSServers {
		lines = append(lines, fmt.Sprintf("lxc.net.dns.%d = %s", i, dns))
	}

	// Configure search domains
	if len(cfg.SearchDomains) > 0 {
		lines = append(lines, fmt.Sprintf("lxc.net.search_domains = %s", strings.Join(cfg.SearchDomains, " ")))
	}

	// Configure port forwarding
	for _, pf := range cfg.PortForwards {
		rule := fmt.Sprintf("lxc.net.port_forward = %s:%d:%d",
			strings.ToLower(pf.Protocol),
			pf.Host,
			pf.Guest,
		)
		lines = append(lines, rule)
		logging.Debug("Adding port forward",
			"container", name,
			"protocol", pf.Protocol,
			"host_port", pf.Host,
			"guest_port", pf.Guest,
		)
	}

	logging.Debug("Writing network configuration",
		"container", name,
		"path", configPath,
		"lines", len(lines),
	)

	return os.WriteFile(configPath, []byte(strings.Join(lines, "\n")+"\n"), 0644)
}

// getNetworkConfig reads network configuration from a container's config file
func (m *LXCManager) getNetworkConfig(name string) (*config.NetworkConfig, error) {
	logging.Debug("Reading network configuration", "container", name)

	configPath := filepath.Join(m.configPath, name, "network")
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
