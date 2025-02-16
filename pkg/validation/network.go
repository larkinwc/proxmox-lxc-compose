package validation

import (
	"fmt"
	"net"
	"proxmox-lxc-compose/pkg/common"
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
	ipPart, cidr, ok := strings.Cut(ip, "/")
	if !ok {
		ipPart = ip
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
		if server == "" {
			return fmt.Errorf("DNS server IP cannot be empty")
		}
		ip := net.ParseIP(server)
		if ip == nil {
			return fmt.Errorf("invalid DNS server IP address: %s", server)
		}
	}

	return nil
}

// ValidateNetworkInterfaceName validates a network interface name
func ValidateNetworkInterfaceName(iface string) error {
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
		return nil // Default MTU
	}

	// Standard minimum MTU values:
	// - IPv4: 576 (RFC 791)
	// - IPv6: 1280 (RFC 8200)
	// - Ethernet: 1500 (most common)
	// Using 576 as absolute minimum for compatibility
	if mtu < 576 || mtu > 65535 {
		return fmt.Errorf("invalid MTU: must be between 576 and 65535")
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

// ValidatePortNumber validates a port number
func ValidatePortNumber(port int) error {
	if port < 1 || port > 65535 {
		return fmt.Errorf("port must be between 1 and 65535")
	}
	return nil
}

// ValidateProtocol validates a protocol type
func ValidateProtocol(protocol string) error {
	protocol = strings.ToLower(protocol)
	if protocol != "tcp" && protocol != "udp" {
		return fmt.Errorf("protocol must be either tcp or udp")
	}
	return nil
}

// ValidatePortForward validates a port forwarding configuration
func ValidatePortForward(pf *PortForward) error {
	if err := ValidateProtocol(pf.Protocol); err != nil {
		return err
	}
	if err := ValidatePortNumber(pf.Host); err != nil {
		return fmt.Errorf("invalid host port: %w", err)
	}
	if err := ValidatePortNumber(pf.Guest); err != nil {
		return fmt.Errorf("invalid guest port: %w", err)
	}
	return nil
}

// ValidateSearchDomains validates DNS search domains
func ValidateSearchDomains(domains []string) error {
	if len(domains) == 0 {
		return nil
	}

	for _, domain := range domains {
		if domain == "" {
			return fmt.Errorf("search domain cannot be empty")
		}

		// Domain name validation according to RFC 1034
		if len(domain) > 253 {
			return fmt.Errorf("invalid search domain %q: domain name too long", domain)
		}

		labels := strings.Split(domain, ".")
		for _, label := range labels {
			if len(label) == 0 || len(label) > 63 {
				return fmt.Errorf("invalid search domain %q: label must be between 1 and 63 characters", domain)
			}

			if !regexp.MustCompile(`^[a-zA-Z0-9]([a-zA-Z0-9-]*[a-zA-Z0-9])?$`).MatchString(label) {
				return fmt.Errorf("invalid search domain %q: label must start and end with alphanumeric characters and can contain hyphens", domain)
			}
		}
	}

	return nil
}

// ValidateNetworkInterface validates a network interface configuration
func ValidateNetworkInterface(iface *NetworkInterface) error {
	if err := ValidateNetworkType(iface.Type); err != nil {
		return err
	}

	if iface.Type == "bridge" && iface.Bridge == "" {
		return fmt.Errorf("bridge name is required for bridge network type")
	}

	if err := ValidateNetworkInterfaceName(iface.Interface); err != nil {
		return err
	}

	// Validate DHCP and static IP settings
	if iface.DHCP {
		if iface.IP != "" {
			return fmt.Errorf("cannot specify static IP when DHCP is enabled")
		}
		if iface.Gateway != "" {
			return fmt.Errorf("cannot specify gateway when DHCP is enabled")
		}
	} else if iface.IP != "" {
		if err := ValidateIPAddress(iface.IP); err != nil {
			return fmt.Errorf("invalid IP address: %w", err)
		}

		if iface.Gateway != "" {
			if err := ValidateIPAddress(iface.Gateway); err != nil {
				return fmt.Errorf("invalid gateway: %w", err)
			}
		}
	}

	if err := ValidateDNSServers(iface.DNS); err != nil {
		return err
	}

	if err := ValidateHostname(iface.Hostname); err != nil {
		return err
	}

	if err := ValidateMTU(iface.MTU); err != nil {
		return err
	}

	if err := ValidateMAC(iface.MAC); err != nil {
		return err
	}

	return nil
}

// ValidateVPNConfig validates the VPN configuration
func ValidateVPNConfig(cfg *common.VPNConfig) error {
	if cfg == nil {
		return nil
	}

	if cfg.Remote == "" {
		return fmt.Errorf("VPN remote server address is required")
	}

	if cfg.Port <= 0 || cfg.Port > 65535 {
		return fmt.Errorf("invalid VPN port: must be between 1 and 65535")
	}

	if cfg.Protocol != "tcp" && cfg.Protocol != "udp" {
		return fmt.Errorf("invalid VPN protocol: must be tcp or udp")
	}

	if cfg.Config == "" && cfg.CA == "" {
		return fmt.Errorf("either OpenVPN config file or CA certificate is required")
	}

	if cfg.Auth != nil {
		if cfg.Auth["username"] == "" || cfg.Auth["password"] == "" {
			return fmt.Errorf("both username and password are required for VPN authentication")
		}
	}

	if (cfg.Cert != "" && cfg.Key == "") || (cfg.Cert == "" && cfg.Key != "") {
		return fmt.Errorf("both certificate and key must be provided together")
	}

	return nil
}

// ValidateNetworkConfig validates the complete network configuration
func ValidateNetworkConfig(cfg *NetworkConfig) error {
	// Support legacy configuration
	if cfg.Type != "" {
		// Convert legacy config to new format
		legacyIface := NetworkInterface{
			Type:      cfg.Type,
			Bridge:    cfg.Bridge,
			Interface: cfg.Interface,
			IP:        cfg.IP,
			Gateway:   cfg.Gateway,
			DNS:       cfg.DNS,
			DHCP:      cfg.DHCP,
			Hostname:  cfg.Hostname,
			MTU:       cfg.MTU,
			MAC:       cfg.MAC,
		}
		if err := ValidateNetworkInterface(&legacyIface); err != nil {
			return err
		}
	}

	// Validate interfaces
	if len(cfg.Interfaces) == 0 && cfg.Type == "" {
		return fmt.Errorf("at least one network interface must be configured")
	}

	for i, iface := range cfg.Interfaces {
		if err := ValidateNetworkInterface(&iface); err != nil {
			return fmt.Errorf("interface %d: %w", i, err)
		}
	}

	// Validate port forwards
	for i, pf := range cfg.PortForwards {
		if err := ValidatePortForward(&pf); err != nil {
			return fmt.Errorf("port forward %d: %w", i, err)
		}
	}

	// Validate DNS configuration
	if err := ValidateDNSServers(cfg.DNSServers); err != nil {
		return fmt.Errorf("DNS servers: %w", err)
	}

	if err := ValidateSearchDomains(cfg.SearchDomains); err != nil {
		return fmt.Errorf("search domains: %w", err)
	}

	return nil
}
