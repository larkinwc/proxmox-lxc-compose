package container_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/larkinwc/proxmox-lxc-compose/pkg/common"
	"github.com/larkinwc/proxmox-lxc-compose/pkg/container"
	"github.com/larkinwc/proxmox-lxc-compose/pkg/internal/mock"
	testing_internal "github.com/larkinwc/proxmox-lxc-compose/pkg/internal/testing"
)

func TestVPNConfiguration(t *testing.T) {
	dir, cleanup := testing_internal.TempDir(t)
	defer cleanup()

	manager, err := container.NewLXCManager(dir)
	testing_internal.AssertNoError(t, err)

	containerName := "test-vpn"
	mock, cleanup := mock.SetupMockCommand(&execCommand)
	defer cleanup()

	// Ensure container exists and is stopped
	mock.AddContainer(containerName, "STOPPED")

	// Create test container
	containerConfig := &common.Container{
		Image: "ubuntu:20.04",
	}
	err = manager.Create(containerName, containerConfig)
	testing_internal.AssertNoError(t, err)

	// Setup test VPN configuration
	vpnConfig := &common.VPNConfig{
		Remote:   "vpn.example.com",
		Port:     1194,
		Protocol: "udp",
		CA:       "test CA certificate",
		Auth: map[string]string{
			"username": "testuser",
			"password": "testpass",
		},
	}

	t.Run("configure_vpn", func(t *testing.T) {
		err := manager.ConfigureVPN(containerName, vpnConfig)
		testing_internal.AssertNoError(t, err)

		// Verify VPN files were created
		vpnDir := filepath.Join(dir, containerName, "vpn")
		files := []string{"ca.crt", "auth.conf", "client.conf"}
		for _, file := range files {
			_, err := os.Stat(filepath.Join(vpnDir, file))
			testing_internal.AssertNoError(t, err)
		}
	})

	t.Run("remove_vpn", func(t *testing.T) {
		err := manager.RemoveVPN(containerName)
		testing_internal.AssertNoError(t, err)

		// Verify VPN directory was removed
		vpnDir := filepath.Join(dir, containerName, "vpn")
		_, err = os.Stat(vpnDir)
		testing_internal.AssertError(t, err)
	})

	t.Run("vpn_validation", func(t *testing.T) {
		invalidConfig := &common.VPNConfig{
			Remote: "", // Missing required field
		}
		err := manager.ConfigureVPN(containerName, invalidConfig)
		testing_internal.AssertError(t, err)

		// Test with invalid container
		err = manager.ConfigureVPN("nonexistent", vpnConfig)
		testing_internal.AssertError(t, err)
	})
}

// Integration tests moved to integration_test.go
// TestVPNConnectivity - requires actual VPN server
// TestVPNReconnection - requires network manipulation
