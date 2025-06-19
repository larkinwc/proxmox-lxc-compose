package validation

import (
	"strings"
	"testing"

	"github.com/larkinwc/proxmox-lxc-compose/pkg/config"
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
			errContains: "invalid network type",
		},
		{
			name:        "invalid type",
			networkType: "invalid",
			wantErr:     true,
			errContains: "invalid network type",
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
			name:        "phys not supported",
			networkType: "phys",
			wantErr:     true,
			errContains: "invalid network type",
		},
		{
			name:        "uppercase not supported",
			networkType: "BRIDGE",
			wantErr:     true,
			errContains: "invalid network type",
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
			errContains: "invalid IP address",
		},
		{
			name:        "invalid CIDR - too high IPv4",
			ip:          "192.168.1.1/33",
			wantErr:     true,
			errContains: "invalid IP address",
		},
		{
			name:        "invalid CIDR - too high IPv6",
			ip:          "2001:db8::1/129",
			wantErr:     true,
			errContains: "invalid IP address",
		},
		{
			name:        "invalid CIDR format",
			ip:          "192.168.1.1/abc",
			wantErr:     true,
			errContains: "invalid IP address",
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
			errContains: "invalid DNS server address",
		},
		{
			name:        "empty server in list",
			servers:     []string{"8.8.8.8", ""},
			wantErr:     true,
			errContains: "invalid DNS server address",
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
		iface       *config.NetworkInterface
		wantErr     bool
		errContains string
	}{
		{
			name: "valid bridge interface",
			iface: &config.NetworkInterface{
				Type:      "bridge",
				Bridge:    "br0",
				Interface: "eth0",
			},
			wantErr: false,
		},
		{
			name: "valid DHCP interface",
			iface: &config.NetworkInterface{
				Type:      "bridge",
				Bridge:    "br0",
				Interface: "eth0",
				DHCP:      true,
			},
			wantErr: false,
		},
		{
			name: "valid static IP interface",
			iface: &config.NetworkInterface{
				Type:      "bridge",
				Bridge:    "br0",
				Interface: "eth0",
				IP:        "192.168.1.100/24",
				Gateway:   "192.168.1.1",
			},
			wantErr: false,
		},
		{
			name: "bridge without name allowed in ValidateNetworkInterface",
			iface: &config.NetworkInterface{
				Type:      "bridge",
				Interface: "eth0",
			},
			wantErr: false,
		},
		{
			name: "unsupported type",
			iface: &config.NetworkInterface{
				Type: "invalid",
			},
			wantErr:     true,
			errContains: "invalid network type",
		},
		{
			name: "invalid IP",
			iface: &config.NetworkInterface{
				Type:      "bridge",
				Bridge:    "br0",
				Interface: "eth0",
				IP:        "invalid",
			},
			wantErr:     true,
			errContains: "invalid IP address",
		},
		{
			name: "invalid gateway",
			iface: &config.NetworkInterface{
				Type:      "bridge",
				Bridge:    "br0",
				Interface: "eth0",
				Gateway:   "invalid",
			},
			wantErr:     true,
			errContains: "invalid IP address",
		},
		{
			name: "interface name not validated by ValidateNetworkInterface",
			iface: &config.NetworkInterface{
				Type:      "bridge",
				Bridge:    "br0",
				Interface: "invalid@name",
			},
			wantErr: false,
		},
		{
			name: "MTU not validated by ValidateNetworkInterface",
			iface: &config.NetworkInterface{
				Type:      "bridge",
				Bridge:    "br0",
				Interface: "eth0",
				MTU:       100, // Would be too low if validated
			},
			wantErr: false,
		},
		{
			name: "invalid MAC",
			iface: &config.NetworkInterface{
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
		cfg         *config.NetworkConfig
		wantErr     bool
		errContains string
	}{
		{
			name: "valid config with single interface",
			cfg: &config.NetworkConfig{
				Interfaces: []config.NetworkInterface{
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
			cfg: &config.NetworkConfig{
				Interfaces: []config.NetworkInterface{
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
			name: "empty interfaces list is allowed",
			cfg: &config.NetworkConfig{
				Interfaces: []config.NetworkInterface{},
			},
			wantErr: false,
		},
		{
			name: "invalid interface",
			cfg: &config.NetworkConfig{
				Interfaces: []config.NetworkInterface{
					{
						Type: "invalid",
					},
				},
			},
			wantErr:     true,
			errContains: "invalid network type",
		},
		{
			name: "bridge without name in config",
			cfg: &config.NetworkConfig{
				Interfaces: []config.NetworkInterface{
					{
						Type: "bridge",
					},
				},
			},
			wantErr:     true,
			errContains: "bridge name is required",
		},
		{
			name: "invalid MTU in config",
			cfg: &config.NetworkConfig{
				Interfaces: []config.NetworkInterface{
					{
						Type:   "bridge",
						Bridge: "br0",
						MTU:    50, // Too low
					},
				},
			},
			wantErr:     true,
			errContains: "MTU must be between",
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
		cfg         *config.VPNConfig
		wantErr     bool
		errContains string
	}{
		{
			name: "valid config with CA",
			cfg: &config.VPNConfig{
				Remote:   "vpn.example.com",
				Port:     1194,
				Protocol: "udp",
				CA:       "ca content",
			},
			wantErr: false,
		},
		{
			name: "valid config with file",
			cfg: &config.VPNConfig{
				Remote:   "vpn.example.com",
				Port:     1194,
				Protocol: "tcp",
				Config:   "config content",
			},
			wantErr: false,
		},
		{
			name: "missing remote",
			cfg: &config.VPNConfig{
				Port:     1194,
				Protocol: "udp",
				CA:       "ca content",
			},
			wantErr:     true,
			errContains: "remote server address is required",
		},
		{
			name: "invalid port",
			cfg: &config.VPNConfig{
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
			cfg: &config.VPNConfig{
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
			cfg: &config.VPNConfig{
				Remote:   "vpn.example.com",
				Port:     1194,
				Protocol: "udp",
			},
			wantErr:     true,
			errContains: "either OpenVPN config file or CA certificate is required",
		},
		{
			name: "incomplete auth",
			cfg: &config.VPNConfig{
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
			cfg: &config.VPNConfig{
				Remote:   "vpn.example.com",
				Port:     1194,
				Protocol: "udp",
				CA:       "ca content",
				Cert:     "cert content",
			},
			wantErr:     true,
			errContains: "client key is required when client certificate is provided",
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
