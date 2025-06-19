package common

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"gopkg.in/yaml.v3"

	"github.com/larkinwc/proxmox-lxc-compose/pkg/config"
	"github.com/larkinwc/proxmox-lxc-compose/pkg/validation"
)

func TestLoad_ValidConfig(t *testing.T) {
	// Create a temporary config file
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "test-config.yml")

	validConfig := `
services:
  web:
    image: "nginx:alpine"
    network:
      type: "bridge"
      bridge: "lxcbr0"
      ip: "192.168.1.100/24"
      gateway: "192.168.1.1"
      dns:
        - "8.8.8.8"
        - "8.8.4.4"
    storage:
      root: "10G"
      backend: "dir"
    security:
      isolation: "default"
      privileged: false
    cpu:
      cores: 2
      shares: 1024
    memory:
      limit: "1G"
      swap: "2G"
    ports:
      - protocol: "tcp"
        host: 8080
        guest: 80
    environment:
      ENV_VAR: "value"
    command: ["/bin/sh", "-c", "nginx"]
    devices:
      - name: "dev1"
        type: "disk"
        source: "/host/path"
        destination: "/container/path"
`

	err := os.WriteFile(configFile, []byte(validConfig), 0644)
	if err != nil {
		t.Fatalf("Failed to write test config file: %v", err)
	}

	// Test loading the config
	cfg, err := config.Load(configFile)
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	if cfg == nil {
		t.Fatal("Load() returned nil config")
	}

	// Verify the loaded configuration
	if len(cfg.Services) != 1 {
		t.Errorf("Expected 1 service, got %d", len(cfg.Services))
	}

	web, exists := cfg.Services["web"]
	if !exists {
		t.Fatal("Expected 'web' service not found")
	}

	if web.Image != "nginx:alpine" {
		t.Errorf("Expected image 'nginx:alpine', got '%s'", web.Image)
	}

	if web.Network == nil {
		t.Fatal("Expected network config, got nil")
	}

	if web.Network.Type != "bridge" {
		t.Errorf("Expected network type 'bridge', got '%s'", web.Network.Type)
	}

	if web.Storage == nil {
		t.Fatal("Expected storage config, got nil")
	}

	if web.Storage.Root != "10G" {
		t.Errorf("Expected storage root '10G', got '%s'", web.Storage.Root)
	}
}

func TestLoad_FileNotFound(t *testing.T) {
	_, err := config.Load("nonexistent-file.yml")
	if err == nil {
		t.Error("Load() should return error for nonexistent file")
	}

	expectedSubstring := "failed to read config file"
	if !containsSubstring(err.Error(), expectedSubstring) {
		t.Errorf("Error should contain '%s', got: %v", expectedSubstring, err)
	}
}

func TestLoad_InvalidYAML(t *testing.T) {
	// Create a temporary config file with invalid YAML
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "invalid-config.yml")

	invalidYAML := `
services:
  web:
    image: "nginx:alpine"
    invalid_yaml: [unclosed bracket
`

	err := os.WriteFile(configFile, []byte(invalidYAML), 0644)
	if err != nil {
		t.Fatalf("Failed to write test config file: %v", err)
	}

	_, err = config.Load(configFile)
	if err == nil {
		t.Error("Load() should return error for invalid YAML")
	}

	expectedSubstring := "failed to parse config file"
	if !containsSubstring(err.Error(), expectedSubstring) {
		t.Errorf("Error should contain '%s', got: %v", expectedSubstring, err)
	}
}

func TestLoad_EmptyFile(t *testing.T) {
	// Create an empty config file
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "empty-config.yml")

	err := os.WriteFile(configFile, []byte(""), 0644)
	if err != nil {
		t.Fatalf("Failed to write test config file: %v", err)
	}

	config, err := config.Load(configFile)
	if err != nil {
		t.Fatalf("Load() failed for empty file: %v", err)
	}

	if config == nil {
		t.Fatal("Load() returned nil config for empty file")
	}

	// For empty YAML files, Services map might be nil, which is acceptable

	if len(config.Services) != 0 {
		t.Errorf("Expected 0 services, got %d", len(config.Services))
	}
}

func TestValidateNetworkConfig_NilConfig(t *testing.T) {
	err := validation.ValidateNetworkConfig(nil)
	if err != nil {
		t.Errorf("ValidateNetworkConfig(nil) should return nil, got: %v", err)
	}
}

func TestValidateNetworkConfig_ValidConfigs(t *testing.T) {
	tests := []struct {
		name   string
		config *config.NetworkConfig
	}{
		{
			name:   "empty config",
			config: &config.NetworkConfig{},
		},
		{
			name: "bridge type",
			config: &config.NetworkConfig{
				Type: "bridge",
			},
		},
		{
			name: "veth type",
			config: &config.NetworkConfig{
				Type: "veth",
			},
		},
		{
			name: "bridge interface with bridge name",
			config: &config.NetworkConfig{
				Interfaces: []config.NetworkInterface{
					{
						Type:   "bridge",
						Bridge: "lxcbr0",
					},
				},
			},
		},
		{
			name: "veth interface with name",
			config: &config.NetworkConfig{
				Interfaces: []config.NetworkInterface{
					{
						Type:      "veth",
						Interface: "veth0",
					},
				},
			},
		},
		{
			name: "macvlan interface with IP",
			config: &config.NetworkConfig{
				Interfaces: []config.NetworkInterface{
					{
						Type: "macvlan",
						IP:   "192.168.1.100/24",
					},
				},
			},
		},
		{
			name: "interface with valid IP",
			config: &config.NetworkConfig{
				Interfaces: []config.NetworkInterface{
					{
						Type: "veth",
						IP:   "192.168.1.100/24",
					},
				},
			},
		},
		{
			name: "interface with valid gateway",
			config: &config.NetworkConfig{
				Interfaces: []config.NetworkInterface{
					{
						Type:    "veth",
						Gateway: "192.168.1.1",
					},
				},
			},
		},
		{
			name: "interface with valid MTU",
			config: &config.NetworkConfig{
				Interfaces: []config.NetworkInterface{
					{
						Type: "veth",
						MTU:  1500,
					},
				},
			},
		},
		{
			name: "interface with valid MAC",
			config: &config.NetworkConfig{
				Interfaces: []config.NetworkInterface{
					{
						Type: "veth",
						MAC:  "aa:bb:cc:dd:ee:ff",
					},
				},
			},
		},
		{
			name: "valid port forwards",
			config: &config.NetworkConfig{
				PortForwards: []config.PortForward{
					{
						Protocol: "tcp",
						Host:     8080,
						Guest:    80,
					},
					{
						Protocol: "udp",
						Host:     53,
						Guest:    53,
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validation.ValidateNetworkConfig(tt.config)
			if err != nil {
				t.Errorf("ValidateNetworkConfig() failed for valid config: %v", err)
			}
		})
	}
}

func TestValidateNetworkConfig_InvalidConfigs(t *testing.T) {
	tests := []struct {
		name          string
		config        *config.NetworkConfig
		expectedError string
	}{
		{
			name: "invalid network type",
			config: &config.NetworkConfig{
				Type: "invalid",
			},
			expectedError: "invalid network type: invalid",
		},
		{
			name: "invalid interface type",
			config: &config.NetworkConfig{
				Interfaces: []config.NetworkInterface{
					{
						Type: "invalid",
					},
				},
			},
			expectedError: "interface 0: invalid network type: invalid",
		},
		{
			name: "bridge interface without bridge name",
			config: &config.NetworkConfig{
				Interfaces: []config.NetworkInterface{
					{
						Type: "bridge",
					},
				},
			},
			expectedError: "interface 0: bridge name is required for bridge network type",
		},
		{
			name: "interface with invalid IP",
			config: &config.NetworkConfig{
				Interfaces: []config.NetworkInterface{
					{
						Type: "veth",
						IP:   "invalid-ip",
					},
				},
			},
			expectedError: "interface 0: invalid IP address",
		},
		{
			name: "interface with invalid gateway",
			config: &config.NetworkConfig{
				Interfaces: []config.NetworkInterface{
					{
						Type:    "veth",
						Gateway: "invalid-gateway",
					},
				},
			},
			expectedError: "interface 0: invalid gateway",
		},
		{
			name: "interface with MTU too low",
			config: &config.NetworkConfig{
				Interfaces: []config.NetworkInterface{
					{
						Type: "veth",
						MTU:  67,
					},
				},
			},
			expectedError: "interface 0: MTU must be between 68 and 65535",
		},
		{
			name: "interface with MTU too high",
			config: &config.NetworkConfig{
				Interfaces: []config.NetworkInterface{
					{
						Type: "veth",
						MTU:  65536,
					},
				},
			},
			expectedError: "interface 0: MTU must be between 68 and 65535",
		},
		{
			name: "interface with invalid MAC",
			config: &config.NetworkConfig{
				Interfaces: []config.NetworkInterface{
					{
						Type: "veth",
						MAC:  "invalid-mac",
					},
				},
			},
			expectedError: "interface 0: invalid MAC address",
		},
		{
			name: "port forward with invalid protocol",
			config: &config.NetworkConfig{
				PortForwards: []config.PortForward{
					{
						Protocol: "invalid",
						Host:     8080,
						Guest:    80,
					},
				},
			},
			expectedError: "port forward 0: protocol must be tcp or udp",
		},
		{
			name: "port forward with invalid host port (too low)",
			config: &config.NetworkConfig{
				PortForwards: []config.PortForward{
					{
						Protocol: "tcp",
						Host:     0,
						Guest:    80,
					},
				},
			},
			expectedError: "port forward 0: host port must be between 1 and 65535",
		},
		{
			name: "port forward with invalid host port (too high)",
			config: &config.NetworkConfig{
				PortForwards: []config.PortForward{
					{
						Protocol: "tcp",
						Host:     65536,
						Guest:    80,
					},
				},
			},
			expectedError: "port forward 0: host port must be between 1 and 65535",
		},
		{
			name: "port forward with invalid guest port (too low)",
			config: &config.NetworkConfig{
				PortForwards: []config.PortForward{
					{
						Protocol: "tcp",
						Host:     8080,
						Guest:    0,
					},
				},
			},
			expectedError: "port forward 0: guest port must be between 1 and 65535",
		},
		{
			name: "port forward with invalid guest port (too high)",
			config: &config.NetworkConfig{
				PortForwards: []config.PortForward{
					{
						Protocol: "tcp",
						Host:     8080,
						Guest:    65536,
					},
				},
			},
			expectedError: "port forward 0: guest port must be between 1 and 65535",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validation.ValidateNetworkConfig(tt.config)
			if err == nil {
				t.Error("ValidateNetworkConfig() should return error for invalid config")
			}

			if !containsSubstring(err.Error(), tt.expectedError) {
				t.Errorf("Error should contain '%s', got: %v", tt.expectedError, err)
			}
		})
	}
}

func TestValidateIPAddress(t *testing.T) {
	tests := []struct {
		name    string
		ip      string
		wantErr bool
		errMsg  string
	}{
		{"empty IP", "", false, ""},
		{"valid IPv4", "192.168.1.1", false, ""},
		{"valid IPv4 with CIDR", "192.168.1.1/24", false, ""},
		{"valid IPv6", "2001:db8::1", false, ""},
		{"valid IPv6 with CIDR", "2001:db8::1/64", false, ""},
		{"invalid IP format", "invalid-ip", true, "invalid IP address format"},
		{"invalid IPv4 CIDR too high", "192.168.1.1/33", true, "invalid IPv4 network prefix length: /33"},
		{"invalid IPv4 CIDR too low", "192.168.1.1/0", true, "invalid IPv4 network prefix length: /0"},
		{"invalid IPv6 CIDR too high", "2001:db8::1/129", true, "invalid IPv6 network prefix length: /129"},
		{"invalid IPv6 CIDR too low", "2001:db8::1/0", true, "invalid IPv6 network prefix length: /0"},
		{"invalid CIDR format", "192.168.1.1/abc", true, "invalid network prefix: abc"},
		{"valid edge case IPv4 CIDR", "192.168.1.1/1", false, ""},
		{"valid edge case IPv4 CIDR", "192.168.1.1/32", false, ""},
		{"valid edge case IPv6 CIDR", "2001:db8::1/1", false, ""},
		{"valid edge case IPv6 CIDR", "2001:db8::1/128", false, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validation.ValidateIPAddress(tt.ip)
			if tt.wantErr {
				if err == nil {
					t.Errorf("validateIPAddress(%s) should return error", tt.ip)
				} else if !containsSubstring(err.Error(), tt.errMsg) {
					t.Errorf("Error should contain '%s', got: %v", tt.errMsg, err)
				}
			} else {
				if err != nil {
					t.Errorf("validateIPAddress(%s) should not return error, got: %v", tt.ip, err)
				}
			}
		})
	}
}

func TestValidateMACAddress(t *testing.T) {
	tests := []struct {
		name    string
		mac     string
		wantErr bool
	}{
		{"empty MAC", "", false},
		{"valid MAC with colons", "aa:bb:cc:dd:ee:ff", false},
		{"valid MAC with hyphens", "aa-bb-cc-dd-ee-ff", false},
		{"valid MAC uppercase", "AA:BB:CC:DD:EE:FF", false},
		{"valid MAC mixed case", "Aa:Bb:Cc:Dd:Ee:Ff", false},
		{"valid MAC with numbers", "12:34:56:78:9a:bc", false},
		{"invalid MAC too short", "aa:bb:cc:dd:ee", true},
		{"invalid MAC too long", "aa:bb:cc:dd:ee:ff:gg", true},
		{"invalid MAC characters", "gg:hh:ii:jj:kk:ll", true},
		{"invalid MAC format", "aabbccddeeff", true},
		{"mixed separators (allowed by current regex)", "aa:bb-cc:dd-ee:ff", false},
		{"invalid MAC single character", "a:b:c:d:e:f", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validation.ValidateMACAddress(tt.mac)
			if tt.wantErr {
				if err == nil {
					t.Errorf("validateMACAddress(%s) should return error", tt.mac)
				}
			} else {
				if err != nil {
					t.Errorf("validateMACAddress(%s) should not return error, got: %v", tt.mac, err)
				}
			}
		})
	}
}

func TestYAMLMarshaling(t *testing.T) {
	// Test that all structs can be marshaled and unmarshaled correctly
	tests := []struct {
		name string
		data interface{}
	}{
		{
			name: "VPNConfig",
			data: &config.VPNConfig{
				Remote:   "vpn.example.com",
				Port:     1194,
				Protocol: "udp",
			},
		},
		{
			name: "BandwidthLimit",
			data: &config.BandwidthLimit{
				IngressRate:  "100Mbps",
				IngressBurst: "200MB",
				EgressRate:   "50Mbps",
				EgressBurst:  "100MB",
			},
		},
		{
			name: "NetworkInterface",
			data: &config.NetworkInterface{
				Type:         "bridge",
				Bridge:       "lxcbr0",
				Interface:    "eth0",
				IP:           "192.168.1.100/24",
				Gateway:      "192.168.1.1",
				DNS:          []string{"8.8.8.8", "8.8.4.4"},
				DHCP:         false,
				Hostname:     "container",
				MTU:          1500,
				MAC:          "aa:bb:cc:dd:ee:ff",
				BandwidthIn:  1000000, // 1 MB/s
				BandwidthOut: 500000,  // 500 KB/s
			},
		},
		{
			name: "PortForward",
			data: &config.PortForward{
				Protocol: "tcp",
				Host:     8080,
				Guest:    80,
			},
		},
		{
			name: "CPUConfig",
			data: &config.CPUConfig{
				Cores:  intPtr(2),
				Shares: int64Ptr(1024),
				Quota:  int64Ptr(100000),
				Period: int64Ptr(100000),
			},
		},
		{
			name: "MemoryConfig",
			data: &config.MemoryConfig{
				Limit:   "1G",
				Swap:    "2G",
				Reserve: "512M",
			},
		},
		{
			name: "Mount",
			data: &config.Mount{
				Source:  "/host/path",
				Target:  "/container/path",
				Type:    "bind",
				Options: []string{"ro", "bind"},
			},
		},
		{
			name: "StorageConfig",
			data: &config.StorageConfig{
				Root:      "10G",
				Backend:   "dir",
				Pool:      "default",
				AutoMount: true,
			},
		},
		{
			name: "SecurityConfig",
			data: &config.SecurityConfig{
				Isolation:       "default",
				Privileged:      true,
				AppArmorProfile: "unconfined",
			},
		},
		{
			name: "DeviceConfig",
			data: &config.DeviceConfig{
				Name:        "test-device",
				Type:        "disk",
				Source:      "/host/device",
				Destination: "/container/device",
			},
		},
		{
			name: "Container",
			data: &config.Container{
				Image: "nginx:alpine",
				Network: &config.NetworkConfig{
					Type:   "bridge",
					Bridge: "lxcbr0",
				},
				Storage: &config.StorageConfig{
					Root:    "10G",
					Backend: "dir",
				},
				Security: &config.SecurityConfig{
					Privileged: false,
				},
				Resources: &config.ResourceConfig{
					Cores:  2,
					Memory: "1G",
				},
				Command:    []string{"/bin/sh", "-c", "nginx"},
				Entrypoint: []string{"/entrypoint.sh"},
				Environment: map[string]string{
					"ANOTHER_VAR": "another_value",
				},
				Devices: []config.DeviceConfig{
					{Name: "dev1", Type: "disk", Source: "/dev/sdb1"},
				},
			},
		},
		{
			name: "ComposeConfig",
			data: &ComposeConfig{
				Services: map[string]Container{
					"web": {
						Image: "nginx:alpine",
						Network: &config.NetworkConfig{
							Type: "bridge",
						},
					},
					"db": {
						Image: "postgres:13",
						Storage: &StorageConfig{
							Root: "20G",
						},
					},
				},
			},
		},
		{
			name: "NetworkInterface with bandwidth limits",
			data: &config.NetworkInterface{
				Type:         "bridge",
				Bridge:       "lxcbr0",
				BandwidthIn:  100000000, // 100 MB/s
				BandwidthOut: 50000000,  // 50 MB/s
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Marshal to YAML
			yamlData, err := yaml.Marshal(tt.data)
			if err != nil {
				t.Fatalf("Failed to marshal %s to YAML: %v", tt.name, err)
			}

			// Create a new instance of the same type
			newData := reflect.New(reflect.TypeOf(tt.data).Elem()).Interface()

			// Unmarshal from YAML
			err = yaml.Unmarshal(yamlData, newData)
			if err != nil {
				t.Fatalf("Failed to unmarshal %s from YAML: %v", tt.name, err)
			}

			// Compare the original and unmarshaled data
			if !reflect.DeepEqual(tt.data, newData) {
				t.Errorf("YAML marshal/unmarshal roundtrip failed for %s", tt.name)
				t.Logf("Original: %+v", tt.data)
				t.Logf("Unmarshaled: %+v", newData)
			}
		})
	}
}

func TestNetworkConfig_DefaultInterfaceType(t *testing.T) {
	// Test that empty interface type defaults to "veth" during validation
	config := &config.NetworkConfig{
		Interfaces: []config.NetworkInterface{
			{
				// Type is empty, should default to "veth"
				IP: "192.168.1.100/24",
			},
		},
	}

	err := validation.ValidateNetworkConfig(config)
	if err != nil {
		t.Errorf("ValidateNetworkConfig() should succeed with default interface type: %v", err)
	}

	// Note: The validation function modifies the interface type in place
	// but doesn't persist it back to the original struct, so we can't test
	// that the default was actually set. This is a limitation of the current
	// implementation.
}

func TestComplexNetworkConfig(t *testing.T) {
	// Test a complex network configuration with multiple interfaces and port forwards
	config := &config.NetworkConfig{
		Type:   "bridge",
		Bridge: "lxcbr0",
		Interfaces: []config.NetworkInterface{
			{
				Type:         "bridge",
				Bridge:       "lxcbr0",
				IP:           "192.168.1.100/24",
				Gateway:      "192.168.1.1",
				DNS:          []string{"8.8.8.8", "8.8.4.4"},
				MTU:          1500,
				MAC:          "aa:bb:cc:dd:ee:ff",
				BandwidthIn:  1000000, // 1 MB/s
				BandwidthOut: 500000,  // 500 KB/s
			},
			{
				Type: "veth",
				IP:   "10.0.0.100/8",
			},
		},
		PortForwards: []config.PortForward{
			{Protocol: "tcp", Host: 8080, Guest: 80},
			{Protocol: "tcp", Host: 8443, Guest: 443},
			{Protocol: "udp", Host: 53, Guest: 53},
		},
	}

	err := validation.ValidateNetworkConfig(config)
	if err != nil {
		t.Errorf("ValidateNetworkConfig() should succeed for complex valid config: %v", err)
	}
}

// Helper functions
func containsSubstring(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
			containsAt(s, substr))))
}

func containsAt(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func int64Ptr(i int64) *int64 {
	return &i
}

func intPtr(i int) *int {
	return &i
}
