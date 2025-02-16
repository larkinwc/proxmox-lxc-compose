package common

import (
	"fmt"
	"net"
	"regexp"
	"strconv"
	"strings"
)

// VPNConfig represents OpenVPN configuration
type VPNConfig struct {
	Remote   string            `yaml:"remote" json:"remote"`                     // VPN server address
	Port     int               `yaml:"port" json:"port"`                         // VPN server port
	Protocol string            `yaml:"protocol" json:"protocol"`                 // udp or tcp
	Config   string            `yaml:"config,omitempty" json:"config,omitempty"` // Path to OpenVPN config file
	Auth     map[string]string `yaml:"auth,omitempty" json:"auth,omitempty"`     // Authentication credentials
	CA       string            `yaml:"ca,omitempty" json:"ca,omitempty"`         // CA certificate content
	Cert     string            `yaml:"cert,omitempty" json:"cert,omitempty"`     // Client certificate content
	Key      string            `yaml:"key,omitempty" json:"key,omitempty"`       // Client key content
}

// NetworkConfig represents network configuration for a container
type NetworkConfig struct {
	Type         string             `yaml:"type" json:"type"`
	Bridge       string             `yaml:"bridge,omitempty" json:"bridge,omitempty"`
	Interface    string             `yaml:"interface,omitempty" json:"interface,omitempty"`
	IP           string             `yaml:"ip,omitempty" json:"ip,omitempty"`
	Gateway      string             `yaml:"gateway,omitempty" json:"gateway,omitempty"`
	DNS          []string           `yaml:"dns,omitempty" json:"dns,omitempty"`
	DHCP         bool               `yaml:"dhcp,omitempty" json:"dhcp,omitempty"`
	Hostname     string             `yaml:"hostname,omitempty" json:"hostname,omitempty"`
	MTU          int                `yaml:"mtu,omitempty" json:"mtu,omitempty"`
	MAC          string             `yaml:"mac,omitempty" json:"mac,omitempty"`
	Interfaces   []NetworkInterface `yaml:"interfaces,omitempty" json:"interfaces,omitempty"`
	PortForwards []PortForward      `yaml:"port_forwards,omitempty" json:"port_forwards,omitempty"`
}

// BandwidthLimit defines bandwidth rate limiting configuration
type BandwidthLimit struct {
	IngressRate  string `yaml:"ingress_rate,omitempty" json:"ingress_rate,omitempty"`
	IngressBurst string `yaml:"ingress_burst,omitempty" json:"ingress_burst,omitempty"`
	EgressRate   string `yaml:"egress_rate,omitempty" json:"egress_rate,omitempty"`
	EgressBurst  string `yaml:"egress_burst,omitempty" json:"egress_burst,omitempty"`
}

// NetworkInterface represents a network interface configuration
type NetworkInterface struct {
	Type      string          `yaml:"type" json:"type"`
	Bridge    string          `yaml:"bridge,omitempty" json:"bridge,omitempty"`
	Interface string          `yaml:"interface,omitempty" json:"interface,omitempty"`
	IP        string          `yaml:"ip,omitempty" json:"ip,omitempty"`
	Gateway   string          `yaml:"gateway,omitempty" json:"gateway,omitempty"`
	DNS       []string        `yaml:"dns,omitempty" json:"dns,omitempty"`
	DHCP      bool            `yaml:"dhcp,omitempty" json:"dhcp,omitempty"`
	Hostname  string          `yaml:"hostname,omitempty" json:"hostname,omitempty"`
	MTU       int             `yaml:"mtu,omitempty" json:"mtu,omitempty"`
	MAC       string          `yaml:"mac,omitempty" json:"mac,omitempty"`
	Bandwidth *BandwidthLimit `yaml:"bandwidth,omitempty" json:"bandwidth,omitempty"`
}

// PortForward represents a port forwarding configuration
type PortForward struct {
	Protocol string `yaml:"protocol" json:"protocol"`
	Host     int    `yaml:"host" json:"host"`
	Guest    int    `yaml:"guest" json:"guest"`
}

// CPUConfig represents CPU resource limits
type CPUConfig struct {
	Shares *int64 `yaml:"shares,omitempty" json:"shares,omitempty"`
	Quota  *int64 `yaml:"quota,omitempty" json:"quota,omitempty"`
	Period *int64 `yaml:"period,omitempty" json:"period,omitempty"`
	Cores  *int   `yaml:"cores,omitempty" json:"cores,omitempty"`
}

// MemoryConfig represents memory resource limits
type MemoryConfig struct {
	Limit string `yaml:"limit,omitempty" json:"limit,omitempty"`
	Swap  string `yaml:"swap,omitempty" json:"swap,omitempty"`
}

// StorageConfig represents storage configuration
type StorageConfig struct {
	Root      string  `yaml:"root,omitempty" json:"root,omitempty"`
	Backend   string  `yaml:"backend,omitempty" json:"backend,omitempty"`
	Pool      string  `yaml:"pool,omitempty" json:"pool,omitempty"`
	AutoMount bool    `yaml:"auto_mount,omitempty" json:"auto_mount,omitempty"`
	Mounts    []Mount `yaml:"mounts,omitempty" json:"mounts,omitempty"`
}

// Mount represents a mount point configuration
type Mount struct {
	Source  string   `yaml:"source" json:"source"`
	Target  string   `yaml:"target" json:"target"`
	Type    string   `yaml:"type" json:"type"`
	Options []string `yaml:"options,omitempty" json:"options,omitempty"`
}

// SecurityConfig represents security settings
type SecurityConfig struct {
	Isolation       string   `yaml:"isolation" json:"isolation"`
	Privileged      bool     `yaml:"privileged,omitempty" json:"privileged,omitempty"`
	AppArmorProfile string   `yaml:"apparmor_profile,omitempty" json:"apparmor_profile,omitempty"`
	SeccompProfile  string   `yaml:"seccomp_profile,omitempty" json:"seccomp_profile,omitempty"`
	SELinuxContext  string   `yaml:"selinux_context,omitempty" json:"selinux_context,omitempty"`
	Capabilities    []string `yaml:"capabilities,omitempty" json:"capabilities,omitempty"`
}

// DeviceConfig represents a device configuration
type DeviceConfig struct {
	Name        string   `yaml:"name" json:"name"`
	Type        string   `yaml:"type" json:"type"`
	Source      string   `yaml:"source" json:"source"`
	Destination string   `yaml:"destination,omitempty" json:"destination,omitempty"`
	Options     []string `yaml:"options,omitempty" json:"options,omitempty"`
}

// Container represents a container configuration
type Container struct {
	Image       string            `yaml:"image" json:"image"`
	Network     *NetworkConfig    `yaml:"network,omitempty" json:"network,omitempty"`
	Storage     *StorageConfig    `yaml:"storage,omitempty" json:"storage,omitempty"`
	Security    *SecurityConfig   `yaml:"security,omitempty" json:"security,omitempty"`
	CPU         *CPUConfig        `yaml:"cpu,omitempty" json:"cpu,omitempty"`
	Memory      *MemoryConfig     `yaml:"memory,omitempty" json:"memory,omitempty"`
	Ports       []PortForward     `yaml:"ports,omitempty" json:"ports,omitempty"`
	Volumes     []string          `yaml:"volumes,omitempty" json:"volumes,omitempty"`
	Env         map[string]string `yaml:"env,omitempty" json:"env,omitempty"`
	Command     []string          `yaml:"command,omitempty" json:"command,omitempty"`
	Entrypoint  []string          `yaml:"entrypoint,omitempty" json:"entrypoint,omitempty"`
	Environment map[string]string `yaml:"environment,omitempty" json:"environment,omitempty"`
	Devices     []DeviceConfig    `yaml:"devices,omitempty" json:"devices,omitempty"`
}

// ComposeConfig represents a docker-compose like configuration
type ComposeConfig struct {
	Services map[string]Container `yaml:"services" json:"services"`
}

// Load loads the configuration from a file
func Load(configFile string) (*ComposeConfig, error) {
	// Implement the function to load the configuration from a file
	return nil, nil
}

// ValidateNetworkConfig validates the network configuration
func ValidateNetworkConfig(cfg *NetworkConfig) error {
	if cfg == nil {
		return nil
	}

	// Handle legacy configuration
	if cfg.Type != "" {
		if cfg.Type != "bridge" && cfg.Type != "veth" {
			return fmt.Errorf("invalid network type: %s", cfg.Type)
		}
	}

	// Validate interfaces
	for i, iface := range cfg.Interfaces {
		if iface.Type == "" {
			iface.Type = "veth" // Default to veth if not specified
		}
		if iface.Type != "bridge" && iface.Type != "veth" {
			return fmt.Errorf("interface %d: invalid network type: %s", i, iface.Type)
		}

		if iface.Type == "bridge" && iface.Bridge == "" {
			return fmt.Errorf("interface %d: bridge name is required for bridge network type", i)
		}

		if iface.IP != "" {
			if err := validateIPAddress(iface.IP); err != nil {
				return fmt.Errorf("interface %d: invalid IP address: %w", i, err)
			}
		}

		if iface.Gateway != "" {
			if err := validateIPAddress(iface.Gateway); err != nil {
				return fmt.Errorf("interface %d: invalid gateway: %w", i, err)
			}
		}

		if iface.MTU != 0 && (iface.MTU < 68 || iface.MTU > 65535) {
			return fmt.Errorf("interface %d: MTU must be between 68 and 65535", i)
		}

		if iface.MAC != "" {
			if err := validateMACAddress(iface.MAC); err != nil {
				return fmt.Errorf("interface %d: invalid MAC address: %w", i, err)
			}
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

func validateIPAddress(ip string) error {
	if ip == "" {
		return nil
	}

	ipPart, cidr, ok := strings.Cut(ip, "/")
	if !ok {
		ipPart = ip
	}

	ipAddr := net.ParseIP(ipPart)
	if ipAddr == nil {
		return fmt.Errorf("invalid IP address format: %s", ip)
	}

	if cidr != "" {
		prefix, err := strconv.Atoi(cidr)
		if err != nil {
			return fmt.Errorf("invalid network prefix: %s", cidr)
		}

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

func validateMACAddress(mac string) error {
	if mac == "" {
		return nil
	}

	macPattern := `^([0-9A-Fa-f]{2}[:-]){5}([0-9A-Fa-f]{2})$`
	if !regexp.MustCompile(macPattern).MatchString(mac) {
		return fmt.Errorf("invalid MAC address format")
	}

	return nil
}
