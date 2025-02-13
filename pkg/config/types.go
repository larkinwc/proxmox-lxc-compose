package config

// ComposeConfig represents the root configuration for lxc-compose.yml
type ComposeConfig struct {
	Version  string               `yaml:"version"`
	Services map[string]Container `yaml:"services"`
}

// Container represents a single LXC container configuration
type Container struct {
	Image       string            `yaml:"image"`
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
	Root   string  `yaml:"root,omitempty"`
	Mounts []Mount `yaml:"mounts,omitempty"`
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
}

// Device represents a device that should be made available to the container
type Device struct {
	Name        string   `yaml:"name"`
	Type        string   `yaml:"type"`
	Source      string   `yaml:"source,omitempty"`
	Destination string   `yaml:"destination,omitempty"`
	Options     []string `yaml:"options,omitempty"`
}
