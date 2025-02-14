package container_test

import (
	"os"
	"path/filepath"
	"testing"

	"proxmox-lxc-compose/pkg/config"
	"proxmox-lxc-compose/pkg/container"
	testing_internal "proxmox-lxc-compose/pkg/internal/testing"
)

func TestConfigureNetwork(t *testing.T) {
	dir, cleanup := testing_internal.TempDir(t)
	defer cleanup()

	// Create test container directory
	containerName := "test-container"
	containerDir := filepath.Join(dir, containerName)
	err := os.MkdirAll(containerDir, 0755)
	testing_internal.AssertNoError(t, err)

	// Create manager
	manager, err := container.NewLXCManager(dir)
	testing_internal.AssertNoError(t, err)

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
				testing_internal.AssertNoError(t, err)
				content := string(data)

				// Verify DHCP settings
				testing_internal.AssertContains(t, content, "lxc.net.0.type = bridge")
				testing_internal.AssertContains(t, content, "lxc.net.0.link = br0")
				testing_internal.AssertContains(t, content, "lxc.net.0.name = eth0")
				testing_internal.AssertContains(t, content, "lxc.net.0.ipv4.method = dhcp")
				testing_internal.AssertContains(t, content, "lxc.net.0.ipv6.method = dhcp")
				testing_internal.AssertContains(t, content, "lxc.net.0.hostname = test-host")
				testing_internal.AssertContains(t, content, "lxc.net.0.mtu = 1500")
				testing_internal.AssertContains(t, content, "lxc.net.0.hwaddr = 00:11:22:33:44:55")
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
				testing_internal.AssertNoError(t, err)
				content := string(data)

				// Verify static IP settings
				testing_internal.AssertContains(t, content, "lxc.net.0.type = bridge")
				testing_internal.AssertContains(t, content, "lxc.net.0.link = br0")
				testing_internal.AssertContains(t, content, "lxc.net.0.name = eth0")
				testing_internal.AssertContains(t, content, "lxc.net.0.ipv4.address = 192.168.1.100/24")
				testing_internal.AssertContains(t, content, "lxc.net.0.ipv4.gateway = 192.168.1.1")
				testing_internal.AssertContains(t, content, "lxc.net.0.ipv4.nameserver.0 = 8.8.8.8")
				testing_internal.AssertContains(t, content, "lxc.net.0.ipv4.nameserver.1 = 8.8.4.4")
				testing_internal.AssertContains(t, content, "lxc.net.0.hostname = test-host")
				testing_internal.AssertContains(t, content, "lxc.net.0.mtu = 1500")
				testing_internal.AssertContains(t, content, "lxc.net.0.hwaddr = 00:11:22:33:44:55")
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

			// Create and configure container with network config
			containerCfg := &config.Container{
				Network: tt.config,
			}
			err := manager.Create(containerName, containerCfg)
			if tt.wantErr {
				testing_internal.AssertError(t, err)
				return
			}
			testing_internal.AssertNoError(t, err)

			if tt.verify != nil {
				tt.verify(t, configPath)
			}
		})
	}
}

func TestGetContainerWithNetwork(t *testing.T) {
	dir, cleanup := testing_internal.TempDir(t)
	defer cleanup()

	// Create test container directory
	containerName := "test-container"
	containerDir := filepath.Join(dir, containerName)
	err := os.MkdirAll(containerDir, 0755)
	testing_internal.AssertNoError(t, err)

	// Create manager
	manager, err := container.NewLXCManager(dir)
	testing_internal.AssertNoError(t, err)

	// Test configurations
	tests := []struct {
		name    string
		config  *config.NetworkConfig
		wantErr bool
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
		},
		{
			name:   "no network config",
			config: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create container with network config
			containerCfg := &config.Container{
				Network: tt.config,
			}
			err := manager.Create(containerName, containerCfg)
			testing_internal.AssertNoError(t, err)

			// Get container and verify network config
			container, err := manager.Get(containerName)
			if tt.wantErr {
				testing_internal.AssertError(t, err)
				return
			}
			testing_internal.AssertNoError(t, err)

			if tt.config == nil {
				if container.Config.Network != nil {
					t.Error("expected nil network config")
				}
				return
			}

			cfg := container.Config.Network
			testing_internal.AssertEqual(t, tt.config.Type, cfg.Type)
			testing_internal.AssertEqual(t, tt.config.Bridge, cfg.Bridge)
			testing_internal.AssertEqual(t, tt.config.Interface, cfg.Interface)
			testing_internal.AssertEqual(t, tt.config.DHCP, cfg.DHCP)
			testing_internal.AssertEqual(t, tt.config.IP, cfg.IP)
			testing_internal.AssertEqual(t, tt.config.Gateway, cfg.Gateway)
			testing_internal.AssertEqual(t, tt.config.Hostname, cfg.Hostname)
			testing_internal.AssertEqual(t, tt.config.MTU, cfg.MTU)
			testing_internal.AssertEqual(t, tt.config.MAC, cfg.MAC)

			if len(tt.config.DNS) != len(cfg.DNS) {
				t.Errorf("expected %d DNS servers, got %d", len(tt.config.DNS), len(cfg.DNS))
			} else {
				for i, dns := range tt.config.DNS {
					testing_internal.AssertEqual(t, dns, cfg.DNS[i])
				}
			}
		})
	}
}
