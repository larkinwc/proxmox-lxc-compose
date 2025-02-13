package config

// ComposeConfig represents the root configuration for lxc-compose.yml
type ComposeConfig struct {
	Version  string               `yaml:"version"`
	Services map[string]Container `yaml:"services"`
}

// Container represents a single LXC container configuration
type Container struct {
	Image       string            `yaml:"image"`
	Size        string            `yaml:"size,omitempty"` // Shorthand for root storage size
	CPU         *CPUConfig        `yaml:"cpu,omitempty"`
	Memory      *MemoryConfig     `yaml:"memory,omitempty"`
	Storage     *StorageConfig    `yaml:"storage,omitempty"`
	Network     *NetworkConfig    `yaml:"network,omitempty"`
	Environment map[string]string `yaml:"environment,omitempty"`
	Command     []string          `yaml:"command,omitempty"`
	Entrypoint  []string          `yaml:"entrypoint,omitempty"`
	Devices     []Device          `yaml:"devices,omitempty"`
}

// CPUConfig represents CPU resource limits
type CPUConfig struct {
	Shares *int64 `yaml:"shares,omitempty"`
	Quota  *int64 `yaml:"quota,omitempty"`
	Period *int64 `yaml:"period,omitempty"`
	Cores  *int   `yaml:"cores,omitempty"`
}

// MemoryConfig represents memory resource limits
type MemoryConfig struct {
	Limit   string `yaml:"limit,omitempty"`
	Swap    string `yaml:"swap,omitempty"`
	Reserve string `yaml:"reserve,omitempty"`
}

// StorageConfig represents storage configuration
type StorageConfig struct {
	Root      string  `yaml:"root,omitempty"`      // Root filesystem size
	Backend   string  `yaml:"backend,omitempty"`   // Storage backend (e.g., dir, zfs, btrfs)
	Pool      string  `yaml:"pool,omitempty"`      // Storage pool name
	Mounts    []Mount `yaml:"mounts,omitempty"`    // Additional mounts
	AutoMount bool    `yaml:"automount,omitempty"` // Whether to automount storage
}

// Mount represents a volume mount configuration
type Mount struct {
	Source  string   `yaml:"source"`
	Target  string   `yaml:"target"`
	Type    string   `yaml:"type,omitempty"`
	Options []string `yaml:"options,omitempty"`
}

// NetworkConfig represents network configuration
type NetworkConfig struct {
	Type      string   `yaml:"type"`
	Bridge    string   `yaml:"bridge,omitempty"`
	Interface string   `yaml:"interface,omitempty"`
	IP        string   `yaml:"ip,omitempty"`
	Gateway   string   `yaml:"gateway,omitempty"`
	DNS       []string `yaml:"dns,omitempty"`
	DHCP      bool     `yaml:"dhcp,omitempty"`
	Hostname  string   `yaml:"hostname,omitempty"`
	MTU       int      `yaml:"mtu,omitempty"`
	MAC       string   `yaml:"mac,omitempty"`
}

// Device represents a device that should be made available to the container
type Device struct {
	Name        string   `yaml:"name"`
	Type        string   `yaml:"type"`
	Source      string   `yaml:"source,omitempty"`
	Destination string   `yaml:"destination,omitempty"`
	Options     []string `yaml:"options,omitempty"`
}

// DefaultStorageConfig returns default storage configuration
func (c *Container) DefaultStorageConfig() *StorageConfig {
	// If storage is already configured, return it
	if c.Storage != nil {
		return c.Storage
	}

	// Create default storage config
	storage := &StorageConfig{
		Backend:   "dir", // Use directory backend by default
		AutoMount: true,  // Enable automounting by default
	}

	// Set root size from container size or default
	if c.Size != "" {
		storage.Root = c.Size
	} else {
		storage.Root = "10G" // Default root size
	}

	return storage
}
