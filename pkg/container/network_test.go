package container

import (
	"os"
	"path/filepath"
	"testing"

	"proxmox-lxc-compose/pkg/config"
	"proxmox-lxc-compose/pkg/testutil"
)

func TestConfigureNetwork(t *testing.T) {
	dir, cleanup := testutil.TempDir(t)
	defer cleanup()

	// Create test container directory
	containerName := "test-container"
	containerDir := filepath.Join(dir, containerName)
	err := os.MkdirAll(containerDir, 0755)
	testutil.AssertNoError(t, err)

	// Create manager
	manager := &LXCManager{
		configPath: dir,
	}

	tests := []struct {
		name    string
		config  *config.NetworkConfig
		wantErr bool
		verify  func(t *testing.T, configPath string)
	}{
		{
			name: "DHCP configuration",
			config: &config.NetworkConfig{
				Type:      "bridge",
				Bridge:    "br0",
				Interface: "eth0",
				DHCP:      true,
				Hostname:  "test-host",
				MTU:       1500,
				MAC:       "00:11:22:33:44:55",
			},
			verify: func(t *testing.T, configPath string) {
				data, err := os.ReadFile(configPath)
				testutil.AssertNoError(t, err)
				content := string(data)

				// Verify DHCP settings
				testutil.AssertContains(t, content, "lxc.net.0.type = bridge")
				testutil.AssertContains(t, content, "lxc.net.0.link = br0")
				testutil.AssertContains(t, content, "lxc.net.0.name = eth0")
				testutil.AssertContains(t, content, "lxc.net.0.ipv4.method = dhcp")
				testutil.AssertContains(t, content, "lxc.net.0.ipv6.method = dhcp")
				testutil.AssertContains(t, content, "lxc.net.0.hostname = test-host")
				testutil.AssertContains(t, content, "lxc.net.0.mtu = 1500")
				testutil.AssertContains(t, content, "lxc.net.0.hwaddr = 00:11:22:33:44:55")
			},
		},
		{
			name: "static IP configuration",
			config: &config.NetworkConfig{
				Type:      "bridge",
				Bridge:    "br0",
				Interface: "eth0",
				IP:        "192.168.1.100/24",
				Gateway:   "192.168.1.1",
				DNS:       []string{"8.8.8.8", "8.8.4.4"},
				Hostname:  "test-host",
				MTU:       1500,
				MAC:       "00:11:22:33:44:55",
			},
			verify: func(t *testing.T, configPath string) {
				data, err := os.ReadFile(configPath)
				testutil.AssertNoError(t, err)
				content := string(data)

				// Verify static IP settings
				testutil.AssertContains(t, content, "lxc.net.0.type = bridge")
				testutil.AssertContains(t, content, "lxc.net.0.link = br0")
				testutil.AssertContains(t, content, "lxc.net.0.name = eth0")
				testutil.AssertContains(t, content, "lxc.net.0.ipv4.address = 192.168.1.100/24")
				testutil.AssertContains(t, content, "lxc.net.0.ipv4.gateway = 192.168.1.1")
				testutil.AssertContains(t, content, "lxc.net.0.ipv4.nameserver.0 = 8.8.8.8")
				testutil.AssertContains(t, content, "lxc.net.0.ipv4.nameserver.1 = 8.8.4.4")
				testutil.AssertContains(t, content, "lxc.net.0.hostname = test-host")
				testutil.AssertContains(t, content, "lxc.net.0.mtu = 1500")
				testutil.AssertContains(t, content, "lxc.net.0.hwaddr = 00:11:22:33:44:55")
			},
		},
		{
			name:   "nil config",
			config: nil,
			verify: func(t *testing.T, configPath string) {
				_, err := os.Stat(configPath)
				if !os.IsNotExist(err) {
					t.Error("expected no config file to be created")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			configPath := filepath.Join(containerDir, "network")
			os.Remove(configPath) // Clean up from previous test

			err := manager.configureNetwork(containerName, tt.config)
			if tt.wantErr {
				testutil.AssertError(t, err)
				return
			}
			testutil.AssertNoError(t, err)

			if tt.verify != nil {
				tt.verify(t, configPath)
			}
		})
	}
}

func TestGetNetworkConfig(t *testing.T) {
	dir, cleanup := testutil.TempDir(t)
	defer cleanup()

	// Create test container directory
	containerName := "test-container"
	containerDir := filepath.Join(dir, containerName)
	err := os.MkdirAll(containerDir, 0755)
	testutil.AssertNoError(t, err)

	// Create manager
	manager := &LXCManager{
		configPath: dir,
	}

	// Test configurations
	tests := []struct {
		name     string
		content  string
		expected *config.NetworkConfig
		wantErr  bool
	}{
		{
			name: "DHCP configuration",
			content: `lxc.net.0.type = bridge
lxc.net.0.link = br0
lxc.net.0.name = eth0
lxc.net.0.ipv4.method = dhcp
lxc.net.0.hostname = test-host
lxc.net.0.mtu = 1500
lxc.net.0.hwaddr = 00:11:22:33:44:55`,
			expected: &config.NetworkConfig{
				Type:      "bridge",
				Bridge:    "br0",
				Interface: "eth0",
				DHCP:      true,
				Hostname:  "test-host",
				MTU:       1500,
				MAC:       "00:11:22:33:44:55",
			},
		},
		{
			name: "static IP configuration",
			content: `lxc.net.0.type = bridge
lxc.net.0.link = br0
lxc.net.0.name = eth0
lxc.net.0.ipv4.address = 192.168.1.100/24
lxc.net.0.ipv4.gateway = 192.168.1.1
lxc.net.0.ipv4.nameserver.0 = 8.8.8.8
lxc.net.0.ipv4.nameserver.1 = 8.8.4.4
lxc.net.0.hostname = test-host
lxc.net.0.mtu = 1500
lxc.net.0.hwaddr = 00:11:22:33:44:55`,
			expected: &config.NetworkConfig{
				Type:      "bridge",
				Bridge:    "br0",
				Interface: "eth0",
				IP:        "192.168.1.100/24",
				Gateway:   "192.168.1.1",
				DNS:       []string{"8.8.8.8", "8.8.4.4"},
				Hostname:  "test-host",
				MTU:       1500,
				MAC:       "00:11:22:33:44:55",
			},
		},
		{
			name:     "no config file",
			content:  "",
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			configPath := filepath.Join(containerDir, "network")
			os.Remove(configPath) // Clean up from previous test

			if tt.content != "" {
				err := os.WriteFile(configPath, []byte(tt.content), 0644)
				testutil.AssertNoError(t, err)
			}

			cfg, err := manager.getNetworkConfig(containerName)
			if tt.wantErr {
				testutil.AssertError(t, err)
				return
			}
			testutil.AssertNoError(t, err)

			if tt.expected == nil {
				if cfg != nil {
					t.Error("expected nil config")
				}
				return
			}

			testutil.AssertEqual(t, tt.expected.Type, cfg.Type)
			testutil.AssertEqual(t, tt.expected.Bridge, cfg.Bridge)
			testutil.AssertEqual(t, tt.expected.Interface, cfg.Interface)
			testutil.AssertEqual(t, tt.expected.DHCP, cfg.DHCP)
			testutil.AssertEqual(t, tt.expected.IP, cfg.IP)
			testutil.AssertEqual(t, tt.expected.Gateway, cfg.Gateway)
			testutil.AssertEqual(t, tt.expected.Hostname, cfg.Hostname)
			testutil.AssertEqual(t, tt.expected.MTU, cfg.MTU)
			testutil.AssertEqual(t, tt.expected.MAC, cfg.MAC)

			if len(tt.expected.DNS) != len(cfg.DNS) {
				t.Errorf("expected %d DNS servers, got %d", len(tt.expected.DNS), len(cfg.DNS))
			} else {
				for i, dns := range tt.expected.DNS {
					testutil.AssertEqual(t, dns, cfg.DNS[i])
				}
			}
		})
	}
}
