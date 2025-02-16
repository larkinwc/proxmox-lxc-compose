package validation

import (
	"strings"
	"testing"

	"proxmox-lxc-compose/pkg/common"
)

func TestValidateNetworkType(t *testing.T) {
	tests := []struct {
		name        string
		networkType string
		wantErr     bool
		errContains string
	}{
		{
			name:        "empty type",
			networkType: "",
			wantErr:     true,
			errContains: "required",
		},
		{
			name:        "invalid type",
			networkType: "invalid",
			wantErr:     true,
			errContains: "unsupported network type",
		},
		{
			name:        "valid type - none",
			networkType: "none",
			wantErr:     false,
		},
		{
			name:        "valid type - bridge",
			networkType: "bridge",
			wantErr:     false,
		},
		{
			name:        "valid type - veth",
			networkType: "veth",
			wantErr:     false,
		},
		{
			name:        "valid type - macvlan",
			networkType: "macvlan",
			wantErr:     false,
		},
		{
			name:        "valid type - phys",
			networkType: "phys",
			wantErr:     false,
		},
		{
			name:        "valid type - uppercase",
			networkType: "BRIDGE",
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateNetworkType(tt.networkType)
			assertTestError(t, err, tt.wantErr, tt.errContains)
		})
	}
}

func TestValidateIPAddress(t *testing.T) {
	tests := []struct {
		name        string
		ip          string
		wantErr     bool
		errContains string
	}{
		{
			name:    "empty IP",
			ip:      "",
			wantErr: false,
		},
		{
			name:    "valid IPv4",
			ip:      "192.168.1.1",
			wantErr: false,
		},
		{
			name:    "valid IPv4 with CIDR",
			ip:      "192.168.1.1/24",
			wantErr: false,
		},
		{
			name:    "valid IPv6",
			ip:      "2001:db8::1",
			wantErr: false,
		},
		{
			name:    "valid IPv6 with CIDR",
			ip:      "2001:db8::1/64",
			wantErr: false,
		},
		{
			name:        "invalid IP format",
			ip:          "256.256.256.256",
			wantErr:     true,
			errContains: "invalid IP address format",
		},
		{
			name:        "invalid CIDR - too high IPv4",
			ip:          "192.168.1.1/33",
			wantErr:     true,
			errContains: "invalid IPv4 network prefix length",
		},
		{
			name:        "invalid CIDR - too high IPv6",
			ip:          "2001:db8::1/129",
			wantErr:     true,
			errContains: "invalid IPv6 network prefix length",
		},
		{
			name:        "invalid CIDR format",
			ip:          "192.168.1.1/abc",
			wantErr:     true,
			errContains: "invalid network prefix",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateIPAddress(tt.ip)
			assertTestError(t, err, tt.wantErr, tt.errContains)
		})
	}
}

func TestValidateDNSServers(t *testing.T) {
	tests := []struct {
		name        string
		servers     []string
		wantErr     bool
		errContains string
	}{
		{
			name:    "empty list",
			servers: []string{},
			wantErr: false,
		},
		{
			name:    "valid servers",
			servers: []string{"8.8.8.8", "8.8.4.4", "2001:4860:4860::8888"},
			wantErr: false,
		},
		{
			name:        "invalid IP",
			servers:     []string{"8.8.8.8", "invalid"},
			wantErr:     true,
			errContains: "invalid DNS server IP",
		},
		{
			name:        "empty server in list",
			servers:     []string{"8.8.8.8", ""},
			wantErr:     true,
			errContains: "DNS server IP cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateDNSServers(tt.servers)
			assertTestError(t, err, tt.wantErr, tt.errContains)
		})
	}
}

func TestValidateNetworkInterface(t *testing.T) {
	tests := []struct {
		name        string
		iface       *NetworkInterface
		wantErr     bool
		errContains string
	}{
		{
			name: "valid bridge with DHCP",
			iface: &NetworkInterface{
				Type:      "bridge",
				Bridge:    "br0",
				Interface: "eth0",
				DHCP:      true,
			},
			wantErr: false,
		},
		{
			name: "valid bridge with static IP",
			iface: &NetworkInterface{
				Type:      "bridge",
				Bridge:    "br0",
				Interface: "eth0",
				IP:        "192.168.1.100/24",
				Gateway:   "192.168.1.1",
				DNS:       []string{"8.8.8.8"},
			},
			wantErr: false,
		},
		{
			name: "bridge without bridge name",
			iface: &NetworkInterface{
				Type:      "bridge",
				Interface: "eth0",
			},
			wantErr:     true,
			errContains: "bridge name is required",
		},
		{
			name: "DHCP with static IP",
			iface: &NetworkInterface{
				Type:      "bridge",
				Bridge:    "br0",
				Interface: "eth0",
				DHCP:      true,
				IP:        "192.168.1.100/24",
			},
			wantErr:     true,
			errContains: "cannot specify static IP when DHCP is enabled",
		},
		{
			name: "invalid interface name",
			iface: &NetworkInterface{
				Type:      "bridge",
				Bridge:    "br0",
				Interface: "invalid@iface",
			},
			wantErr:     true,
			errContains: "invalid interface name",
		},
		{
			name: "invalid MTU",
			iface: &NetworkInterface{
				Type:      "bridge",
				Bridge:    "br0",
				Interface: "eth0",
				MTU:       100, // Too low
			},
			wantErr:     true,
			errContains: "invalid MTU",
		},
		{
			name: "invalid MAC",
			iface: &NetworkInterface{
				Type:      "bridge",
				Bridge:    "br0",
				Interface: "eth0",
				MAC:       "invalid",
			},
			wantErr:     true,
			errContains: "invalid MAC address",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateNetworkInterface(tt.iface)
			assertTestError(t, err, tt.wantErr, tt.errContains)
		})
	}
}

func TestValidateNetworkConfig(t *testing.T) {
	tests := []struct {
		name        string
		cfg         *NetworkConfig
		wantErr     bool
		errContains string
	}{
		{
			name: "valid config with single interface",
			cfg: &NetworkConfig{
				Interfaces: []NetworkInterface{
					{
						Type:      "bridge",
						Bridge:    "br0",
						Interface: "eth0",
						DHCP:      true,
					},
				},
			},
			wantErr: false,
		},
		{
			name: "valid config with multiple interfaces",
			cfg: &NetworkConfig{
				Interfaces: []NetworkInterface{
					{
						Type:      "bridge",
						Bridge:    "br0",
						Interface: "eth0",
						IP:        "192.168.1.100/24",
						Gateway:   "192.168.1.1",
					},
					{
						Type:      "bridge",
						Bridge:    "br1",
						Interface: "eth1",
						DHCP:      true,
					},
				},
			},
			wantErr: false,
		},
		{
			name: "no interfaces",
			cfg: &NetworkConfig{
				Interfaces: []NetworkInterface{},
			},
			wantErr:     true,
			errContains: "at least one network interface must be configured",
		},
		{
			name: "invalid interface",
			cfg: &NetworkConfig{
				Interfaces: []NetworkInterface{
					{
						Type: "invalid",
					},
				},
			},
			wantErr:     true,
			errContains: "unsupported network type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateNetworkConfig(tt.cfg)
			assertTestError(t, err, tt.wantErr, tt.errContains)
		})
	}
}

func TestValidateVPNConfig(t *testing.T) {
	tests := []struct {
		name        string
		cfg         *common.VPNConfig
		wantErr     bool
		errContains string
	}{
		{
			name: "valid config with CA",
			cfg: &common.VPNConfig{
				Remote:   "vpn.example.com",
				Port:     1194,
				Protocol: "udp",
				CA:       "ca content",
			},
			wantErr: false,
		},
		{
			name: "valid config with file",
			cfg: &common.VPNConfig{
				Remote:   "vpn.example.com",
				Port:     1194,
				Protocol: "tcp",
				Config:   "config content",
			},
			wantErr: false,
		},
		{
			name: "missing remote",
			cfg: &common.VPNConfig{
				Port:     1194,
				Protocol: "udp",
				CA:       "ca content",
			},
			wantErr:     true,
			errContains: "remote server address is required",
		},
		{
			name: "invalid port",
			cfg: &common.VPNConfig{
				Remote:   "vpn.example.com",
				Port:     70000,
				Protocol: "udp",
				CA:       "ca content",
			},
			wantErr:     true,
			errContains: "invalid VPN port",
		},
		{
			name: "invalid protocol",
			cfg: &common.VPNConfig{
				Remote:   "vpn.example.com",
				Port:     1194,
				Protocol: "invalid",
				CA:       "ca content",
			},
			wantErr:     true,
			errContains: "invalid VPN protocol",
		},
		{
			name: "missing CA and config",
			cfg: &common.VPNConfig{
				Remote:   "vpn.example.com",
				Port:     1194,
				Protocol: "udp",
			},
			wantErr:     true,
			errContains: "either OpenVPN config file or CA certificate is required",
		},
		{
			name: "incomplete auth",
			cfg: &common.VPNConfig{
				Remote:   "vpn.example.com",
				Port:     1194,
				Protocol: "udp",
				CA:       "ca content",
				Auth: map[string]string{
					"username": "user",
				},
			},
			wantErr:     true,
			errContains: "both username and password are required",
		},
		{
			name: "cert without key",
			cfg: &common.VPNConfig{
				Remote:   "vpn.example.com",
				Port:     1194,
				Protocol: "udp",
				CA:       "ca content",
				Cert:     "cert content",
			},
			wantErr:     true,
			errContains: "both certificate and key must be provided together",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateVPNConfig(tt.cfg)
			assertTestError(t, err, tt.wantErr, tt.errContains)
		})
	}
}

// Helper function to assert test errors
func assertTestError(t *testing.T, err error, wantErr bool, errContains string) {
	t.Helper()
	if (err != nil) != wantErr {
		t.Errorf("validation error = %v, wantErr %v", err, wantErr)
		return
	}
	if err != nil && errContains != "" {
		if !strings.Contains(err.Error(), errContains) {
			t.Errorf("error %q does not contain %q", err.Error(), errContains)
		}
	}
}
