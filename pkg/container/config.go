package container

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/larkinwc/proxmox-lxc-compose/pkg/common"
	"github.com/larkinwc/proxmox-lxc-compose/pkg/logging"
)

// applyConfig applies the container configuration
func (m *LXCManager) applyConfig(name string, cfg *common.Container) error {
	configPath := filepath.Join(m.configPath, name, "config")

	// Create config file if it doesn't exist
	if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	f, err := os.Create(configPath)
	if err != nil {
		return fmt.Errorf("failed to create config file: %w", err)
	}
	defer f.Close()

	// Write base configuration
	if err := writeConfig(f, "lxc.uts.name", name); err != nil {
		return err
	}

	// Set rootfs path
	rootfsPath := filepath.Join(m.configPath, name, "rootfs")
	if err := writeConfig(f, "lxc.rootfs.path", fmt.Sprintf("dir:%s", rootfsPath)); err != nil {
		return err
	}

	// Add essential LXC configuration settings
	if err := writeConfig(f, "lxc.mount.auto", "proc:mixed sys:ro cgroup:mixed"); err != nil {
		return err
	}
	if err := writeConfig(f, "lxc.tty.max", "4"); err != nil {
		return err
	}
	if err := writeConfig(f, "lxc.pty.max", "1024"); err != nil {
		return err
	}

	// Apply security configuration
	if err := m.applySecurityConfig(f, cfg.Security); err != nil {
		return err
	}

	// Apply resource limits
	if err := m.applyCPUConfig(f, cfg.CPU); err != nil {
		return err
	}
	if err := m.applyMemoryConfig(f, cfg.Memory); err != nil {
		return err
	}

	// Apply network configuration
	if err := m.applyNetworkConfig(f, cfg.Network); err != nil {
		return err
	}

	// Apply storage configuration
	if err := m.applyStorageConfig(f, cfg.Storage); err != nil {
		return err
	}

	// Apply environment variables and entrypoint configuration
	if err := m.applyEnvironmentConfig(f, cfg.Environment); err != nil {
		return err
	}

	if err := m.applyEntrypointConfig(f, cfg.Entrypoint, cfg.Command); err != nil {
		return err
	}

	return nil
}

func (m *LXCManager) applyCPUConfig(f *os.File, cfg *common.CPUConfig) error {
	if cfg == nil {
		return nil
	}

	if cfg.Shares != nil {
		if err := writeConfig(f, "lxc.cpu.shares", fmt.Sprintf("%d", *cfg.Shares)); err != nil {
			return err
		}
	}

	if cfg.Quota != nil {
		if err := writeConfig(f, "lxc.cpu.cfs_quota_us", fmt.Sprintf("%d", *cfg.Quota)); err != nil {
			return err
		}
	}

	if cfg.Period != nil {
		if err := writeConfig(f, "lxc.cpu.cfs_period_us", fmt.Sprintf("%d", *cfg.Period)); err != nil {
			return err
		}
	}

	if cfg.Cores != nil {
		// Use cgroup CPU shares which is widely supported across LXC versions
		if err := writeConfig(f, "lxc.cgroup.cpu.shares", fmt.Sprintf("%d", *cfg.Cores*1024)); err != nil {
			return err
		}
	}

	return nil
}

func (m *LXCManager) applyMemoryConfig(f *os.File, cfg *common.MemoryConfig) error {
	if cfg == nil {
		return nil
	}

	if cfg.Limit != "" {
		if err := writeConfig(f, "lxc.cgroup.memory.limit_in_bytes", cfg.Limit); err != nil {
			return err
		}
	}

	if cfg.Swap != "" {
		if err := writeConfig(f, "lxc.cgroup.memory.memsw.limit_in_bytes", cfg.Swap); err != nil {
			return err
		}
	}

	return nil
}

func (m *LXCManager) applyNetworkConfig(f *os.File, cfg *common.NetworkConfig) error {
	if cfg == nil {
		return nil
	}

	// Write network type - use veth for better compatibility across LXC versions
	networkType := cfg.Type
	if networkType == "bridge" {
		// For bridge networking, use veth type which is more widely supported
		networkType = "veth"
	}
	if err := writeConfig(f, "lxc.net.0.type", networkType); err != nil {
		return err
	}

	// Write bridge if specified (required for veth type)
	if cfg.Bridge != "" {
		if err := writeConfig(f, "lxc.net.0.link", cfg.Bridge); err != nil {
			return err
		}
	} else if networkType == "veth" {
		// Default to lxcbr0 if no bridge specified for veth
		if err := writeConfig(f, "lxc.net.0.link", "lxcbr0"); err != nil {
			return err
		}
	}

	// Write interface name if specified
	if cfg.Interface != "" {
		if err := writeConfig(f, "lxc.net.0.name", cfg.Interface); err != nil {
			return err
		}
	}

	// Write IP configuration
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
	}

	// Write gateway if specified
	if cfg.Gateway != "" {
		if err := writeConfig(f, "lxc.net.0.ipv4.gateway", cfg.Gateway); err != nil {
			return err
		}
	}

	// Skip DNS configuration for older LXC versions compatibility
	// DNS can be configured manually inside the container or through other means
	// TODO: Implement DNS configuration for older LXC versions

	// Write hostname if specified
	if cfg.Hostname != "" {
		if err := writeConfig(f, "lxc.net.0.hostname", cfg.Hostname); err != nil {
			return err
		}
	}

	// Write MTU if specified
	if cfg.MTU > 0 {
		if err := writeConfig(f, "lxc.net.0.mtu", fmt.Sprintf("%d", cfg.MTU)); err != nil {
			return err
		}
	}

	// Write MAC address if specified
	if cfg.MAC != "" {
		if err := writeConfig(f, "lxc.net.0.hwaddr", cfg.MAC); err != nil {
			return err
		}
	}

	// Configure additional interfaces
	if len(cfg.Interfaces) > 0 {
		for i, iface := range cfg.Interfaces {
			prefix := fmt.Sprintf("lxc.net.%d", i)

			// Use veth for better compatibility
			ifaceType := iface.Type
			if ifaceType == "bridge" {
				ifaceType = "veth"
			}
			if err := writeConfig(f, prefix+".type", ifaceType); err != nil {
				return err
			}

			if iface.Bridge != "" {
				if err := writeConfig(f, prefix+".link", iface.Bridge); err != nil {
					return err
				}
			} else if ifaceType == "veth" {
				// Default to lxcbr0 if no bridge specified for veth
				if err := writeConfig(f, prefix+".link", "lxcbr0"); err != nil {
					return err
				}
			}

			if iface.Interface != "" {
				if err := writeConfig(f, prefix+".name", iface.Interface); err != nil {
					return err
				}
			}

			if iface.DHCP {
				if err := writeConfig(f, prefix+".ipv4.method", "dhcp"); err != nil {
					return err
				}
			} else if iface.IP != "" {
				if err := writeConfig(f, prefix+".ipv4.address", iface.IP); err != nil {
					return err
				}
			}

			if iface.Gateway != "" {
				if err := writeConfig(f, prefix+".ipv4.gateway", iface.Gateway); err != nil {
					return err
				}
			}

			// Skip DNS for additional interfaces - use the main interface DNS
			// Additional interfaces in older LXC versions don't support DNS configuration

			if iface.MTU > 0 {
				if err := writeConfig(f, prefix+".mtu", fmt.Sprintf("%d", iface.MTU)); err != nil {
					return err
				}
			}

			if iface.MAC != "" {
				if err := writeConfig(f, prefix+".hwaddr", iface.MAC); err != nil {
					return err
				}
			}
		}
	}

	// Configure port forwarding
	if len(cfg.PortForwards) > 0 {
		for _, pf := range cfg.PortForwards {
			// Add pre-start hook for port forwarding
			preStartCmd := fmt.Sprintf("iptables -t nat -A PREROUTING -p %s --dport %d -j DNAT --to %s:%d",
				pf.Protocol, pf.Host, cfg.IP, pf.Guest)
			if err := writeConfig(f, "lxc.hook.pre-start", preStartCmd); err != nil {
				return err
			}

			// Add post-stop hook to clean up port forwarding
			postStopCmd := fmt.Sprintf("iptables -t nat -D PREROUTING -p %s --dport %d -j DNAT --to %s:%d",
				pf.Protocol, pf.Host, cfg.IP, pf.Guest)
			if err := writeConfig(f, "lxc.hook.post-stop", postStopCmd); err != nil {
				return err
			}
		}
	}

	return nil
}

func (m *LXCManager) applyStorageConfig(f *os.File, cfg *common.StorageConfig) error {
	if cfg == nil {
		return nil
	}

	// Skip rootfs size configuration for older LXC versions compatibility
	// Storage size should be managed at the filesystem/directory level
	// TODO: Implement storage size configuration for older LXC versions

	// Skip advanced storage configurations for older LXC versions compatibility
	// Backend, pool, and automount configurations are not widely supported in older versions
	// TODO: Implement advanced storage configuration for newer LXC versions

	// Apply additional mounts
	for i, mount := range cfg.Mounts {
		prefix := fmt.Sprintf("lxc.mount.entry.%d", i)

		mountOptions := "defaults"
		if len(mount.Options) > 0 {
			mountOptions = strings.Join(mount.Options, ",")
		}

		value := fmt.Sprintf("%s %s %s %s 0 0",
			mount.Source,
			mount.Target,
			mount.Type,
			mountOptions,
		)

		if err := writeConfig(f, prefix, value); err != nil {
			return err
		}
	}

	return nil
}

func (m *LXCManager) applySecurityConfig(f *os.File, cfg *common.SecurityConfig) error {
	if cfg == nil {
		// Apply default security settings
		return writeConfig(f, "lxc.apparmor.profile", "lxc-container-default")
	}

	// Write isolation level - but check if the config file exists first
	if cfg.Isolation != "" {
		isolationConfigPath := fmt.Sprintf("/usr/share/lxc/config/%s.conf", cfg.Isolation)
		if _, err := os.Stat(isolationConfigPath); err == nil {
			// File exists, include it
			if err := writeConfig(f, "lxc.include", isolationConfigPath); err != nil {
				return err
			}
		} else {
			// File doesn't exist, apply basic security settings instead
			logging.Debug("Isolation config file not found, using basic security settings",
				"path", isolationConfigPath, "container", "unknown")
		}
	} else {
		// No isolation specified, try to use common configs available on the system
		commonConfigs := []string{
			"/usr/share/lxc/config/common.conf",
			"/usr/share/lxc/config/ubuntu.common.conf",
		}

		for _, configPath := range commonConfigs {
			if _, err := os.Stat(configPath); err == nil {
				logging.Debug("Using available LXC common config", "path", configPath)
				if err := writeConfig(f, "lxc.include", configPath); err != nil {
					return err
				}
				break
			}
		}
	}

	// Write security configurations
	if cfg.Privileged {
		if err := writeConfig(f, "lxc.apparmor.profile", "unconfined"); err != nil {
			return err
		}
		if err := writeConfig(f, "lxc.cap.drop", ""); err != nil {
			return err
		}
	} else {
		if cfg.AppArmorProfile != "" {
			if err := writeConfig(f, "lxc.apparmor.profile", cfg.AppArmorProfile); err != nil {
				return err
			}
		}
		if cfg.SELinuxContext != "" {
			if err := writeConfig(f, "lxc.selinux.context", cfg.SELinuxContext); err != nil {
				return err
			}
		}
		if len(cfg.Capabilities) > 0 {
			if err := writeConfig(f, "lxc.cap.drop", "all"); err != nil {
				return err
			}
			if err := writeConfig(f, "lxc.cap.keep", strings.Join(cfg.Capabilities, " ")); err != nil {
				return err
			}
		}
	}

	// Apply seccomp profile if specified
	if cfg.SeccompProfile != "" {
		if err := writeConfig(f, "lxc.seccomp.profile", cfg.SeccompProfile); err != nil {
			return err
		}
	}

	return nil
}

func (m *LXCManager) applyEnvironmentConfig(f *os.File, env map[string]string) error {
	if len(env) == 0 {
		return nil
	}
	// Write environment variables to container config
	for key, value := range env {
		if err := writeConfig(f, "lxc.environment", fmt.Sprintf("%s=%s", key, value)); err != nil {
			return fmt.Errorf("failed to set environment variable %s: %w", key, err)
		}
	}
	return nil
}

func (m *LXCManager) applyEntrypointConfig(f *os.File, entrypoint, command []string) error {
	// If neither entrypoint nor command is set, return
	if len(entrypoint) == 0 && len(command) == 0 {
		return nil
	}

	// Combine entrypoint and command
	var cmd []string
	cmd = append(cmd, entrypoint...)
	cmd = append(cmd, command...)

	// Create the init script that will be executed when the container starts
	initScript := filepath.Join(m.configPath, "init.sh")
	if err := os.WriteFile(initScript, []byte(fmt.Sprintf(`#!/bin/sh
exec %s
`, strings.Join(cmd, " "))), 0755); err != nil {
		return fmt.Errorf("failed to create init script: %w", err)
	}

	// Set the init script as the container's init command
	if err := writeConfig(f, "lxc.init.cmd", initScript); err != nil {
		return fmt.Errorf("failed to set init command: %w", err)
	}
	return nil
}

func writeConfig(f *os.File, key, value string) error {
	_, err := fmt.Fprintf(f, "%s = %s\n", key, value)
	return err
}

func validateContainerConfig(container *common.Container) error {
	// Validate network configuration
	if container.Network != nil {
		if container.Network.Type != "" && container.Network.Type != "bridge" && container.Network.Type != "veth" {
			return fmt.Errorf("invalid network type: %s", container.Network.Type)
		}

		networkCfg := &common.NetworkConfig{
			Type:      container.Network.Type,
			Bridge:    container.Network.Bridge,
			Interface: container.Network.Interface,
			IP:        container.Network.IP,
			Gateway:   container.Network.Gateway,
			DNS:       container.Network.DNS,
			DHCP:      container.Network.DHCP,
			Hostname:  container.Network.Hostname,
			MTU:       container.Network.MTU,
			MAC:       container.Network.MAC,
		}
		err := common.ValidateNetworkConfig(networkCfg)
		if err != nil {
			return fmt.Errorf("invalid network configuration: %w", err)
		}
	}

	// ... existing code ...

	return nil
}

// ApplyConfig applies the container configuration
func (m *LXCManager) ApplyConfig(name string, cfg *common.Container) error {
	return m.applyConfig(name, cfg)
}

// ApplyCPUConfig applies CPU configuration to the container
func (m *LXCManager) ApplyCPUConfig(f *os.File, cfg *common.CPUConfig) error {
	return m.applyCPUConfig(f, cfg)
}

// ApplyMemoryConfig applies memory configuration to the container
func (m *LXCManager) ApplyMemoryConfig(f *os.File, cfg *common.MemoryConfig) error {
	return m.applyMemoryConfig(f, cfg)
}

// ApplyNetworkConfig applies network configuration to the container
func (m *LXCManager) ApplyNetworkConfig(f *os.File, cfg *common.NetworkConfig) error {
	return m.applyNetworkConfig(f, cfg)
}

// ApplyStorageConfig applies storage configuration to the container
func (m *LXCManager) ApplyStorageConfig(f *os.File, cfg *common.StorageConfig) error {
	return m.applyStorageConfig(f, cfg)
}

// ApplySecurityConfig applies security configuration to the container
func (m *LXCManager) ApplySecurityConfig(f *os.File, cfg *common.SecurityConfig) error {
	return m.applySecurityConfig(f, cfg)
}

// ApplyEnvironmentConfig applies environment variables to the container
func (m *LXCManager) ApplyEnvironmentConfig(f *os.File, env map[string]string) error {
	return m.applyEnvironmentConfig(f, env)
}

// ApplyEntrypointConfig applies entrypoint and command configuration to the container
func (m *LXCManager) ApplyEntrypointConfig(f *os.File, entrypoint, command []string) error {
	return m.applyEntrypointConfig(f, entrypoint, command)
}

// WriteConfig writes a configuration key-value pair to the file
func WriteConfig(f *os.File, key, value string) error {
	return writeConfig(f, key, value)
}
