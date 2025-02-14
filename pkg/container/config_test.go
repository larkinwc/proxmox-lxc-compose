package container_test

import (
	"proxmox-lxc-compose/pkg/config"
	"proxmox-lxc-compose/pkg/container"
	testing_internal "proxmox-lxc-compose/pkg/internal/testing"
	"testing"
)

func TestConfig(t *testing.T) {
	tmpDir := t.TempDir()

	manager, err := container.NewLXCManager(tmpDir)
	testing_internal.AssertNoError(t, err)

	// First create the container
	containerName := "test-container"
	cfg := &config.Container{
		Security: &config.SecurityConfig{
			Isolation: "strict",
		},
	}

	// Create container first
	err = manager.Create(containerName, cfg)
	testing_internal.AssertNoError(t, err)

	// Then update its configuration
	err = manager.Update(containerName, cfg)
	testing_internal.AssertNoError(t, err)
}
