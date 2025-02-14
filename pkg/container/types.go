package container

import (
	"proxmox-lxc-compose/pkg/config"
)

// Container represents a running or stopped LXC container
type Container struct {
	Name   string            `json:"name"`
	State  string            `json:"state"`
	Config *config.Container `json:"config"`
}
