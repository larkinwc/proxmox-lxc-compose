package container

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"proxmox-lxc-compose/pkg/config"
)

// applyConfig applies the container configuration
func (m *LXCManager) applyConfig(name string, cfg *config.Container) error {
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

func (m *LXCManager) applyCPUConfig(f *os.File, cfg *config.CPUConfig) error {
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
		if err := writeConfig(f, "lxc.cpu.nr_cpus", fmt.Sprintf("%d", *cfg.Cores)); err != nil {
			return err
		}
	}

	return nil
}

func (m *LXCManager) applyMemoryConfig(f *os.File, cfg *config.MemoryConfig) error {
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

func (m *LXCManager) applyNetworkConfig(f *os.File, cfg *config.NetworkConfig) error {
	if cfg == nil {
		return nil
	}

	if err := writeConfig(f, "lxc.net.0.type", cfg.Type); err != nil {
		return err
	}

	if cfg.Bridge != "" {
		if err := writeConfig(f, "lxc.net.0.link", cfg.Bridge); err != nil {
			return err
		}
	}

	if cfg.IP != "" {
		if err := writeConfig(f, "lxc.net.0.ipv4.address", cfg.IP); err != nil {
			return err
		}
	}

	if cfg.Gateway != "" {
		if err := writeConfig(f, "lxc.net.0.ipv4.gateway", cfg.Gateway); err != nil {
			return err
		}
	}

	if len(cfg.DNS) > 0 {
		for i, dns := range cfg.DNS {
			key := fmt.Sprintf("lxc.net.0.ipv4.nameserver.%d", i)
			if err := writeConfig(f, key, dns); err != nil {
				return err
			}
		}
	}

	return nil
}

func (m *LXCManager) applyStorageConfig(f *os.File, cfg *config.StorageConfig) error {
	if cfg == nil {
		return nil
	}

	// Apply root storage configuration
	if cfg.Root != "" {
		if err := writeConfig(f, "lxc.rootfs.size", cfg.Root); err != nil {
			return err
		}
	}

	// Apply storage backend configuration
	if cfg.Backend != "" {
		if err := writeConfig(f, "lxc.rootfs.backend", cfg.Backend); err != nil {
			return err
		}
	}

	// Apply storage pool configuration if specified
	if cfg.Pool != "" {
		if err := writeConfig(f, "lxc.rootfs.pool", cfg.Pool); err != nil {
			return err
		}
	}

	// Configure automount if enabled
	if cfg.AutoMount {
		if err := writeConfig(f, "lxc.rootfs.mount.auto", "1"); err != nil {
			return err
		}
	}

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
