package config

// ComposeConfig represents the root configuration from lxc-compose.yml
type ComposeConfig struct {
	// Version is the compose file format version
	Version string `yaml:"version" json:"version"`
	// Services defines the container configurations
	Services map[string]Container `yaml:"services" json:"services"`
}

// Container represents a single LXC container configuration
type Container struct {
	Image       string            `yaml:"image" json:"image"`
	Size        string            `yaml:"size,omitempty" json:"size,omitempty"`
	CPU         *CPUConfig        `yaml:"cpu,omitempty" json:"cpu,omitempty"`
	Memory      *MemoryConfig     `yaml:"memory,omitempty" json:"memory,omitempty"`
	Storage     *StorageConfig    `yaml:"storage,omitempty" json:"storage,omitempty"`
	Network     *NetworkConfig    `yaml:"network,omitempty" json:"network,omitempty"`
	Environment map[string]string `yaml:"environment,omitempty" json:"environment,omitempty"`
	Command     []string          `yaml:"command,omitempty" json:"command,omitempty"`
	Entrypoint  []string          `yaml:"entrypoint,omitempty" json:"entrypoint,omitempty"`
	Devices     []DeviceConfig    `yaml:"devices,omitempty" json:"devices,omitempty"`
	Security    *SecurityConfig   `yaml:"security,omitempty" json:"security,omitempty"`
}

// CPUConfig represents CPU resource limits
type CPUConfig struct {
	Shares *int64 `yaml:"shares,omitempty" json:"shares,omitempty"`
	Quota  *int64 `yaml:"quota,omitempty" json:"quota,omitempty"`
	Period *int64 `yaml:"period,omitempty" json:"period,omitempty"`
	Cores  *int   `yaml:"cores,omitempty" json:"cores,omitempty"`
}

// MemoryConfig represents memory resource limits
type MemoryConfig struct {
	Limit   string `yaml:"limit,omitempty" json:"limit,omitempty"`
	Swap    string `yaml:"swap,omitempty" json:"swap,omitempty"`
	Reserve string `yaml:"reserve,omitempty" json:"reserve,omitempty"`
}

// StorageConfig represents storage configuration
type StorageConfig struct {
	Root      string        `yaml:"root,omitempty" json:"root,omitempty"`
	Backend   string        `yaml:"backend,omitempty" json:"backend,omitempty"`
	Pool      string        `yaml:"pool,omitempty" json:"pool,omitempty"`
	Mounts    []MountConfig `yaml:"mounts,omitempty" json:"mounts,omitempty"`
	AutoMount bool          `yaml:"automount,omitempty" json:"auto_mount,omitempty"`
}

// MountConfig represents a volume mount configuration
type MountConfig struct {
	Source   string   `yaml:"source" json:"source"`
	Target   string   `yaml:"target" json:"target"`
	Type     string   `yaml:"type,omitempty" json:"type,omitempty"`
	Options  []string `yaml:"options,omitempty" json:"options,omitempty"`
	ReadOnly bool     `yaml:"read_only,omitempty" json:"read_only,omitempty"`
}

// NetworkConfig represents network configuration
type NetworkConfig struct {
	Type      string   `yaml:"type" json:"type"`
	Bridge    string   `yaml:"bridge,omitempty" json:"bridge,omitempty"`
	Interface string   `yaml:"interface,omitempty" json:"interface,omitempty"`
	IP        string   `yaml:"ip,omitempty" json:"ip,omitempty"`
	Gateway   string   `yaml:"gateway,omitempty" json:"gateway,omitempty"`
	DNS       []string `yaml:"dns,omitempty" json:"dns,omitempty"`
	DHCP      bool     `yaml:"dhcp,omitempty" json:"dhcp,omitempty"`
	Hostname  string   `yaml:"hostname,omitempty" json:"hostname,omitempty"`
	MTU       int      `yaml:"mtu,omitempty" json:"mtu,omitempty"`
	MAC       string   `yaml:"mac,omitempty" json:"mac,omitempty"`
}

// DeviceConfig represents a device configuration
type DeviceConfig struct {
	Name        string   `yaml:"name" json:"name"`
	Type        string   `yaml:"type" json:"type"`
	Source      string   `yaml:"source,omitempty" json:"source"`
	Destination string   `yaml:"destination,omitempty" json:"target"`
	Options     []string `yaml:"options,omitempty" json:"options,omitempty"`
}

// SecurityConfig represents container security settings
type SecurityConfig struct {
	Isolation       string   `yaml:"isolation,omitempty" json:"isolation,omitempty"`
	Privileged      bool     `yaml:"privileged,omitempty" json:"privileged,omitempty"`
	AppArmorProfile string   `yaml:"apparmor_profile,omitempty" json:"apparmor_profile,omitempty"`
	SELinuxContext  string   `yaml:"selinux_context,omitempty" json:"selinux_context,omitempty"`
	Capabilities    []string `yaml:"capabilities,omitempty" json:"capabilities,omitempty"`
	SeccompProfile  string   `yaml:"seccomp_profile,omitempty" json:"seccomp_profile,omitempty"`
}

// DefaultStorageConfig returns default storage configuration
func (c *Container) DefaultStorageConfig() *StorageConfig {
	if c.Storage != nil {
		return c.Storage
	}
	storage := &StorageConfig{
		Backend:   "dir", // Use directory backend by default
		AutoMount: true,  // Enable automounting by default
	}
	if c.Size != "" {
		storage.Root = c.Size
	} else {
		storage.Root = "10G" // Default root size
	}
	return storage
}
