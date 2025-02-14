package validation

import (
	"fmt"
	"strings"
)

// NetworkInterface represents a network interface configuration
type NetworkInterface struct {
	Type      string   `json:"type"`
	Bridge    string   `json:"bridge,omitempty"`
	Interface string   `json:"interface,omitempty"`
	IP        string   `json:"ip,omitempty"`
	Gateway   string   `json:"gateway,omitempty"`
	DNS       []string `json:"dns,omitempty"`
	DHCP      bool     `json:"dhcp,omitempty"`
	Hostname  string   `json:"hostname,omitempty"`
	MTU       int      `json:"mtu,omitempty"`
	MAC       string   `json:"mac,omitempty"`
}

// NetworkConfig represents network configuration for a container
type NetworkConfig struct {
	Interfaces    []NetworkInterface `json:"interfaces"`
	PortForwards  []PortForward      `json:"port_forwards,omitempty"`
	DNSServers    []string           `json:"dns_servers,omitempty"`
	SearchDomains []string           `json:"search_domains,omitempty"`
	Isolated      bool               `json:"isolated,omitempty"`

	// Legacy fields for backward compatibility
	Type      string   `json:"type,omitempty"`
	Bridge    string   `json:"bridge,omitempty"`
	Interface string   `json:"interface,omitempty"`
	IP        string   `json:"ip,omitempty"`
	Gateway   string   `json:"gateway,omitempty"`
	DNS       []string `json:"dns,omitempty"`
	DHCP      bool     `json:"dhcp,omitempty"`
	Hostname  string   `json:"hostname,omitempty"`
	MTU       int      `json:"mtu,omitempty"`
	MAC       string   `json:"mac,omitempty"`
}

// PortForward represents a port forwarding configuration
type PortForward struct {
	Protocol string `json:"protocol"` // tcp or udp
	Host     int    `json:"host"`     // host port
	Guest    int    `json:"guest"`    // container port
}

// SecurityProfile represents container security settings
type SecurityProfile struct {
	Isolation    string   `json:"isolation,omitempty"`
	Privileged   bool     `json:"privileged,omitempty"`
	Capabilities []string `json:"capabilities,omitempty"`
}

// ValidateSecurityProfile validates security configuration
func ValidateSecurityProfile(cfg *SecurityProfile) error {
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
func isValidCapability(capability string) bool {
	return validCaps[capability]
}

var validCaps = map[string]bool{
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
