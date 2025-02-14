package container

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"proxmox-lxc-compose/pkg/config"
)

// configureNetwork configures network settings for a container
func (m *LXCManager) configureNetwork(name string, cfg *config.NetworkConfig) error {
	if cfg == nil {
		return nil
	}

	configPath := filepath.Join(m.configPath, name, "network")
	var lines []string

	// Basic network config
	lines = append(lines,
		fmt.Sprintf("lxc.net.0.type = %s", cfg.Type),
		fmt.Sprintf("lxc.net.0.link = %s", cfg.Bridge),
		fmt.Sprintf("lxc.net.0.name = %s", cfg.Interface),
	)

	// IP configuration
	if cfg.DHCP {
		lines = append(lines,
			"lxc.net.0.ipv4.method = dhcp",
			"lxc.net.0.ipv6.method = dhcp",
		)
	} else if cfg.IP != "" {
		lines = append(lines, fmt.Sprintf("lxc.net.0.ipv4.address = %s", cfg.IP))
		if cfg.Gateway != "" {
			lines = append(lines, fmt.Sprintf("lxc.net.0.ipv4.gateway = %s", cfg.Gateway))
		}
	}

	// DNS servers
	for i, dns := range cfg.DNS {
		lines = append(lines, fmt.Sprintf("lxc.net.0.ipv4.nameserver.%d = %s", i, dns))
	}

	// Optional settings
	if cfg.Hostname != "" {
		lines = append(lines, fmt.Sprintf("lxc.net.0.hostname = %s", cfg.Hostname))
	}
	if cfg.MTU > 0 {
		lines = append(lines, fmt.Sprintf("lxc.net.0.mtu = %d", cfg.MTU))
	}
	if cfg.MAC != "" {
		lines = append(lines, fmt.Sprintf("lxc.net.0.hwaddr = %s", cfg.MAC))
	}

	// Write config file
	return os.WriteFile(configPath, []byte(strings.Join(lines, "\n")+"\n"), 0644)
}

// getNetworkConfig reads network configuration from a container's config file
func (m *LXCManager) getNetworkConfig(name string) (*config.NetworkConfig, error) {
	configPath := filepath.Join(m.configPath, name, "network")
	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read network config: %w", err)
	}

	cfg := &config.NetworkConfig{}
	var dns []string

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

		switch key {
		case "lxc.net.0.type":
			cfg.Type = value
		case "lxc.net.0.link":
			cfg.Bridge = value
		case "lxc.net.0.name":
			cfg.Interface = value
		case "lxc.net.0.ipv4.method":
			cfg.DHCP = value == "dhcp"
		case "lxc.net.0.ipv4.address":
			cfg.IP = value
		case "lxc.net.0.ipv4.gateway":
			cfg.Gateway = value
		case "lxc.net.0.hostname":
			cfg.Hostname = value
		case "lxc.net.0.mtu":
			if mtu, err := strconv.Atoi(value); err == nil {
				cfg.MTU = mtu
			}
		case "lxc.net.0.hwaddr":
			cfg.MAC = value
		default:
			if strings.HasPrefix(key, "lxc.net.0.ipv4.nameserver.") {
				dns = append(dns, value)
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to parse network config: %w", err)
	}

	if len(dns) > 0 {
		cfg.DNS = dns
	}

	return cfg, nil
}
