package container_test

import (
	"os"
	"path/filepath"
	"testing"

	"proxmox-lxc-compose/pkg/common"
	"proxmox-lxc-compose/pkg/container"
	"proxmox-lxc-compose/pkg/internal/mock"
	testing_internal "proxmox-lxc-compose/pkg/internal/testing"
)

func TestConfigureNetwork(t *testing.T) {
	dir, cleanup := testing_internal.TempDir(t)
	defer cleanup()

	// Set required environment variable for mock
	os.Setenv("CONTAINER_CONFIG_PATH", dir)
	defer os.Unsetenv("CONTAINER_CONFIG_PATH")

	// Create test container directory
	containerName := "test-container"
	containerDir := filepath.Join(dir, containerName)
	err := os.MkdirAll(containerDir, 0755)
	testing_internal.AssertNoError(t, err)

	// Setup mock command
	mockCmd, mockCleanup := mock.SetupMockCommand(&container.ExecCommand)
	defer mockCleanup()
	mockCmd.SetDebug(true)

	tests := []struct {
		name    string
		config  *common.NetworkConfig
		wantErr bool
		verify  func(t *testing.T, configPath string)
	}{
		{
			name: "DHCP configuration",
			config: &common.NetworkConfig{
				Type:      "veth",
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
				testing_internal.AssertContains(t, content, "lxc.net.0.type = veth")
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
			config: &common.NetworkConfig{
				Type:      "veth",
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
				testing_internal.AssertContains(t, content, "lxc.net.0.type = veth")
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
		{
			name: "multiple interfaces configuration",
			config: &common.NetworkConfig{
				Interfaces: []common.NetworkInterface{
					{
						Type:      "veth",
						Bridge:    "br0",
						Interface: "eth0",
						DHCP:      true,
						MTU:       1500,
					},
					{
						Type:      "veth",
						Bridge:    "br1",
						Interface: "eth1",
						IP:        "10.0.0.100/24",
						Gateway:   "10.0.0.1",
						DNS:       []string{"1.1.1.1"},
						MTU:       1500,
					},
				},
			},
			verify: func(t *testing.T, configPath string) {
				data, err := os.ReadFile(configPath)
				testing_internal.AssertNoError(t, err)
				content := string(data)

				// First interface
				testing_internal.AssertContains(t, content, "lxc.net.0.type = veth")
				testing_internal.AssertContains(t, content, "lxc.net.0.link = br0")
				testing_internal.AssertContains(t, content, "lxc.net.0.name = eth0")
				testing_internal.AssertContains(t, content, "lxc.net.0.ipv4.method = dhcp")
				testing_internal.AssertContains(t, content, "lxc.net.0.mtu = 1500")

				// Second interface
				testing_internal.AssertContains(t, content, "lxc.net.1.type = veth")
				testing_internal.AssertContains(t, content, "lxc.net.1.link = br1")
				testing_internal.AssertContains(t, content, "lxc.net.1.name = eth1")
				testing_internal.AssertContains(t, content, "lxc.net.1.ipv4.address = 10.0.0.100/24")
				testing_internal.AssertContains(t, content, "lxc.net.1.ipv4.gateway = 10.0.0.1")
				testing_internal.AssertContains(t, content, "lxc.net.1.ipv4.nameserver.0 = 1.1.1.1")
				testing_internal.AssertContains(t, content, "lxc.net.1.mtu = 1500")
			},
		},
		{
			name: "network with port forwarding",
			config: &common.NetworkConfig{
				Interfaces: []common.NetworkInterface{
					{
						Type:      "veth",
						Bridge:    "br0",
						Interface: "eth0",
						IP:        "192.168.1.100/24",
					},
				},
				PortForwards: []common.PortForward{
					{
						Protocol: "tcp",
						Host:     80,
						Guest:    8080,
					},
					{
						Protocol: "udp",
						Host:     53,
						Guest:    53,
					},
				},
			},
			verify: func(t *testing.T, configPath string) {
				data, err := os.ReadFile(configPath)
				testing_internal.AssertNoError(t, err)
				content := string(data)

				// Basic network config
				testing_internal.AssertContains(t, content, "lxc.net.0.type = veth")
				testing_internal.AssertContains(t, content, "lxc.net.0.link = br0")
				testing_internal.AssertContains(t, content, "lxc.net.0.name = eth0")
				testing_internal.AssertContains(t, content, "lxc.net.0.ipv4.address = 192.168.1.100/24")

				// Port forwards
				testing_internal.AssertContains(t, content, "lxc.hook.pre-start = iptables -t nat -A PREROUTING -p tcp --dport 80 -j DNAT --to 192.168.1.100:8080")
				testing_internal.AssertContains(t, content, "lxc.hook.pre-start = iptables -t nat -A PREROUTING -p udp --dport 53 -j DNAT --to 192.168.1.100:53")
				testing_internal.AssertContains(t, content, "lxc.hook.post-stop = iptables -t nat -D PREROUTING -p tcp --dport 80 -j DNAT --to 192.168.1.100:8080")
				testing_internal.AssertContains(t, content, "lxc.hook.post-stop = iptables -t nat -D PREROUTING -p udp --dport 53 -j DNAT --to 192.168.1.100:53")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clean up any existing container and state
			_ = os.RemoveAll(containerDir)
			err := os.MkdirAll(containerDir, 0755)
			testing_internal.AssertNoError(t, err)

			// Clean up state directory
			stateDir := filepath.Join(dir, "state")
			_ = os.RemoveAll(stateDir)
			err = os.MkdirAll(stateDir, 0755)
			testing_internal.AssertNoError(t, err)

			// Reset mock state
			mockCmd.RemoveContainer(containerName)

			// Create new manager for each test to ensure clean state
			manager, err := container.NewLXCManager(dir)
			testing_internal.AssertNoError(t, err)

			configPath := filepath.Join(containerDir, "network.conf")
			_ = os.Remove(configPath) // Clean up from previous test

			// Create container with network config
			containerCfg := &common.Container{
				Network: tt.config,
			}
			err = manager.Create(containerName, containerCfg)
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

	// Setup mock command
	mockCmd := mock.NewCommandState()
	mockCmd.SetDebug(true)

	// Add test container
	err = mockCmd.AddContainer(containerName, "STOPPED")
	if err != nil {
		t.Fatalf("Failed to add test container: %v", err)
	}

	// Create manager
	manager, err := container.NewLXCManager(dir)
	testing_internal.AssertNoError(t, err)

	// Test configurations
	tests := []struct {
		name    string
		config  *common.NetworkConfig
		wantErr bool
	}{
		{
			name: "DHCP configuration",
			config: &common.NetworkConfig{
				Type:      "veth",
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
			config: &common.NetworkConfig{
				Type:      "veth",
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
			// Clean up any existing container first
			_ = manager.Remove(containerName)
			_ = os.RemoveAll(containerDir)
			err := os.MkdirAll(containerDir, 0755)
			testing_internal.AssertNoError(t, err)

			// Clean up state directory
			stateDir := filepath.Join(dir, "state")
			_ = os.RemoveAll(stateDir)
			err = os.MkdirAll(stateDir, 0755)
			testing_internal.AssertNoError(t, err)

			// Reset mock state
			mockCmd.RemoveContainer(containerName)
			mockCmd.AddContainer(containerName, "STOPPED")

			// Create new manager for each test to ensure clean state
			manager, err = container.NewLXCManager(dir)
			testing_internal.AssertNoError(t, err)

			// Create container with network config
			containerCfg := &common.Container{
				Network: tt.config,
			}
			err = manager.Create(containerName, containerCfg)
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

// TestBasicNetworkConfiguration tests the basic network configuration functionality
// without relying on CLI commands
func TestBasicNetworkConfiguration(t *testing.T) {
	dir, cleanup := testing_internal.TempDir(t)
	defer cleanup()

	containerName := "test-network"
	containerDir := filepath.Join(dir, containerName)
	err := os.MkdirAll(containerDir, 0755)
	testing_internal.AssertNoError(t, err)

	// Setup mock command
	mockCmd := mock.NewCommandState()
	mockCmd.SetDebug(true)

	// Add test container
	err = mockCmd.AddContainer(containerName, "STOPPED")
	if err != nil {
		t.Fatalf("Failed to add test container: %v", err)
	}

	// Create manager
	manager, err := container.NewLXCManager(dir)
	testing_internal.AssertNoError(t, err)

	tests := []struct {
		name    string
		config  *common.NetworkConfig
		wantErr bool
		verify  func(t *testing.T, configPath string)
	}{
		{
			name: "basic_dhcp_config",
			config: &common.NetworkConfig{
				Type:      "veth",
				Bridge:    "br0",
				Interface: "eth0",
				DHCP:      true,
				Hostname:  "test-host",
			},
			wantErr: false,
			verify: func(t *testing.T, configPath string) {
				content, err := os.ReadFile(configPath)
				testing_internal.AssertNoError(t, err)
				testing_internal.AssertContains(t, string(content), "lxc.net.0.type = veth")
				testing_internal.AssertContains(t, string(content), "lxc.net.0.link = br0")
				testing_internal.AssertContains(t, string(content), "lxc.net.0.flags = up")
			},
		},
		{
			name: "static_ip_config",
			config: &common.NetworkConfig{
				Type:      "veth",
				Bridge:    "br0",
				Interface: "eth0",
				IP:        "192.168.1.100/24",
				Gateway:   "192.168.1.1",
				DNS:       []string{"8.8.8.8"},
			},
			wantErr: false,
			verify: func(t *testing.T, configPath string) {
				content, err := os.ReadFile(configPath)
				testing_internal.AssertNoError(t, err)
				testing_internal.AssertContains(t, string(content), "lxc.net.0.ipv4.address = 192.168.1.100/24")
				testing_internal.AssertContains(t, string(content), "lxc.net.0.ipv4.gateway = 192.168.1.1")
			},
		},
		{
			name: "invalid_interface",
			config: &common.NetworkConfig{
				Type:      "invalid",
				Interface: "eth0",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clean up any existing container first
			_ = manager.Remove(containerName)
			_ = os.RemoveAll(containerDir)
			err := os.MkdirAll(containerDir, 0755)
			testing_internal.AssertNoError(t, err)

			// Clean up state directory
			stateDir := filepath.Join(dir, "state")
			_ = os.RemoveAll(stateDir)
			err = os.MkdirAll(stateDir, 0755)
			testing_internal.AssertNoError(t, err)

			// Reset mock state
			mockCmd.RemoveContainer(containerName)
			mockCmd.AddContainer(containerName, "STOPPED")

			// Create new manager for each test to ensure clean state
			manager, err = container.NewLXCManager(dir)
			testing_internal.AssertNoError(t, err)

			configPath := filepath.Join(containerDir, "network.conf")
			_ = os.Remove(configPath) // Clean up from previous test

			containerCfg := &common.Container{
				Network: tt.config,
			}
			err = manager.Create(containerName, containerCfg)
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
