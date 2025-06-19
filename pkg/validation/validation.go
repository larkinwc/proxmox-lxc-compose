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

	// Remove optional 'B' suffix first
	if strings.HasSuffix(size, "B") {
		size = size[:len(size)-1]
	}

	// Extract numeric part and unit
	var valueStr string
	var unit string

	if strings.HasSuffix(size, "G") {
		unit = "G"
		valueStr = size[:len(size)-1]
	} else if strings.HasSuffix(size, "M") {
		unit = "M"
		valueStr = size[:len(size)-1]
	} else if strings.HasSuffix(size, "K") {
		unit = "K"
		valueStr = size[:len(size)-1]
	} else {
		// No unit, just a number
		valueStr = size
	}

	value, err := strconv.ParseInt(valueStr, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid size format: %s", size)
	}

	if value < 0 {
		return 0, fmt.Errorf("invalid size format: %s", size)
	}

	switch unit {
	case "G":
		return value * 1024 * 1024 * 1024, nil
	case "M":
		return value * 1024 * 1024, nil
	case "K":
		return value * 1024, nil
	default:
		return value, nil
	}
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

	if cfg.Type != "" {
		if err := ValidateNetworkType(cfg.Type); err != nil {
			return err
		}
	}

	for i, iface := range cfg.Interfaces {
		if iface.Type != "" {
			if err := ValidateNetworkType(iface.Type); err != nil {
				return fmt.Errorf("interface %d: %s", i, err.Error())
			}
		}

		// Bridge type requires bridge name
		if iface.Type == "bridge" && iface.Bridge == "" {
			return fmt.Errorf("interface %d: bridge name is required for bridge network type", i)
		}

		if iface.IP != "" {
			if err := ValidateIPAddress(iface.IP); err != nil {
				return fmt.Errorf("interface %d: invalid IP address", i)
			}
		}

		if iface.Gateway != "" {
			if err := ValidateIPAddress(iface.Gateway); err != nil {
				return fmt.Errorf("interface %d: invalid gateway", i)
			}
		}

		if iface.MAC != "" {
			if err := ValidateMACAddress(iface.MAC); err != nil {
				return fmt.Errorf("interface %d: invalid MAC address", i)
			}
		}

		// Validate MTU if specified
		if iface.MTU != 0 && (iface.MTU < 68 || iface.MTU > 65535) {
			return fmt.Errorf("interface %d: MTU must be between 68 and 65535", i)
		}

		if err := ValidateDNSServers(iface.DNS); err != nil {
			return fmt.Errorf("interface %d: %s", i, err.Error())
		}
	}

	// Validate port forwards
	for i, pf := range cfg.PortForwards {
		if pf.Protocol != "tcp" && pf.Protocol != "udp" {
			return fmt.Errorf("port forward %d: protocol must be tcp or udp", i)
		}
		if pf.Host < 1 || pf.Host > 65535 {
			return fmt.Errorf("port forward %d: host port must be between 1 and 65535", i)
		}
		if pf.Guest < 1 || pf.Guest > 65535 {
			return fmt.Errorf("port forward %d: guest port must be between 1 and 65535", i)
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

// ValidateDeviceType validates a device type
func ValidateDeviceType(deviceType string) error {
	validTypes := []string{"disk", "nic", "usb", "serial", "console"}
	for _, t := range validTypes {
		if deviceType == t {
			return nil
		}
	}
	return fmt.Errorf("invalid device type: %s", deviceType)
}

// ValidateDeviceName validates a device name
func ValidateDeviceName(name string) error {
	if name == "" {
		return fmt.Errorf("device name cannot be empty")
	}
	// Device names should be alphanumeric with dashes and underscores
	pattern := regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)
	if !pattern.MatchString(name) {
		return fmt.Errorf("invalid device name: %s", name)
	}
	return nil
}

// ValidateDevicePath validates a device path
func ValidateDevicePath(path string) error {
	if path == "" {
		return fmt.Errorf("device path cannot be empty")
	}
	// Simple path validation - should start with /
	if !strings.HasPrefix(path, "/") {
		return fmt.Errorf("device path must be absolute: %s", path)
	}
	return nil
}

// ValidateDeviceOptions validates device options
func ValidateDeviceOptions(options []string) error {
	// Basic validation for device options
	for _, opt := range options {
		if opt == "" {
			return fmt.Errorf("device option cannot be empty")
		}
	}
	return nil
}

// ValidateDevice validates a complete device configuration
func ValidateDevice(device *config.DeviceConfig) error {
	if device == nil {
		return fmt.Errorf("device configuration cannot be nil")
	}

	if err := ValidateDeviceName(device.Name); err != nil {
		return err
	}

	if err := ValidateDeviceType(device.Type); err != nil {
		return err
	}

	if device.Source != "" {
		if err := ValidateDevicePath(device.Source); err != nil {
			return err
		}
	}

	if device.Destination != "" {
		if err := ValidateDevicePath(device.Destination); err != nil {
			return err
		}
	}

	return ValidateDeviceOptions(device.Options)
}

// ValidateNetworkType validates a network type
func ValidateNetworkType(networkType string) error {
	validTypes := []string{"bridge", "veth", "macvlan", "none"}
	for _, t := range validTypes {
		if networkType == t {
			return nil
		}
	}
	return fmt.Errorf("invalid network type: %s", networkType)
}

// ValidateDNSServers validates DNS server addresses
func ValidateDNSServers(servers []string) error {
	for _, server := range servers {
		if net.ParseIP(server) == nil {
			return fmt.Errorf("invalid DNS server address: %s", server)
		}
	}
	return nil
}

// ValidateNetworkInterface validates a network interface configuration
func ValidateNetworkInterface(iface *config.NetworkInterface) error {
	if iface == nil {
		return fmt.Errorf("network interface configuration cannot be nil")
	}

	if err := ValidateNetworkType(iface.Type); err != nil {
		return err
	}

	if iface.IP != "" {
		if err := ValidateIPAddress(iface.IP); err != nil {
			return err
		}
	}

	if iface.Gateway != "" {
		if err := ValidateIPAddress(iface.Gateway); err != nil {
			return err
		}
	}

	if iface.MAC != "" {
		if err := ValidateMACAddress(iface.MAC); err != nil {
			return err
		}
	}

	return ValidateDNSServers(iface.DNS)
}

// ValidateVPNConfig validates a VPN configuration
func ValidateVPNConfig(cfg *config.VPNConfig) error {
	if cfg == nil {
		return nil
	}

	if cfg.Remote == "" {
		return fmt.Errorf("remote server address is required")
	}

	if cfg.Port < 1 || cfg.Port > 65535 {
		return fmt.Errorf("invalid VPN port: %d", cfg.Port)
	}

	if cfg.Protocol != "tcp" && cfg.Protocol != "udp" {
		return fmt.Errorf("invalid VPN protocol: %s", cfg.Protocol)
	}

	// Either config file or CA certificate is required
	if cfg.Config == "" && cfg.CA == "" {
		return fmt.Errorf("either OpenVPN config file or CA certificate is required")
	}

	// If auth is provided, both username and password are required
	if len(cfg.Auth) > 0 {
		if cfg.Auth["username"] == "" || cfg.Auth["password"] == "" {
			return fmt.Errorf("both username and password are required for authentication")
		}
	}

	// If certificate is provided, key must also be provided
	if cfg.Cert != "" && cfg.Key == "" {
		return fmt.Errorf("client key is required when client certificate is provided")
	}

	return nil
}
