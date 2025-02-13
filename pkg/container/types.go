package container

import "proxmox-lxc-compose/pkg/config"

// Container represents a container instance
type Container struct {
	Name   string
	State  string
	Config *config.Container
}
