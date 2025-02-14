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
