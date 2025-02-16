//go:build integration
// +build integration

package container_test

import (
	"net"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/larkinwc/proxmox-lxc-compose/pkg/container"
	testing_internal "github.com/larkinwc/proxmox-lxc-compose/pkg/internal/testing"
)

type integrationTest struct {
	name       string
	skip       bool
	skipReason string
	setup      func(t *testing.T) (*container.LXCManager, string, func())
	test       func(t *testing.T, manager *container.LXCManager, containerName string)
}

func setupIntegrationTest(t *testing.T) (*container.LXCManager, string, func()) {
	if !isRoot() {
		t.Skip("integration tests require root privileges")
	}

	dir, cleanup := testing_internal.TempDir(t)
	manager, err := container.NewLXCManager(dir)
	testing_internal.AssertNoError(t, err)

	containerName := "integration-test"
	return manager, containerName, cleanup
}

// func TestNetworkIntegration(t *testing.T) {
// 	if testing.Short() {
// 		t.Skip("skipping integration test in short mode")
// 	}

// 	tests := []integrationTest{
// 		{
// 			name: "DHCP Configuration",
// 			setup: func(t *testing.T) (*container.LXCManager, string, func()) {
// 				manager, containerName, cleanup := setupIntegrationTest(t)
// 				config := &common.Container{
// 					Image: "ubuntu:20.04",
// 					Network: &common.NetworkConfig{
// 						Interfaces: []common.NetworkInterface{
// 							{
// 								Type:      "bridge",
// 								Bridge:    "lxcbr0",
// 								Interface: "eth0",
// 								DHCP:      true,
// 							},
// 						},
// 					},
// 				}
// 				err := manager.Create(containerName, config)
// 				testing_internal.AssertNoError(t, err)
// 				return manager, containerName, cleanup
// 			},
// 			test: func(t *testing.T, manager *container.LXCManager, containerName string) {
// 				err := manager.Start(containerName)
// 				testing_internal.AssertNoError(t, err)

// 				// Wait for DHCP
// 				time.Sleep(5 * time.Second)

// 				output := runInContainer(t, containerName, "ip", "addr", "show", "eth0")
// 				if !containsIP(output) {
// 					t.Error("container did not receive IP from DHCP")
// 				}
// 			},
// 		},
// 	}

// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			if tt.skip {
// 				t.Skip(tt.skipReason)
// 			}

// 			manager, containerName, cleanup := tt.setup(t)
// 			defer cleanup()
// 			tt.test(t, manager, containerName)
// 		})
// 	}
// }

// func TestContainerLifecycle(t *testing.T) {
// 	if testing.Short() {
// 		t.Skip("skipping integration test in short mode")
// 	}

// 	manager, containerName, cleanup := setupIntegrationTest(t)
// 	defer cleanup()

// 	cfg := &config.Container{
// 		Image: "ubuntu:20.04",
// 	}
// 	commonCfg := cfg.ToCommonContainer()

// 	// Test full lifecycle
// 	t.Run("lifecycle", func(t *testing.T) {
// 		// Create
// 		err := manager.Create(containerName, commonCfg)
// 		testing_internal.AssertNoError(t, err)

// 		// Start
// 		err = manager.Start(containerName)
// 		testing_internal.AssertNoError(t, err)

// 		// Pause
// 		err = manager.Pause(containerName)
// 		testing_internal.AssertNoError(t, err)

// 		// Resume
// 		err = manager.Resume(containerName)
// 		testing_internal.AssertNoError(t, err)

// 		// Stop
// 		err = manager.Stop(containerName)
// 		testing_internal.AssertNoError(t, err)

// 		// Remove
// 		err = manager.Remove(containerName)
// 		testing_internal.AssertNoError(t, err)
// 	})
// }

// func TestLogStreaming(t *testing.T) {
// 	if testing.Short() {
// 		t.Skip("skipping integration test in short mode")
// 	}

// 	manager, containerName, cleanup := setupIntegrationTest(t)
// 	defer cleanup()

// 	cfg := &config.Container{
// 		Image: "ubuntu:20.04",
// 	}
// 	commonCfg := cfg.ToCommonContainer()

// 	err := manager.Create(containerName, commonCfg)
// 	testing_internal.AssertNoError(t, err)

// 	err = manager.Start(containerName)
// 	testing_internal.AssertNoError(t, err)

// 	t.Run("follow_logs", func(t *testing.T) {
// 		done := make(chan bool)
// 		logs, err := manager.GetLogs(containerName, container.LogOptions{
// 			Follow: true,
// 		})
// 		testing_internal.AssertNoError(t, err)

// 		go func() {
// 			buf := make([]byte, 1024)
// 			_, err := logs.Read(buf)
// 			testing_internal.AssertNoError(t, err)
// 			done <- true
// 		}()

// 		// Generate some logs
// 		runInContainer(t, containerName, "echo", "test log message")

// 		select {
// 		case <-done:
// 			// Success
// 		case <-time.After(5 * time.Second):
// 			t.Error("timeout waiting for log message")
// 		}
// 	})
// }

// func TestVPNConnectivity(t *testing.T) {
// 	if testing.Short() {
// 		t.Skip("skipping integration test in short mode")
// 	}

// 	manager, containerName, cleanup := setupIntegrationTest(t)
// 	defer cleanup()

// 	cfg := &config.Container{
// 		Image: "ubuntu:20.04",
// 	}
// 	commonCfg := cfg.ToCommonContainer()

// 	err := manager.Create(containerName, commonCfg)
// 	testing_internal.AssertNoError(t, err)

// 	vpnConfig := &common.VPNConfig{
// 		Remote:   os.Getenv("TEST_VPN_REMOTE"),
// 		Port:     1194,
// 		Protocol: "udp",
// 		CA:       os.Getenv("TEST_VPN_CA"),
// 	}

// 	if vpnConfig.Remote == "" || vpnConfig.CA == "" {
// 		t.Skip("TEST_VPN_REMOTE and TEST_VPN_CA environment variables required")
// 	}

// 	t.Run("vpn_connection", func(t *testing.T) {
// 		err := manager.ConfigureVPN(containerName, vpnConfig)
// 		testing_internal.AssertNoError(t, err)

// 		err = manager.Start(containerName)
// 		testing_internal.AssertNoError(t, err)

// 		// Wait for VPN connection
// 		time.Sleep(10 * time.Second)

// 		// Verify VPN interface exists
// 		output := runInContainer(t, containerName, "ip", "addr", "show", "tun0")
// 		if !strings.Contains(output, "tun0") {
// 			t.Error("VPN interface not found")
// 		}
// 	})
// }

// func TestLiveStats(t *testing.T) {
// 	if testing.Short() {
// 		t.Skip("skipping integration test in short mode")
// 	}

// 	manager, containerName, cleanup := setupIntegrationTest(t)
// 	defer cleanup()

// 	cfg := &config.Container{
// 		Image: "ubuntu:20.04",
// 		Resources: &config.ResourceConfig{
// 			CPUShares: 1024,
// 			Memory:    "128M",
// 		},
// 	}
// 	commonCfg := cfg.ToCommonContainer()

// 	err := manager.Create(containerName, commonCfg)
// 	testing_internal.AssertNoError(t, err)

// 	err = manager.Start(containerName)
// 	testing_internal.AssertNoError(t, err)

// 	t.Run("resource_stats", func(t *testing.T) {
// 		// Generate some CPU load
// 		go runInContainer(t, containerName, "yes", ">", "/dev/null")

// 		time.Sleep(2 * time.Second)

// 		stats, err := manager.GetCPUStats(containerName)
// 		testing_internal.AssertNoError(t, err)
// 		if stats.UsagePercentage == 0 {
// 			t.Error("expected non-zero CPU usage")
// 		}

// 		memStats, err := manager.GetMemoryStats(containerName)
// 		testing_internal.AssertNoError(t, err)
// 		if memStats.UsageBytes == 0 {
// 			t.Error("expected non-zero memory usage")
// 		}
// 	})
// }

func isRoot() bool {
	return os.Geteuid() == 0
}

func runInContainer(t *testing.T, container string, command ...string) string {
	cmd := exec.Command("lxc-attach", "-n", container, "--")
	cmd.Args = append(cmd.Args, command...)
	output, err := cmd.CombinedOutput()
	testing_internal.AssertNoError(t, err)
	return string(output)
}

func containsIP(output string) bool {
	for _, line := range strings.Split(output, "\n") {
		if strings.Contains(line, "inet ") {
			fields := strings.Fields(line)
			for _, field := range fields {
				ip, _, err := net.ParseCIDR(field)
				if err == nil && !ip.IsLoopback() {
					return true
				}
			}
		}
	}
	return false
}
