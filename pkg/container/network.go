package container

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"proxmox-lxc-compose/pkg/config"
)

// configureNetwork configures the network for a container
func (m *LXCManager) configureNetwork(name string, cfg *config.NetworkConfig) error {
	if cfg == nil {
		return nil
	}

	// Get container's network config file path
	networkConfigPath := filepath.Join(m.configPath, name, "network")

	// Create network config file
	f, err := os.Create(networkConfigPath)
	if err != nil {
		return fmt.Errorf("failed to create network config file: %w", err)
	}
	defer f.Close()

	// Write network type and interface
	if err := writeConfig(f, "lxc.net.0.type", cfg.Type); err != nil {
		return err
	}

	if cfg.Interface != "" {
		if err := writeConfig(f, "lxc.net.0.name", cfg.Interface); err != nil {
			return err
		}
	}

	// Configure bridge if specified
	if cfg.Bridge != "" {
		if err := writeConfig(f, "lxc.net.0.link", cfg.Bridge); err != nil {
			return err
		}
	}

	// Configure DHCP or static IP
	if cfg.DHCP {
		if err := writeConfig(f, "lxc.net.0.ipv4.method", "dhcp"); err != nil {
			return err
		}
		if err := writeConfig(f, "lxc.net.0.ipv6.method", "dhcp"); err != nil {
			return err
		}
	} else if cfg.IP != "" {
		if err := writeConfig(f, "lxc.net.0.ipv4.address", cfg.IP); err != nil {
			return err
		}
		if cfg.Gateway != "" {
			if err := writeConfig(f, "lxc.net.0.ipv4.gateway", cfg.Gateway); err != nil {
				return err
			}
		}
	}

	// Configure DNS servers
	if len(cfg.DNS) > 0 {
		for i, dns := range cfg.DNS {
			key := fmt.Sprintf("lxc.net.0.ipv4.nameserver.%d", i)
			if err := writeConfig(f, key, dns); err != nil {
				return err
			}
		}
	}

	// Configure hostname if specified
	if cfg.Hostname != "" {
		if err := writeConfig(f, "lxc.net.0.hostname", cfg.Hostname); err != nil {
			return err
		}
	}

	// Configure MTU if specified
	if cfg.MTU > 0 {
		if err := writeConfig(f, "lxc.net.0.mtu", fmt.Sprintf("%d", cfg.MTU)); err != nil {
			return err
		}
	}

	// Configure MAC address if specified
	if cfg.MAC != "" {
		if err := writeConfig(f, "lxc.net.0.hwaddr", strings.ToUpper(cfg.MAC)); err != nil {
			return err
		}
	}

	return nil
}

// getNetworkConfig retrieves the current network configuration of a container
func (m *LXCManager) getNetworkConfig(name string) (*config.NetworkConfig, error) {
	networkConfigPath := filepath.Join(m.configPath, name, "network")

	data, err := os.ReadFile(networkConfigPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read network config: %w", err)
	}

	cfg := &config.NetworkConfig{}
	lines := strings.Split(string(data), "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
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
		case "lxc.net.0.name":
			cfg.Interface = value
		case "lxc.net.0.link":
			cfg.Bridge = value
		case "lxc.net.0.ipv4.method":
			cfg.DHCP = value == "dhcp"
		case "lxc.net.0.ipv4.address":
			cfg.IP = value
		case "lxc.net.0.ipv4.gateway":
			cfg.Gateway = value
		case "lxc.net.0.hostname":
			cfg.Hostname = value
		case "lxc.net.0.mtu":
			mtu, _ := strconv.Atoi(value)
			cfg.MTU = mtu
		case "lxc.net.0.hwaddr":
			cfg.MAC = value
		}

		if strings.HasPrefix(key, "lxc.net.0.ipv4.nameserver.") {
			cfg.DNS = append(cfg.DNS, value)
		}
	}

	return cfg, nil
}
