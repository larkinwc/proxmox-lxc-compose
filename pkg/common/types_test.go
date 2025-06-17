package common

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"gopkg.in/yaml.v3"
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
	config, err := Load(configFile)
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	if config == nil {
		t.Fatal("Load() returned nil config")
	}

	// Verify the loaded configuration
	if len(config.Services) != 1 {
		t.Errorf("Expected 1 service, got %d", len(config.Services))
	}

	web, exists := config.Services["web"]
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
	_, err := Load("nonexistent-file.yml")
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

	_, err = Load(configFile)
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

	config, err := Load(configFile)
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
	err := ValidateNetworkConfig(nil)
	if err != nil {
		t.Errorf("ValidateNetworkConfig(nil) should return nil, got: %v", err)
	}
}

func TestValidateNetworkConfig_ValidConfigs(t *testing.T) {
	tests := []struct {
		name   string
		config *NetworkConfig
	}{
		{
			name: "empty config",
			config: &NetworkConfig{},
		},
		{
			name: "bridge type",
			config: &NetworkConfig{
				Type: "bridge",
			},
		},
		{
			name: "veth type",
			config: &NetworkConfig{
				Type: "veth",
			},
		},
		{
			name: "bridge interface with bridge name",
			config: &NetworkConfig{
				Interfaces: []NetworkInterface{
					{
						Type:   "bridge",
						Bridge: "lxcbr0",
					},
				},
			},
		},
		{
			name: "veth interface",
			config: &NetworkConfig{
				Interfaces: []NetworkInterface{
					{
						Type: "veth",
					},
				},
			},
		},
		{
			name: "interface with valid IP",
			config: &NetworkConfig{
				Interfaces: []NetworkInterface{
					{
						Type: "veth",
						IP:   "192.168.1.100/24",
					},
				},
			},
		},
		{
			name: "interface with valid gateway",
			config: &NetworkConfig{
				Interfaces: []NetworkInterface{
					{
						Type:    "veth",
						Gateway: "192.168.1.1",
					},
				},
			},
		},
		{
			name: "interface with valid MTU",
			config: &NetworkConfig{
				Interfaces: []NetworkInterface{
					{
						Type: "veth",
						MTU:  1500,
					},
				},
			},
		},
		{
			name: "interface with valid MAC",
			config: &NetworkConfig{
				Interfaces: []NetworkInterface{
					{
						Type: "veth",
						MAC:  "aa:bb:cc:dd:ee:ff",
					},
				},
			},
		},
		{
			name: "valid port forwards",
			config: &NetworkConfig{
				PortForwards: []PortForward{
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
			err := ValidateNetworkConfig(tt.config)
			if err != nil {
				t.Errorf("ValidateNetworkConfig() failed for valid config: %v", err)
			}
		})
	}
}

func TestValidateNetworkConfig_InvalidConfigs(t *testing.T) {
	tests := []struct {
		name          string
		config        *NetworkConfig
		expectedError string
	}{
		{
			name: "invalid network type",
			config: &NetworkConfig{
				Type: "invalid",
			},
			expectedError: "invalid network type: invalid",
		},
		{
			name: "invalid interface type",
			config: &NetworkConfig{
				Interfaces: []NetworkInterface{
					{
						Type: "invalid",
					},
				},
			},
			expectedError: "interface 0: invalid network type: invalid",
		},
		{
			name: "bridge interface without bridge name",
			config: &NetworkConfig{
				Interfaces: []NetworkInterface{
					{
						Type: "bridge",
					},
				},
			},
			expectedError: "interface 0: bridge name is required for bridge network type",
		},
		{
			name: "interface with invalid IP",
			config: &NetworkConfig{
				Interfaces: []NetworkInterface{
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
			config: &NetworkConfig{
				Interfaces: []NetworkInterface{
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
			config: &NetworkConfig{
				Interfaces: []NetworkInterface{
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
			config: &NetworkConfig{
				Interfaces: []NetworkInterface{
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
			config: &NetworkConfig{
				Interfaces: []NetworkInterface{
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
			config: &NetworkConfig{
				PortForwards: []PortForward{
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
			config: &NetworkConfig{
				PortForwards: []PortForward{
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
			config: &NetworkConfig{
				PortForwards: []PortForward{
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
			config: &NetworkConfig{
				PortForwards: []PortForward{
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
			config: &NetworkConfig{
				PortForwards: []PortForward{
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
			err := ValidateNetworkConfig(tt.config)
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
			err := validateIPAddress(tt.ip)
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
			err := validateMACAddress(tt.mac)
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
			data: &VPNConfig{
				Remote:   "vpn.example.com",
				Port:     1194,
				Protocol: "udp",
				Config:   "/etc/openvpn/client.conf",
				Auth: map[string]string{
					"username": "user",
					"password": "pass",
				},
				CA:   "ca-cert-content",
				Cert: "client-cert-content",
				Key:  "client-key-content",
			},
		},
		{
			name: "BandwidthLimit",
			data: &BandwidthLimit{
				IngressRate:  "1mbit",
				IngressBurst: "2mbit",
				EgressRate:   "500kbit",
				EgressBurst:  "1mbit",
			},
		},
		{
			name: "NetworkInterface",
			data: &NetworkInterface{
				Type:      "bridge",
				Bridge:    "lxcbr0",
				Interface: "eth0",
				IP:        "192.168.1.100/24",
				Gateway:   "192.168.1.1",
				DNS:       []string{"8.8.8.8", "8.8.4.4"},
				DHCP:      false,
				Hostname:  "container",
				MTU:       1500,
				MAC:       "aa:bb:cc:dd:ee:ff",
				Bandwidth: &BandwidthLimit{
					IngressRate: "1mbit",
					EgressRate:  "500kbit",
				},
			},
		},
		{
			name: "PortForward",
			data: &PortForward{
				Protocol: "tcp",
				Host:     8080,
				Guest:    80,
			},
		},
		{
			name: "CPUConfig",
			data: &CPUConfig{
				Shares: int64Ptr(1024),
				Quota:  int64Ptr(50000),
				Period: int64Ptr(100000),
				Cores:  intPtr(2),
			},
		},
		{
			name: "MemoryConfig",
			data: &MemoryConfig{
				Limit: "1G",
				Swap:  "2G",
			},
		},
		{
			name: "Mount",
			data: &Mount{
				Source:  "/host/path",
				Target:  "/container/path",
				Type:    "bind",
				Options: []string{"ro", "bind"},
			},
		},
		{
			name: "StorageConfig",
			data: &StorageConfig{
				Root:      "10G",
				Backend:   "dir",
				Pool:      "default",
				AutoMount: true,
				Mounts: []Mount{
					{
						Source: "/host/data",
						Target: "/container/data",
						Type:   "bind",
					},
				},
			},
		},
		{
			name: "SecurityConfig",
			data: &SecurityConfig{
				Isolation:       "strict",
				Privileged:      false,
				AppArmorProfile: "lxc-container-default",
				SeccompProfile:  "default",
				SELinuxContext:  "system_u:system_r:container_t:s0",
				Capabilities:    []string{"NET_ADMIN", "SYS_TIME"},
			},
		},
		{
			name: "DeviceConfig",
			data: &DeviceConfig{
				Name:        "dev1",
				Type:        "disk",
				Source:      "/dev/sdb1",
				Destination: "/container/dev/sdb1",
				Options:     []string{"rw", "create=file"},
			},
		},
		{
			name: "Container",
			data: &Container{
				Image: "nginx:alpine",
				Network: &NetworkConfig{
					Type:   "bridge",
					Bridge: "lxcbr0",
				},
				Storage: &StorageConfig{
					Root:    "10G",
					Backend: "dir",
				},
				Security: &SecurityConfig{
					Isolation: "default",
				},
				CPU: &CPUConfig{
					Cores: intPtr(2),
				},
				Memory: &MemoryConfig{
					Limit: "1G",
				},
				Ports: []PortForward{
					{Protocol: "tcp", Host: 8080, Guest: 80},
				},
				Volumes: []string{"/host:/container"},
				Env: map[string]string{
					"ENV_VAR": "value",
				},
				Command:    []string{"/bin/sh", "-c", "nginx"},
				Entrypoint: []string{"/entrypoint.sh"},
				Environment: map[string]string{
					"ANOTHER_VAR": "another_value",
				},
				Devices: []DeviceConfig{
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
						Network: &NetworkConfig{
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
	config := &NetworkConfig{
		Interfaces: []NetworkInterface{
			{
				// Type is empty, should default to "veth"
				IP: "192.168.1.100/24",
			},
		},
	}

	err := ValidateNetworkConfig(config)
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
	config := &NetworkConfig{
		Type:   "bridge",
		Bridge: "lxcbr0",
		Interfaces: []NetworkInterface{
			{
				Type:    "bridge",
				Bridge:  "lxcbr0",
				IP:      "192.168.1.100/24",
				Gateway: "192.168.1.1",
				DNS:     []string{"8.8.8.8", "8.8.4.4"},
				MTU:     1500,
				MAC:     "aa:bb:cc:dd:ee:ff",
				Bandwidth: &BandwidthLimit{
					IngressRate:  "1mbit",
					IngressBurst: "2mbit",
					EgressRate:   "500kbit",
					EgressBurst:  "1mbit",
				},
			},
			{
				Type: "veth",
				IP:   "10.0.0.100/8",
			},
		},
		PortForwards: []PortForward{
			{Protocol: "tcp", Host: 8080, Guest: 80},
			{Protocol: "tcp", Host: 8443, Guest: 443},
			{Protocol: "udp", Host: 53, Guest: 53},
		},
	}

	err := ValidateNetworkConfig(config)
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