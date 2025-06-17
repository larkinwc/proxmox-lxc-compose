package proxmox

import (
	"fmt"
)

type Client interface {
	SetContainerOptions(name string, options ContainerOptions) error
}

type client struct{}

func NewClient() (Client, error) {
	return &client{}, nil
}

func (c *client) SetContainerOptions(name string, options ContainerOptions) error {
	// This is a mock implementation.
	// In a real implementation, this would interact with the Proxmox API.
	fmt.Printf("Setting options for container %s: %+v\n", name, options)
	return nil
}

type ContainerOptions struct {
	Cores  *int
	Memory string
}
