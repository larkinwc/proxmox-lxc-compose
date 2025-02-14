package config

import (
	"fmt"
	"proxmox-lxc-compose/pkg/validation"
)

func validateContainerConfig(container *Container) error {
	if container == nil {
		return fmt.Errorf("container configuration is required")
	}

	// Validate storage size
	if container.Storage != nil {
		bytes, err := validation.ValidateStorageSize(container.Storage.Root)
		if err != nil {
			return fmt.Errorf("invalid storage size: %w", err)
		}
		container.Storage.Root = validation.FormatBytes(bytes)
	}

	// Validate network configuration
	if container.Network != nil {
		networkCfg := &validation.NetworkConfig{
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
		err := validation.ValidateNetworkConfig(networkCfg)
		if err != nil {
			return fmt.Errorf("invalid network configuration: %w", err)
		}
	}

	// Validate device configuration
	if len(container.Devices) > 0 {
		for _, device := range container.Devices {
			err := validation.ValidateDevice(device.Name, device.Type, device.Source, device.Destination, device.Options)
			if err != nil {
				return fmt.Errorf("invalid device configuration: %w", err)
			}
		}
	}

	// Validate security configuration
	if container.Security != nil {
		if err := validateSecurityConfig(container.Security); err != nil {
			return fmt.Errorf("invalid security configuration: %w", err)
		}
	}

	return nil
}

// validateSecurityConfig validates security configuration
func validateSecurityConfig(cfg *SecurityConfig) error {
	// Validate isolation level
	if cfg.Isolation != "" {
		switch cfg.Isolation {
		case "default", "strict", "privileged":
			// Valid values
		default:
			return fmt.Errorf("invalid isolation level: %s", cfg.Isolation)
		}
	}

	// Privileged mode and strict isolation are mutually exclusive
	if cfg.Privileged && cfg.Isolation == "strict" {
		return fmt.Errorf("cannot use privileged mode with strict isolation")
	}

	// Validate capabilities
	for _, cap := range cfg.Capabilities {
		if !isValidCapability(cap) {
			return fmt.Errorf("invalid capability: %s", cap)
		}
	}

	return nil
}

// isValidCapability checks if a Linux capability is valid
func isValidCapability(capability string) bool {
	validCaps := map[string]bool{
		"CHOWN":            true,
		"DAC_OVERRIDE":     true,
		"DAC_READ_SEARCH":  true,
		"FOWNER":           true,
		"FSETID":           true,
		"KILL":             true,
		"SETGID":           true,
		"SETUID":           true,
		"SETPCAP":          true,
		"LINUX_IMMUTABLE":  true,
		"NET_BIND_SERVICE": true,
		"NET_BROADCAST":    true,
		"NET_ADMIN":        true,
		"NET_RAW":          true,
		"IPC_LOCK":         true,
		"IPC_OWNER":        true,
		"SYS_MODULE":       true,
		"SYS_RAWIO":        true,
		"SYS_CHROOT":       true,
		"SYS_PTRACE":       true,
		"SYS_PACCT":        true,
		"SYS_ADMIN":        true,
		"SYS_BOOT":         true,
		"SYS_NICE":         true,
		"SYS_RESOURCE":     true,
		"SYS_TIME":         true,
		"SYS_TTY_CONFIG":   true,
		"MKNOD":            true,
		"LEASE":            true,
		"AUDIT_WRITE":      true,
		"AUDIT_CONTROL":    true,
		"SETFCAP":          true,
		"MAC_OVERRIDE":     true,
		"MAC_ADMIN":        true,
		"SYSLOG":           true,
		"WAKE_ALARM":       true,
		"BLOCK_SUSPEND":    true,
		"AUDIT_READ":       true,
	}
	return validCaps[capability]
}
