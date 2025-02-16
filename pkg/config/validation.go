package config

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// validateStorage validates storage configuration
func validateStorage(cfg *StorageConfig) error {
	if cfg.Root == "" {
		return fmt.Errorf("root storage size is required")
	}
	// Convert size to bytes for validation
	bytes, err := parseSize(cfg.Root)
	if err != nil {
		return fmt.Errorf("invalid root storage size: %w", err)
	}
	// Size must be at least 1MB
	if bytes < 1024*1024 {
		return fmt.Errorf("root storage size must be at least 1MB")
	}
	return nil
}

// parseSize converts a size string (e.g., "10G") to bytes
func parseSize(size string) (int64, error) {
	sizeRegex := regexp.MustCompile(`(?i)^(\d+)([KMGTP]B?)?$`)
	match := sizeRegex.FindStringSubmatch(size)
	if match == nil {
		return 0, fmt.Errorf("invalid size format")
	}

	value, err := strconv.ParseInt(match[1], 10, 64)
	if err != nil {
		return 0, err
	}

	unit := strings.ToUpper(strings.TrimSuffix(match[2], "B"))
	switch unit {
	case "K":
		value *= 1024
	case "M":
		value *= 1024 * 1024
	case "G":
		value *= 1024 * 1024 * 1024
	case "T":
		value *= 1024 * 1024 * 1024 * 1024
	case "P":
		value *= 1024 * 1024 * 1024 * 1024 * 1024
	}

	return value, nil
}

// validateNetwork validates network configuration
func validateNetwork(cfg *NetworkConfig) error {
	if cfg.Type != "" && !isValidNetworkType(cfg.Type) {
		return fmt.Errorf("invalid network type: %s", cfg.Type)
	}
	if cfg.Type == "bridge" && cfg.Bridge == "" {
		return fmt.Errorf("bridge name is required for bridge network type")
	}
	if cfg.IP != "" {
		if err := validateIP(cfg.IP); err != nil {
			return fmt.Errorf("invalid IP address: %w", err)
		}
	}
	return nil
}

// validateIP validates an IP address with optional CIDR notation
func validateIP(ip string) error {
	parts := strings.Split(ip, "/")
	if len(parts) > 2 {
		return fmt.Errorf("invalid IP format")
	}
	// TODO: Add proper IP validation
	return nil
}

// isValidNetworkType checks if a network type is valid
func isValidNetworkType(t string) bool {
	validTypes := map[string]bool{
		"none":    true,
		"veth":    true,
		"bridge":  true,
		"macvlan": true,
		"phys":    true,
	}
	return validTypes[strings.ToLower(t)]
}

// validateSecurity validates security configuration
func validateSecurity(cfg *SecurityConfig) error {
	if cfg.Isolation != "" {
		switch strings.ToLower(cfg.Isolation) {
		case "default", "strict", "privileged":
			// Valid values
		default:
			return fmt.Errorf("invalid isolation level: %s", cfg.Isolation)
		}
	}
	if cfg.Privileged && strings.ToLower(cfg.Isolation) == "strict" {
		return fmt.Errorf("cannot use privileged mode with strict isolation")
	}
	for _, cap := range cfg.Capabilities {
		if !isValidCapability(cap) {
			return fmt.Errorf("invalid capability: %s", cap)
		}
	}
	return nil
}

// isValidCapability checks if a Linux capability is valid
func isValidCapability(cap string) bool {
	fmt.Printf("DEBUG: Checking capability: %s\n", cap)

	// First try exact match
	if validCaps[strings.ToUpper(cap)] {
		return true
	}

	// Try with CAP_ prefix if not present
	if !strings.HasPrefix(strings.ToUpper(cap), "CAP_") {
		withPrefix := "CAP_" + strings.ToUpper(cap)
		fmt.Printf("DEBUG: Checking with CAP_ prefix: %s\n", withPrefix)
		return validCaps[withPrefix]
	}

	// Try without CAP_ prefix if present
	if strings.HasPrefix(strings.ToUpper(cap), "CAP_") {
		withoutPrefix := strings.TrimPrefix(strings.ToUpper(cap), "CAP_")
		fmt.Printf("DEBUG: Checking without CAP_ prefix: %s\n", withoutPrefix)
		return validCaps[withoutPrefix]
	}

	return false
}

var validCaps = map[string]bool{
	// Standard capabilities
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

	// With CAP_ prefix
	"CAP_CHOWN":            true,
	"CAP_DAC_OVERRIDE":     true,
	"CAP_DAC_READ_SEARCH":  true,
	"CAP_FOWNER":           true,
	"CAP_FSETID":           true,
	"CAP_KILL":             true,
	"CAP_SETGID":           true,
	"CAP_SETUID":           true,
	"CAP_SETPCAP":          true,
	"CAP_LINUX_IMMUTABLE":  true,
	"CAP_NET_BIND_SERVICE": true,
	"CAP_NET_BROADCAST":    true,
	"CAP_NET_ADMIN":        true,
	"CAP_NET_RAW":          true,
	"CAP_IPC_LOCK":         true,
	"CAP_IPC_OWNER":        true,
	"CAP_SYS_MODULE":       true,
	"CAP_SYS_RAWIO":        true,
	"CAP_SYS_CHROOT":       true,
	"CAP_SYS_PTRACE":       true,
	"CAP_SYS_PACCT":        true,
	"CAP_SYS_ADMIN":        true,
	"CAP_SYS_BOOT":         true,
	"CAP_SYS_NICE":         true,
	"CAP_SYS_RESOURCE":     true,
	"CAP_SYS_TIME":         true,
	"CAP_SYS_TTY_CONFIG":   true,
	"CAP_MKNOD":            true,
	"CAP_LEASE":            true,
	"CAP_AUDIT_WRITE":      true,
	"CAP_AUDIT_CONTROL":    true,
	"CAP_SETFCAP":          true,
	"CAP_MAC_OVERRIDE":     true,
	"CAP_MAC_ADMIN":        true,
	"CAP_SYSLOG":           true,
	"CAP_WAKE_ALARM":       true,
	"CAP_BLOCK_SUSPEND":    true,
	"CAP_AUDIT_READ":       true,
}

// validateContainerConfig validates the complete container configuration
func validateContainerConfig(container *Container) error {
	if container == nil {
		return fmt.Errorf("container configuration is required")
	}

	// Validate storage configuration
	if container.Storage != nil {
		if err := validateStorage(container.Storage); err != nil {
			return fmt.Errorf("invalid storage configuration: %w", err)
		}
	}

	// Validate network configuration
	if container.Network != nil {
		if err := validateNetwork(container.Network); err != nil {
			return fmt.Errorf("invalid network configuration: %w", err)
		}
	}

	// Validate security configuration
	if container.Security != nil {
		if err := validateSecurity(container.Security); err != nil {
			return fmt.Errorf("invalid security configuration: %w", err)
		}
	}

	return nil
}
