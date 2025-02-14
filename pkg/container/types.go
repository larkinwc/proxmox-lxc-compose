package container

import (
	"proxmox-lxc-compose/pkg/config"
	"time"
)

// Container represents a running or stopped LXC container
type Container struct {
	Name   string            `json:"name"`
	State  string            `json:"state"`
	Config *config.Container `json:"config"`
}

// LogOptions represents the options for retrieving container logs
type LogOptions struct {
	Follow    bool      // Follow log output
	Tail      int       // Number of lines to show from the end of logs
	Since     time.Time // Show logs since timestamp
	Timestamp bool      // Show timestamps
}
