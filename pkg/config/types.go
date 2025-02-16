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
	Resources   *ResourceConfig   `yaml:"resources,omitempty" json:"resources,omitempty"`
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
	AutoMount bool          `yaml:"auto_mount,omitempty" json:"auto_mount,omitempty"`
}

// MountConfig represents a volume mount configuration
type MountConfig struct {
	Source   string   `yaml:"source" json:"source"`
	Target   string   `yaml:"target" json:"target"`
	Type     string   `yaml:"type,omitempty" json:"type,omitempty"`
	Options  []string `yaml:"options,omitempty" json:"options,omitempty"`
	ReadOnly bool     `yaml:"read_only,omitempty" json:"read_only,omitempty"`
}

// NetworkInterface represents a single network interface configuration
type NetworkInterface struct {
	Type         string   `yaml:"type" json:"type"`
	Bridge       string   `yaml:"bridge,omitempty" json:"bridge,omitempty"`
	Interface    string   `yaml:"interface,omitempty" json:"interface,omitempty"`
	IP           string   `yaml:"ip,omitempty" json:"ip,omitempty"`
	Gateway      string   `yaml:"gateway,omitempty" json:"gateway,omitempty"`
	DNS          []string `yaml:"dns,omitempty" json:"dns,omitempty"`
	DHCP         bool     `yaml:"dhcp,omitempty" json:"dhcp,omitempty"`
	Hostname     string   `yaml:"hostname,omitempty" json:"hostname,omitempty"`
	MTU          int      `yaml:"mtu,omitempty" json:"mtu,omitempty"`
	MAC          string   `yaml:"mac,omitempty" json:"mac,omitempty"`
	BandwidthIn  int64    `yaml:"bandwidth_in,omitempty" json:"bandwidth_in,omitempty"`   // Ingress bandwidth limit in bytes per second
	BandwidthOut int64    `yaml:"bandwidth_out,omitempty" json:"bandwidth_out,omitempty"` // Egress bandwidth limit in bytes per second
}

// PortForward represents a port forwarding configuration
type PortForward struct {
	Protocol string `yaml:"protocol" json:"protocol"` // tcp or udp
	Host     int    `yaml:"host" json:"host"`         // host port
	Guest    int    `yaml:"guest" json:"guest"`       // container port
}

// NetworkConfig represents network configuration for a container
type NetworkConfig struct {
	Interfaces    []NetworkInterface `yaml:"interfaces" json:"interfaces"`
	PortForwards  []PortForward      `yaml:"port_forwards,omitempty" json:"port_forwards,omitempty"`
	DNSServers    []string           `yaml:"dns_servers,omitempty" json:"dns_servers,omitempty"`
	SearchDomains []string           `yaml:"search_domains,omitempty" json:"search_domains,omitempty"`
	Isolated      bool               `yaml:"isolated,omitempty" json:"isolated,omitempty"`
	VPN           *VPNConfig         `yaml:"vpn,omitempty" json:"vpn,omitempty"`

	// Legacy fields for backward compatibility
	Type      string   `yaml:"type,omitempty" json:"type,omitempty"`
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

// VPNConfig represents OpenVPN configuration
type VPNConfig struct {
	Remote   string            `yaml:"remote" json:"remote"`                     // VPN server address
	Port     int               `yaml:"port" json:"port"`                         // VPN server port
	Protocol string            `yaml:"protocol" json:"protocol"`                 // udp or tcp
	Config   string            `yaml:"config,omitempty" json:"config,omitempty"` // Path to OpenVPN config file
	Auth     map[string]string `yaml:"auth,omitempty" json:"auth,omitempty"`     // Authentication credentials
	CA       string            `yaml:"ca,omitempty" json:"ca,omitempty"`         // CA certificate content
	Cert     string            `yaml:"cert,omitempty" json:"cert,omitempty"`     // Client certificate content
	Key      string            `yaml:"key,omitempty" json:"key,omitempty"`       // Client key content
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
	storage := &StorageConfig{
		Backend:   "dir", // Use directory backend by default
		AutoMount: true,  // Enable automounting by default
		Root:      "10G", // Default root size
	}
	if c.Storage != nil {
		if c.Storage.Root != "" {
			storage.Root = c.Storage.Root
		}
		if c.Storage.Backend != "" {
			storage.Backend = c.Storage.Backend
		}
		if c.Storage.Pool != "" {
			storage.Pool = c.Storage.Pool
		}
		// Only override AutoMount in the case where we have a complete storage config
		// This maintains the default true value for partial configs
		if c.Storage.Pool != "" || (c.Storage.Root != "" && c.Storage.Backend != "") {
			storage.AutoMount = c.Storage.AutoMount
		}
	}
	return storage
}

// ResourceConfig defines resource limits and reservations
type ResourceConfig struct {
	Cores        int    `yaml:"cores,omitempty" json:"cores,omitempty"`
	CPUShares    int64  `yaml:"cpu_shares,omitempty" json:"cpu_shares,omitempty"`
	CPUQuota     int64  `yaml:"cpu_quota,omitempty" json:"cpu_quota,omitempty"`
	CPUPeriod    int64  `yaml:"cpu_period,omitempty" json:"cpu_period,omitempty"`
	Memory       string `yaml:"memory,omitempty" json:"memory,omitempty"`
	MemorySwap   string `yaml:"memory_swap,omitempty" json:"memory_swap,omitempty"`
	KernelMemory string `yaml:"kernel_memory,omitempty" json:"kernel_memory,omitempty"`
}
