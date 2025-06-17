package validation

import (
	"fmt"
	"net"
	"regexp"
	"strconv"
	"strings"

	"github.com/larkinwc/proxmox-lxc-compose/pkg/config"
)

// ValidateIPAddress validates an IP address (with optional CIDR)
func ValidateIPAddress(ip string) error {
	if ip == "" {
		return nil
	}

	// Try parsing as CIDR first
	if _, _, err := net.ParseCIDR(ip); err == nil {
		return nil
	}

	// Try parsing as plain IP
	if net.ParseIP(ip) == nil {
		return fmt.Errorf("invalid IP address: %s", ip)
	}

	return nil
}

// ValidateMACAddress validates a MAC address
func ValidateMACAddress(mac string) error {
	if mac == "" {
		return nil
	}

	// MAC address pattern: 6 groups of 2 hex digits separated by colons
	macPattern := regexp.MustCompile(`^([0-9a-fA-F]{2}:){5}[0-9a-fA-F]{2}$`)
	if !macPattern.MatchString(mac) {
		return fmt.Errorf("invalid MAC address: %s", mac)
	}

	return nil
}

// ValidateStorageSize validates a storage size string
func ValidateStorageSize(size string) (int64, error) {
	if size == "" {
		return 0, nil
	}

	size = strings.ToUpper(size)
	value, err := strconv.ParseInt(strings.TrimRight(size, "B"), 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid size format: %s", size)
	}

	if strings.HasSuffix(size, "G") {
		return value * 1024 * 1024 * 1024, nil
	}
	if strings.HasSuffix(size, "M") {
		return value * 1024 * 1024, nil
	}
	if strings.HasSuffix(size, "K") {
		return value * 1024, nil
	}

	return value, nil
}

// FormatBytes formats bytes into a human-readable string
func FormatBytes(bytes int64) string {
	if bytes >= 1024*1024*1024 {
		return fmt.Sprintf("%dG", bytes/(1024*1024*1024))
	}
	if bytes >= 1024*1024 {
		return fmt.Sprintf("%dM", bytes/(1024*1024))
	}
	if bytes >= 1024 {
		return fmt.Sprintf("%dK", bytes/1024)
	}
	return fmt.Sprintf("%dB", bytes)
}

// ValidateNetworkConfig validates a network configuration
func ValidateNetworkConfig(cfg *config.NetworkConfig) error {
	if cfg == nil {
		return nil
	}

	if cfg.Type != "" && cfg.Type != "bridge" && cfg.Type != "veth" {
		return fmt.Errorf("invalid network type: %s", cfg.Type)
	}

	for i, iface := range cfg.Interfaces {
		if iface.Type != "" && iface.Type != "bridge" && iface.Type != "veth" {
			return fmt.Errorf("interface %d: invalid network type: %s", i, iface.Type)
		}
	}

	return nil
}

// ValidateDeviceConfig validates a device configuration
func ValidateDeviceConfig(cfg *config.DeviceConfig) error {
	if cfg == nil {
		return nil
	}
	if cfg.Name == "" {
		return fmt.Errorf("device name is required")
	}
	if cfg.Type == "" {
		return fmt.Errorf("device type is required")
	}
	return nil
}

// ValidateStorageConfig validates a storage configuration
func ValidateStorageConfig(cfg *config.StorageConfig) error {
	if cfg == nil {
		return nil
	}
	if cfg.Root != "" {
		if _, err := ValidateStorageSize(cfg.Root); err != nil {
			return err
		}
	}
	return nil
}
