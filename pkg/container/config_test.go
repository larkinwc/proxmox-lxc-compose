package container_test

import (
	"testing"

	"github.com/larkinwc/proxmox-lxc-compose/pkg/config"
	"github.com/larkinwc/proxmox-lxc-compose/pkg/container"
	"github.com/larkinwc/proxmox-lxc-compose/pkg/internal/mock"
	testing_internal "github.com/larkinwc/proxmox-lxc-compose/pkg/internal/testing"
)

func TestContainerConfiguration(t *testing.T) {
	containerName := "test-container"

	t.Run("basic_config", func(t *testing.T) {
		tmpDir := t.TempDir()
		mock, cleanup := mock.SetupMockCommand(&execCommand)
		defer cleanup()

		manager, err := container.NewLXCManager(tmpDir)
		testing_internal.AssertNoError(t, err)

		cfg := &config.Container{
			Security: &config.SecurityConfig{
				Isolation: "strict",
			},
		}
		mock.AddContainer(containerName, "STOPPED")

		commonCfg := cfg.ToCommonContainer()
		err = manager.Create(containerName, commonCfg)
		testing_internal.AssertNoError(t, err)

		err = manager.Update(containerName, commonCfg)
		testing_internal.AssertNoError(t, err)
	})

	t.Run("full_config", func(t *testing.T) {
		tmpDir := t.TempDir()
		mock, cleanup := mock.SetupMockCommand(&execCommand)
		defer cleanup()

		manager, err := container.NewLXCManager(tmpDir)
		testing_internal.AssertNoError(t, err)

		cfg := &config.Container{
			Image: "ubuntu:20.04",
			Network: &config.NetworkConfig{
				Type:      "bridge",
				Bridge:    "br0",
				Interface: "eth0",
				IP:        "192.168.1.100/24",
				Gateway:   "192.168.1.1",
				DNS:       []string{"8.8.8.8"},
				DHCP:      false,
				Hostname:  "host1",
				MTU:       1500,
				MAC:       "00:11:22:33:44:55",
			},
			Security: &config.SecurityConfig{
				Isolation:  "strict",
				Privileged: false,
			},
			Resources: &config.ResourceConfig{
				CPUShares:  1024,
				Memory:     "1G",
				MemorySwap: "2G",
			},
		}

		mock.AddContainer(containerName, "STOPPED")
		commonCfg := cfg.ToCommonContainer()
		err = manager.Create(containerName, commonCfg)
		testing_internal.AssertNoError(t, err)

		// Verify configuration was applied
		container, err := manager.Get(containerName)
		testing_internal.AssertNoError(t, err)
		testing_internal.AssertEqual(t, cfg.Image, container.Config.Image)
		testing_internal.AssertEqual(t, cfg.Network.IP, container.Config.Network.IP)
		testing_internal.AssertEqual(t, cfg.Security.Isolation, container.Config.Security.Isolation)
	})

	t.Run("invalid_config", func(t *testing.T) {
		tmpDir := t.TempDir()
		mock, cleanup := mock.SetupMockCommand(&execCommand)
		defer cleanup()

		manager, err := container.NewLXCManager(tmpDir)
		testing_internal.AssertNoError(t, err)

		invalidCfg := &config.Container{
			Network: &config.NetworkConfig{
				Type: "invalid",
			},
		}

		mock.AddContainer(containerName, "STOPPED")
		commonCfg := invalidCfg.ToCommonContainer()
		err = manager.Create(containerName, commonCfg)
		testing_internal.AssertError(t, err)
	})
}
