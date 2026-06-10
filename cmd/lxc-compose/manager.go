package main

import (
	"fmt"

	"github.com/larkinwc/proxmox-lxc-compose/pkg/container"
)

// lxcConfigPath is the base path where LXC container state is stored.
// It is a package variable so tests can override it.
var lxcConfigPath = "/var/lib/lxc"

// newManager creates a container manager rooted at the configured LXC path.
func newManager() (*container.LXCManager, error) {
	manager, err := container.NewLXCManager(lxcConfigPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create container manager: %w", err)
	}
	return manager, nil
}
