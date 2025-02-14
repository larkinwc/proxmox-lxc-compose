package container

import (
	"os"
	"path/filepath"
	"testing"

	"proxmox-lxc-compose/pkg/config"
	. "proxmox-lxc-compose/pkg/internal/testing"
)

func TestConfigureNetwork(t *testing.T) {
	dir, cleanup := TempDir(t)
	defer cleanup()

	// Create test container directory
	containerName := "test-container"
	containerDir := filepath.Join(dir, containerName)
	err := os.MkdirAll(containerDir, 0755)
	AssertNoError(t, err)

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
				AssertNoError(t, err)
				content := string(data)

				// Verify DHCP settings
				AssertContains(t, content, "lxc.net.0.type = bridge")
				AssertContains(t, content, "lxc.net.0.link = br0")
				AssertContains(t, content, "lxc.net.0.name = eth0")
				AssertContains(t, content, "lxc.net.0.ipv4.method = dhcp")
				AssertContains(t, content, "lxc.net.0.ipv6.method = dhcp")
				AssertContains(t, content, "lxc.net.0.hostname = test-host")
				AssertContains(t, content, "lxc.net.0.mtu = 1500")
				AssertContains(t, content, "lxc.net.0.hwaddr = 00:11:22:33:44:55")
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
				AssertNoError(t, err)
				content := string(data)

				// Verify static IP settings
				AssertContains(t, content, "lxc.net.0.type = bridge")
				AssertContains(t, content, "lxc.net.0.link = br0")
				AssertContains(t, content, "lxc.net.0.name = eth0")
				AssertContains(t, content, "lxc.net.0.ipv4.address = 192.168.1.100/24")
				AssertContains(t, content, "lxc.net.0.ipv4.gateway = 192.168.1.1")
				AssertContains(t, content, "lxc.net.0.ipv4.nameserver.0 = 8.8.8.8")
				AssertContains(t, content, "lxc.net.0.ipv4.nameserver.1 = 8.8.4.4")
				AssertContains(t, content, "lxc.net.0.hostname = test-host")
				AssertContains(t, content, "lxc.net.0.mtu = 1500")
				AssertContains(t, content, "lxc.net.0.hwaddr = 00:11:22:33:44:55")
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
				AssertError(t, err)
				return
			}
			AssertNoError(t, err)

			if tt.verify != nil {
				tt.verify(t, configPath)
			}
		})
	}
}

func TestGetNetworkConfig(t *testing.T) {
	dir, cleanup := TempDir(t)
	defer cleanup()

	// Create test container directory
	containerName := "test-container"
	containerDir := filepath.Join(dir, containerName)
	err := os.MkdirAll(containerDir, 0755)
	AssertNoError(t, err)

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
				AssertNoError(t, err)
			}

			cfg, err := manager.getNetworkConfig(containerName)
			if tt.wantErr {
				AssertError(t, err)
				return
			}
			AssertNoError(t, err)

			if tt.expected == nil {
				if cfg != nil {
					t.Error("expected nil config")
				}
				return
			}

			AssertEqual(t, tt.expected.Type, cfg.Type)
			AssertEqual(t, tt.expected.Bridge, cfg.Bridge)
			AssertEqual(t, tt.expected.Interface, cfg.Interface)
			AssertEqual(t, tt.expected.DHCP, cfg.DHCP)
			AssertEqual(t, tt.expected.IP, cfg.IP)
			AssertEqual(t, tt.expected.Gateway, cfg.Gateway)
			AssertEqual(t, tt.expected.Hostname, cfg.Hostname)
			AssertEqual(t, tt.expected.MTU, cfg.MTU)
			AssertEqual(t, tt.expected.MAC, cfg.MAC)

			if len(tt.expected.DNS) != len(cfg.DNS) {
				t.Errorf("expected %d DNS servers, got %d", len(tt.expected.DNS), len(cfg.DNS))
			} else {
				for i, dns := range tt.expected.DNS {
					AssertEqual(t, dns, cfg.DNS[i])
				}
			}
		})
	}
}
