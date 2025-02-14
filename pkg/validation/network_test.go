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

func TestValidateNetworkInterfaceName(t *testing.T) {
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
			err := ValidateNetworkInterfaceName(tt.iface)
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
		config      *NetworkConfig
		wantErr     bool
		errContains string
	}{
		{
			name: "valid bridge config with static IP",
			config: &NetworkConfig{
				Type:      "bridge",
				Bridge:    "br0",
				Interface: "eth0",
				IP:        "192.168.1.100/24",
				Gateway:   "192.168.1.1",
				DNS:       []string{"8.8.8.8", "8.8.4.4"},
				DHCP:      false,
				Hostname:  "host1",
				MTU:       1500,
				MAC:       "00:11:22:33:44:55",
			},
			wantErr: false,
		},
		{
			name: "valid bridge config with DHCP",
			config: &NetworkConfig{
				Type:      "bridge",
				Bridge:    "br0",
				Interface: "eth0",
				DHCP:      true,
				Hostname:  "host1",
				MTU:       1500,
				MAC:       "00:11:22:33:44:55",
			},
			wantErr: false,
		},
		{
			name: "DHCP with static IP",
			config: &NetworkConfig{
				Type:      "bridge",
				Bridge:    "br0",
				Interface: "eth0",
				IP:        "192.168.1.100/24",
				DHCP:      true,
			},
			wantErr:     true,
			errContains: "cannot specify static IP when DHCP is enabled",
		},
		{
			name: "DHCP with gateway",
			config: &NetworkConfig{
				Type:      "bridge",
				Bridge:    "br0",
				Interface: "eth0",
				Gateway:   "192.168.1.1",
				DHCP:      true,
			},
			wantErr:     true,
			errContains: "cannot specify gateway when DHCP is enabled",
		},
		{
			name: "invalid hostname",
			config: &NetworkConfig{
				Type:      "bridge",
				Bridge:    "br0",
				Interface: "eth0",
				DHCP:      true,
				Hostname:  "-invalid",
			},
			wantErr:     true,
			errContains: "must start and end with alphanumeric",
		},
		{
			name: "invalid MTU",
			config: &NetworkConfig{
				Type:      "bridge",
				Bridge:    "br0",
				Interface: "eth0",
				DHCP:      true,
				MTU:       50,
			},
			wantErr:     true,
			errContains: "must be between 68 and 65535",
		},
		{
			name: "invalid MAC",
			config: &NetworkConfig{
				Type:      "bridge",
				Bridge:    "br0",
				Interface: "eth0",
				DHCP:      true,
				MAC:       "invalid",
			},
			wantErr:     true,
			errContains: "invalid MAC address",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateNetworkConfig(tt.config)
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

func TestValidatePortNumber(t *testing.T) {
	tests := []struct {
		name    string
		port    int
		wantErr bool
	}{
		{
			name:    "valid port",
			port:    8080,
			wantErr: false,
		},
		{
			name:    "port zero",
			port:    0,
			wantErr: true,
		},
		{
			name:    "negative port",
			port:    -1,
			wantErr: true,
		},
		{
			name:    "port too high",
			port:    65536,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePortNumber(tt.port)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidatePortNumber() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidatePortForward(t *testing.T) {
	tests := []struct {
		name      string
		forward   PortForward
		wantErr   bool
		errString string
	}{
		{
			name: "valid tcp forward",
			forward: PortForward{
				Protocol: "tcp",
				Host:     8080,
				Guest:    80,
			},
			wantErr: false,
		},
		{
			name: "valid udp forward",
			forward: PortForward{
				Protocol: "UDP",
				Host:     53,
				Guest:    53,
			},
			wantErr: false,
		},
		{
			name: "invalid protocol",
			forward: PortForward{
				Protocol: "invalid",
				Host:     8080,
				Guest:    80,
			},
			wantErr:   true,
			errString: "protocol must be either tcp or udp",
		},
		{
			name: "invalid host port",
			forward: PortForward{
				Protocol: "tcp",
				Host:     0,
				Guest:    80,
			},
			wantErr:   true,
			errString: "invalid host port",
		},
		{
			name: "invalid guest port",
			forward: PortForward{
				Protocol: "tcp",
				Host:     8080,
				Guest:    0,
			},
			wantErr:   true,
			errString: "invalid guest port",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePortForward(&tt.forward)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidatePortForward() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil && tt.errString != "" && !contains(err.Error(), tt.errString) {
				t.Errorf("ValidatePortForward() error = %v, want error containing %q", err, tt.errString)
			}
		})
	}
}

func TestValidateNetworkInterface(t *testing.T) {
	tests := []struct {
		name      string
		iface     NetworkInterface
		wantErr   bool
		errString string
	}{
		{
			name: "valid bridge interface",
			iface: NetworkInterface{
				Type:      "bridge",
				Bridge:    "br0",
				Interface: "eth0",
				IP:        "192.168.1.100/24",
				Gateway:   "192.168.1.1",
				DNS:       []string{"8.8.8.8"},
				Hostname:  "test-host",
				MTU:       1500,
				MAC:       "00:11:22:33:44:55",
			},
			wantErr: false,
		},
		{
			name: "valid dhcp interface",
			iface: NetworkInterface{
				Type:      "bridge",
				Bridge:    "br0",
				Interface: "eth0",
				DHCP:      true,
			},
			wantErr: false,
		},
		{
			name: "missing bridge name",
			iface: NetworkInterface{
				Type:      "bridge",
				Interface: "eth0",
			},
			wantErr:   true,
			errString: "bridge name is required",
		},
		{
			name: "dhcp with static ip",
			iface: NetworkInterface{
				Type:   "bridge",
				Bridge: "br0",
				DHCP:   true,
				IP:     "192.168.1.100/24",
			},
			wantErr:   true,
			errString: "cannot specify static IP when DHCP is enabled",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateNetworkInterface(&tt.iface)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateNetworkInterface() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil && tt.errString != "" && !contains(err.Error(), tt.errString) {
				t.Errorf("ValidateNetworkInterface() error = %v, want error containing %q", err, tt.errString)
			}
		})
	}
}

func TestValidateSearchDomains(t *testing.T) {
	tests := []struct {
		name    string
		domains []string
		wantErr bool
	}{
		{
			name:    "valid domains",
			domains: []string{"example.com", "test.local"},
			wantErr: false,
		},
		{
			name:    "empty list",
			domains: []string{},
			wantErr: false,
		},
		{
			name:    "invalid domain",
			domains: []string{"example.com", "-invalid.com"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateSearchDomains(tt.domains)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateSearchDomains() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
