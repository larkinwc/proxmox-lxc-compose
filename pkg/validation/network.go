package validation

import (
	"fmt"
	"net"
	"regexp"
	"strconv"
	"strings"
)

// Supported network types
var supportedNetworkTypes = map[string]bool{
	"none":    true,
	"veth":    true,
	"bridge":  true,
	"macvlan": true,
	"phys":    true,
}

// ValidateNetworkType validates the network type
func ValidateNetworkType(networkType string) error {
	if networkType == "" {
		return fmt.Errorf("network type is required")
	}

	if !supportedNetworkTypes[strings.ToLower(networkType)] {
		return fmt.Errorf("unsupported network type: %s (supported types: none, veth, bridge, macvlan, phys)", networkType)
	}

	return nil
}

// ValidateIPAddress validates an IPv4 or IPv6 address with optional CIDR notation
func ValidateIPAddress(ip string) error {
	if ip == "" {
		return nil // IP is optional
	}

	// Split IP and CIDR if present
	ipPart := ip
	var cidr string
	if idx := strings.Index(ip, "/"); idx != -1 {
		ipPart = ip[:idx]
		cidr = ip[idx+1:]
	}

	// Validate IP part
	ipAddr := net.ParseIP(ipPart)
	if ipAddr == nil {
		return fmt.Errorf("invalid IP address format: %s", ip)
	}

	// Validate CIDR if present
	if cidr != "" {
		prefix, err := strconv.Atoi(cidr)
		if err != nil {
			return fmt.Errorf("invalid network prefix: %s", cidr)
		}

		// Check if IPv4 or IPv6
		if ipAddr.To4() != nil {
			if prefix < 1 || prefix > 32 {
				return fmt.Errorf("invalid IPv4 network prefix length: /%d (must be between 1 and 32)", prefix)
			}
		} else {
			if prefix < 1 || prefix > 128 {
				return fmt.Errorf("invalid IPv6 network prefix length: /%d (must be between 1 and 128)", prefix)
			}
		}
	}

	return nil
}

// ValidateDNSServers validates a list of DNS server IP addresses
func ValidateDNSServers(servers []string) error {
	if len(servers) == 0 {
		return nil // DNS servers are optional
	}

	for _, server := range servers {
		ip := net.ParseIP(server)
		if ip == nil {
			return fmt.Errorf("invalid DNS server IP address: %s", server)
		}
	}

	return nil
}

// ValidateNetworkInterface validates a network interface name
func ValidateNetworkInterface(iface string) error {
	if iface == "" {
		return nil // Interface name is optional
	}

	// Basic interface name validation
	// According to Linux kernel documentation, interface names:
	// - Must not be longer than 15 characters
	// - Should only contain alphanumeric characters, hyphens, and underscores
	if len(iface) > 15 {
		return fmt.Errorf("interface name too long (max 15 characters): %s", iface)
	}

	validName := regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)
	if !validName.MatchString(iface) {
		return fmt.Errorf("invalid interface name (must contain only letters, numbers, hyphens, and underscores): %s", iface)
	}

	return nil
}

// ValidateHostname validates a hostname
func ValidateHostname(hostname string) error {
	if hostname == "" {
		return nil
	}

	if len(hostname) > 63 {
		return fmt.Errorf("hostname too long (max 63 characters)")
	}

	// RFC 1123 hostname validation
	if !regexp.MustCompile(`^[a-zA-Z0-9]([a-zA-Z0-9\-]*[a-zA-Z0-9])?$`).MatchString(hostname) {
		return fmt.Errorf("hostname must start and end with alphanumeric characters and can contain hyphens")
	}

	return nil
}

// ValidateMTU validates the MTU value
func ValidateMTU(mtu int) error {
	if mtu == 0 {
		return nil
	}

	if mtu < 68 || mtu > 65535 {
		return fmt.Errorf("MTU must be between 68 and 65535")
	}

	return nil
}

// ValidateMAC validates a MAC address
func ValidateMAC(mac string) error {
	if mac == "" {
		return nil
	}

	// Support both colon and hyphen separators
	macPattern := regexp.MustCompile(`^([0-9A-Fa-f]{2}[:-]){5}([0-9A-Fa-f]{2})$`)
	if !macPattern.MatchString(mac) {
		return fmt.Errorf("invalid MAC address format")
	}

	return nil
}

// ValidateNetworkConfig validates the complete network configuration
func ValidateNetworkConfig(networkType, bridge, iface, ip, gateway string, dns []string, dhcp bool, hostname string, mtu int, mac string) error {
	if err := ValidateNetworkType(networkType); err != nil {
		return err
	}

	if networkType == "bridge" && bridge == "" {
		return fmt.Errorf("bridge name is required for bridge network type")
	}

	if err := ValidateNetworkInterface(iface); err != nil {
		return err
	}

	// Validate DHCP and static IP settings
	if dhcp {
		if ip != "" {
			return fmt.Errorf("cannot specify static IP when DHCP is enabled")
		}
		if gateway != "" {
			return fmt.Errorf("cannot specify gateway when DHCP is enabled")
		}
	} else if ip != "" {
		if err := ValidateIPAddress(ip); err != nil {
			return fmt.Errorf("invalid IP address: %w", err)
		}

		if gateway != "" {
			if err := ValidateIPAddress(gateway); err != nil {
				return fmt.Errorf("invalid gateway: %w", err)
			}
		}
	}

	if err := ValidateDNSServers(dns); err != nil {
		return err
	}

	if err := ValidateHostname(hostname); err != nil {
		return err
	}

	if err := ValidateMTU(mtu); err != nil {
		return err
	}

	if err := ValidateMAC(mac); err != nil {
		return err
	}

	return nil
}
