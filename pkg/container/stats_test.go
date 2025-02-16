//go:build integration || tests || that || need || real || cgroups || access
// +build integration tests that need real cgroups access

package container_test

import (
	"testing"

	"proxmox-lxc-compose/pkg/common"
	"proxmox-lxc-compose/pkg/container"
	"proxmox-lxc-compose/pkg/internal/mock"
	testing_internal "proxmox-lxc-compose/pkg/internal/testing"
)

// Unit tests that can run in CICD
func TestContainerStats(t *testing.T) {
	// Create mock manager
	manager := container.NewMockLXCManager()

	mock, cleanup := mock.SetupMockCommand(&execCommand)
	defer cleanup()

	containerName := "test-stats"
	containerConfig := &common.Container{
		Network: &common.NetworkConfig{
			Interfaces: []common.NetworkInterface{
				{
					Type:      "bridge",
					Bridge:    "lxcbr0",
					Interface: "eth0",
				},
			},
		},
	}

	// Setup container in mock
	mock.AddContainer(containerName, "RUNNING")
	err := manager.Create(containerName, containerConfig)
	testing_internal.AssertNoError(t, err)

	t.Run("network_stats", func(t *testing.T) {
		// Mock network stats output
		mock.AddMockOutput("cat /sys/class/net/eth0/statistics/rx_bytes", []byte("1024000"))
		mock.AddMockOutput("cat /sys/class/net/eth0/statistics/tx_bytes", []byte("512000"))
		mock.AddMockOutput("cat /sys/class/net/eth0/statistics/rx_packets", []byte("1000"))
		mock.AddMockOutput("cat /sys/class/net/eth0/statistics/tx_packets", []byte("500"))

		stats, err := manager.GetNetworkStats(containerName, "eth0")
		testing_internal.AssertNoError(t, err)
		testing_internal.AssertEqual(t, int64(1024000), stats.BytesReceived)
		testing_internal.AssertEqual(t, int64(512000), stats.BytesSent)
		testing_internal.AssertEqual(t, int64(1000), stats.PacketsReceived)
		testing_internal.AssertEqual(t, int64(500), stats.PacketsSent)
	})

	t.Run("memory_stats", func(t *testing.T) {
		// Mock memory stats output
		mock.AddMockOutput("cat /sys/fs/cgroup/memory/memory.usage_in_bytes", []byte("104857600"))  // 100MB
		mock.AddMockOutput("cat /sys/fs/cgroup/memory/memory.limit_in_bytes", []byte("1073741824")) // 1GB

		stats, err := manager.GetMemoryStats(containerName)
		testing_internal.AssertNoError(t, err)
		testing_internal.AssertEqual(t, int64(104857600), stats.UsageBytes)
		testing_internal.AssertEqual(t, int64(1073741824), stats.LimitBytes)
	})

	t.Run("cpu_stats", func(t *testing.T) {
		// Mock CPU stats output
		mock.AddMockOutput("cat /sys/fs/cgroup/cpu/cpu.shares", []byte("1024"))
		mock.AddMockOutput("cat /proc/stat", []byte("cpu 1000 200 300 400 500 600 700 800"))

		stats, err := manager.GetCPUStats(containerName)
		testing_internal.AssertNoError(t, err)
		testing_internal.AssertEqual(t, int64(1024), stats.Shares)
		testing_internal.AssertNotEqual(t, int64(0), stats.UsagePercentage)
	})
}

// TestLiveStats moved to integration_test.go
