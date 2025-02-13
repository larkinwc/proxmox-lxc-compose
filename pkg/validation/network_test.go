package validation

import "testing"

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
			name:        "valid type - uppercase",
			networkType: "BRIDGE",
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateNetworkType(tt.networkType)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				} else if tt.errContains != "" && !contains(err.Error(), tt.errContains) {
					t.Errorf("error %q does not contain %q", err.Error(), tt.errContains)
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
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
			name:    "empty ip",
			ip:      "",
			wantErr: false,
		},
		{
			name:    "valid ipv4",
			ip:      "192.168.1.1",
			wantErr: false,
		},
		{
			name:    "valid ipv6",
			ip:      "2001:db8::1",
			wantErr: false,
		},
		{
			name:    "valid ipv4 with cidr",
			ip:      "192.168.1.1/24",
			wantErr: false,
		},
		{
			name:    "valid ipv6 with cidr",
			ip:      "2001:db8::1/64",
			wantErr: false,
		},
		{
			name:        "invalid ip",
			ip:          "256.256.256.256",
			wantErr:     true,
			errContains: "invalid IP address",
		},
		{
			name:        "invalid cidr",
			ip:          "192.168.1.1/33",
			wantErr:     true,
			errContains: "must be between 1 and 32",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateIPAddress(tt.ip)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				} else if tt.errContains != "" && !contains(err.Error(), tt.errContains) {
					t.Errorf("error %q does not contain %q", err.Error(), tt.errContains)
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
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
			servers: nil,
			wantErr: false,
		},
		{
			name:    "valid servers",
			servers: []string{"8.8.8.8", "8.8.4.4"},
			wantErr: false,
		},
		{
			name:    "valid ipv6 servers",
			servers: []string{"2001:4860:4860::8888", "2001:4860:4860::8844"},
			wantErr: false,
		},
		{
			name:        "invalid server",
			servers:     []string{"8.8.8.8", "invalid"},
			wantErr:     true,
			errContains: "invalid DNS server",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateDNSServers(tt.servers)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				} else if tt.errContains != "" && !contains(err.Error(), tt.errContains) {
					t.Errorf("error %q does not contain %q", err.Error(), tt.errContains)
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestValidateNetworkInterface(t *testing.T) {
	tests := []struct {
		name        string
		iface       string
		wantErr     bool
		errContains string
	}{
		{
			name:    "empty interface",
			iface:   "",
			wantErr: false,
		},
		{
			name:    "valid interface",
			iface:   "eth0",
			wantErr: false,
		},
		{
			name:    "valid interface with numbers",
			iface:   "eth123",
			wantErr: false,
		},
		{
			name:    "valid interface with hyphen",
			iface:   "eth-wan",
			wantErr: false,
		},
		{
			name:    "valid interface with underscore",
			iface:   "eth_lan",
			wantErr: false,
		},
		{
			name:        "too long interface name",
			iface:       "very-long-interface-name",
			wantErr:     true,
			errContains: "too long",
		},
		{
			name:        "invalid characters",
			iface:       "eth@0",
			wantErr:     true,
			errContains: "invalid interface name",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateNetworkInterface(tt.iface)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				} else if tt.errContains != "" && !contains(err.Error(), tt.errContains) {
					t.Errorf("error %q does not contain %q", err.Error(), tt.errContains)
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestValidateHostname(t *testing.T) {
	tests := []struct {
		name        string
		hostname    string
		wantErr     bool
		errContains string
	}{
		{
			name:     "empty hostname",
			hostname: "",
			wantErr:  false,
		},
		{
			name:     "valid hostname",
			hostname: "host1",
			wantErr:  false,
		},
		{
			name:     "valid hostname with hyphen",
			hostname: "web-server",
			wantErr:  false,
		},
		{
			name:        "hostname starts with hyphen",
			hostname:    "-host",
			wantErr:     true,
			errContains: "must start and end with alphanumeric",
		},
		{
			name:        "hostname ends with hyphen",
			hostname:    "host-",
			wantErr:     true,
			errContains: "must start and end with alphanumeric",
		},
		{
			name:        "hostname with invalid characters",
			hostname:    "host_name",
			wantErr:     true,
			errContains: "must start and end with alphanumeric",
		},
		{
			name:        "hostname too long",
			hostname:    "a123456789012345678901234567890123456789012345678901234567890123",
			wantErr:     true,
			errContains: "too long",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateHostname(tt.hostname)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				} else if tt.errContains != "" && !contains(err.Error(), tt.errContains) {
					t.Errorf("error %q does not contain %q", err.Error(), tt.errContains)
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestValidateMTU(t *testing.T) {
	tests := []struct {
		name        string
		mtu         int
		wantErr     bool
		errContains string
	}{
		{
			name:    "zero mtu (default)",
			mtu:     0,
			wantErr: false,
		},
		{
			name:    "minimum valid mtu",
			mtu:     68,
			wantErr: false,
		},
		{
			name:    "maximum valid mtu",
			mtu:     65535,
			wantErr: false,
		},
		{
			name:        "mtu too small",
			mtu:         67,
			wantErr:     true,
			errContains: "must be between 68 and 65535",
		},
		{
			name:        "mtu too large",
			mtu:         65536,
			wantErr:     true,
			errContains: "must be between 68 and 65535",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateMTU(tt.mtu)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				} else if tt.errContains != "" && !contains(err.Error(), tt.errContains) {
					t.Errorf("error %q does not contain %q", err.Error(), tt.errContains)
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestValidateMAC(t *testing.T) {
	tests := []struct {
		name        string
		mac         string
		wantErr     bool
		errContains string
	}{
		{
			name:    "empty mac",
			mac:     "",
			wantErr: false,
		},
		{
			name:    "valid mac",
			mac:     "00:11:22:33:44:55",
			wantErr: false,
		},
		{
			name:    "valid mac with different separator",
			mac:     "00-11-22-33-44-55",
			wantErr: false,
		},
		{
			name:        "invalid mac format",
			mac:         "00:11:22:33:44",
			wantErr:     true,
			errContains: "invalid MAC address",
		},
		{
			name:        "invalid mac characters",
			mac:         "00:11:22:33:44:ZZ",
			wantErr:     true,
			errContains: "invalid MAC address",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateMAC(tt.mac)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				} else if tt.errContains != "" && !contains(err.Error(), tt.errContains) {
					t.Errorf("error %q does not contain %q", err.Error(), tt.errContains)
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestValidateNetworkConfig(t *testing.T) {
	tests := []struct {
		name        string
		networkType string
		bridge      string
		iface       string
		ip          string
		gateway     string
		dns         []string
		dhcp        bool
		hostname    string
		mtu         int
		mac         string
		wantErr     bool
		errContains string
	}{
		{
			name:        "valid bridge config with static IP",
			networkType: "bridge",
			bridge:      "br0",
			iface:       "eth0",
			ip:          "192.168.1.100/24",
			gateway:     "192.168.1.1",
			dns:         []string{"8.8.8.8", "8.8.4.4"},
			dhcp:        false,
			hostname:    "host1",
			mtu:         1500,
			mac:         "00:11:22:33:44:55",
			wantErr:     false,
		},
		{
			name:        "valid bridge config with DHCP",
			networkType: "bridge",
			bridge:      "br0",
			iface:       "eth0",
			dhcp:        true,
			hostname:    "host1",
			mtu:         1500,
			mac:         "00:11:22:33:44:55",
			wantErr:     false,
		},
		{
			name:        "DHCP with static IP",
			networkType: "bridge",
			bridge:      "br0",
			iface:       "eth0",
			ip:          "192.168.1.100/24",
			dhcp:        true,
			wantErr:     true,
			errContains: "cannot specify static IP when DHCP is enabled",
		},
		{
			name:        "DHCP with gateway",
			networkType: "bridge",
			bridge:      "br0",
			iface:       "eth0",
			gateway:     "192.168.1.1",
			dhcp:        true,
			wantErr:     true,
			errContains: "cannot specify gateway when DHCP is enabled",
		},
		{
			name:        "invalid hostname",
			networkType: "bridge",
			bridge:      "br0",
			iface:       "eth0",
			dhcp:        true,
			hostname:    "-invalid",
			wantErr:     true,
			errContains: "must start and end with alphanumeric",
		},
		{
			name:        "invalid MTU",
			networkType: "bridge",
			bridge:      "br0",
			iface:       "eth0",
			dhcp:        true,
			mtu:         50,
			wantErr:     true,
			errContains: "must be between 68 and 65535",
		},
		{
			name:        "invalid MAC",
			networkType: "bridge",
			bridge:      "br0",
			iface:       "eth0",
			dhcp:        true,
			mac:         "invalid",
			wantErr:     true,
			errContains: "invalid MAC address",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateNetworkConfig(
				tt.networkType,
				tt.bridge,
				tt.iface,
				tt.ip,
				tt.gateway,
				tt.dns,
				tt.dhcp,
				tt.hostname,
				tt.mtu,
				tt.mac,
			)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				} else if tt.errContains != "" && !contains(err.Error(), tt.errContains) {
					t.Errorf("error %q does not contain %q", err.Error(), tt.errContains)
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}
