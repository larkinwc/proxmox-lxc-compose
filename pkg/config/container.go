package config

import (
	"fmt"
	"proxmox-lxc-compose/pkg/common"
	"strconv"
	"strings"
)

func ValidateContainer(container *common.Container) error {
	// Validate storage configuration
	if container.Storage != nil {
		bytes, err := ValidateStorageSize(container.Storage.Root)
		if err != nil {
			return err
		}
		container.Storage.Root = FormatBytes(bytes)
	}

	// Validate devices
	if len(container.Devices) > 0 {
		for _, device := range container.Devices {
			if err := ValidateDevice(&device); err != nil {
				return err
			}
		}
	}

	// Validate security configuration
	if container.Security != nil {
		if err := validateSecurityConfig(container.Security); err != nil {
			return err
		}
	}

	return nil
}

func ValidateStorageSize(size string) (int64, error) {
	if size == "" {
		return 0, nil
	}

	size = strings.ToUpper(size)
	multiplier := int64(1)

	if strings.HasSuffix(size, "G") || strings.HasSuffix(size, "GB") {
		multiplier = 1024 * 1024 * 1024
		size = strings.TrimSuffix(strings.TrimSuffix(size, "GB"), "G")
	} else if strings.HasSuffix(size, "M") || strings.HasSuffix(size, "MB") {
		multiplier = 1024 * 1024
		size = strings.TrimSuffix(strings.TrimSuffix(size, "MB"), "M")
	}

	value, err := strconv.ParseInt(size, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid storage size format: %s", size)
	}

	return value * multiplier, nil
}

func FormatBytes(bytes int64) string {
	const (
		GB = 1024 * 1024 * 1024
		MB = 1024 * 1024
	)

	if bytes >= GB && bytes%GB == 0 {
		return fmt.Sprintf("%dG", bytes/GB)
	}
	if bytes >= MB && bytes%MB == 0 {
		return fmt.Sprintf("%dM", bytes/MB)
	}
	return fmt.Sprintf("%d", bytes)
}

func ValidateDevice(device *common.DeviceConfig) error {
	if device.Name == "" {
		return fmt.Errorf("device name is required")
	}
	if device.Type == "" {
		return fmt.Errorf("device type is required")
	}
	if device.Source == "" {
		return fmt.Errorf("device source is required")
	}
	return nil
}

func validateSecurityConfig(config *common.SecurityConfig) error {
	if config == nil {
		return nil
	}

	validIsolationLevels := map[string]bool{
		"default":    true,
		"strict":     true,
		"privileged": true,
	}

	if !validIsolationLevels[config.Isolation] {
		return fmt.Errorf("invalid isolation level: %s", config.Isolation)
	}

	if config.Isolation == "strict" && config.Privileged {
		return fmt.Errorf("cannot use privileged mode with strict isolation")
	}

	// Validate capabilities
	validCaps := map[string]bool{
		"NET_ADMIN": true,
		"SYS_TIME":  true,
		"SYS_ADMIN": true,
	}

	for _, cap := range config.Capabilities {
		if !validCaps[cap] {
			return fmt.Errorf("invalid capability: %s", cap)
		}
	}

	return nil
}

// Convert legacy CPU/Memory configs to ResourceConfig
func (c *Container) migrateToResourceConfig() {
	if c.Resources == nil {
		c.Resources = &ResourceConfig{}
	}
}

// ToCommonContainer converts the configuration to a common.Container
func (c *Container) ToCommonContainer() *common.Container {
	c.migrateToResourceConfig()

	return &common.Container{
		Image:      c.Image,
		Storage:    c.Storage.ToCommonStorageConfig(),
		Network:    c.Network.ToCommonNetworkConfig(),
		Security:   c.Security.ToCommonSecurityConfig(),
		Command:    c.Command,
		Entrypoint: c.Entrypoint,
		Devices:    ToCommonDeviceConfigs(c.Devices),
		CPU: &common.CPUConfig{
			Cores:  &c.Resources.Cores,
			Shares: &c.Resources.CPUShares,
			Quota:  &c.Resources.CPUQuota,
			Period: &c.Resources.CPUPeriod,
		},
		Memory: &common.MemoryConfig{
			Limit: c.Resources.Memory,
			Swap:  c.Resources.MemorySwap,
		},
	}
}

// FromCommonContainer converts a common.Container to config.Container
func FromCommonContainer(c *common.Container) *Container {
	if c == nil {
		return nil
	}
	return &Container{
		Image:       c.Image,
		Network:     FromCommonNetworkConfig(c.Network),
		Storage:     FromCommonStorageConfig(c.Storage),
		Security:    FromCommonSecurityConfig(c.Security),
		Resources:   FromCommonResources(c.CPU, c.Memory),
		Devices:     FromCommonDeviceConfigs(c.Devices),
		Command:     c.Command,
		Entrypoint:  c.Entrypoint,
		Environment: c.Environment,
	}
}

// ToCommonNetworkConfig converts config.NetworkConfig to common.NetworkConfig
func (c *NetworkConfig) ToCommonNetworkConfig() *common.NetworkConfig {
	if c == nil {
		return nil
	}
	nc := &common.NetworkConfig{
		Type:         c.Type,
		Bridge:       c.Bridge,
		Interface:    c.Interface,
		IP:           c.IP,
		Gateway:      c.Gateway,
		DNS:          c.DNS,
		DHCP:         c.DHCP,
		Hostname:     c.Hostname,
		MTU:          c.MTU,
		MAC:          c.MAC,
		Interfaces:   make([]common.NetworkInterface, len(c.Interfaces)),
		PortForwards: make([]common.PortForward, len(c.PortForwards)),
	}

	for i, iface := range c.Interfaces {
		nc.Interfaces[i] = common.NetworkInterface{
			Type:      iface.Type,
			Bridge:    iface.Bridge,
			Interface: iface.Interface,
			IP:        iface.IP,
			Gateway:   iface.Gateway,
			DNS:       iface.DNS,
			DHCP:      iface.DHCP,
			Hostname:  iface.Hostname,
			MTU:       iface.MTU,
			MAC:       iface.MAC,
		}
	}

	for i, pf := range c.PortForwards {
		nc.PortForwards[i] = common.PortForward{
			Protocol: pf.Protocol,
			Host:     pf.Host,
			Guest:    pf.Guest,
		}
	}

	return nc
}

// FromCommonNetworkConfig converts common.NetworkConfig to config.NetworkConfig
func FromCommonNetworkConfig(c *common.NetworkConfig) *NetworkConfig {
	if c == nil {
		return nil
	}
	nc := &NetworkConfig{
		Type:         c.Type,
		Bridge:       c.Bridge,
		Interface:    c.Interface,
		IP:           c.IP,
		Gateway:      c.Gateway,
		DNS:          c.DNS,
		DHCP:         c.DHCP,
		Hostname:     c.Hostname,
		MTU:          c.MTU,
		MAC:          c.MAC,
		Interfaces:   make([]NetworkInterface, len(c.Interfaces)),
		PortForwards: make([]PortForward, len(c.PortForwards)),
	}

	for i, iface := range c.Interfaces {
		nc.Interfaces[i] = NetworkInterface{
			Type:      iface.Type,
			Bridge:    iface.Bridge,
			Interface: iface.Interface,
			IP:        iface.IP,
			Gateway:   iface.Gateway,
			DNS:       iface.DNS,
			DHCP:      iface.DHCP,
			Hostname:  iface.Hostname,
			MTU:       iface.MTU,
			MAC:       iface.MAC,
		}
	}

	for i, pf := range c.PortForwards {
		nc.PortForwards[i] = PortForward{
			Protocol: pf.Protocol,
			Host:     pf.Host,
			Guest:    pf.Guest,
		}
	}

	return nc
}

// ToCommonStorageConfig converts config.StorageConfig to common.StorageConfig
func (c *StorageConfig) ToCommonStorageConfig() *common.StorageConfig {
	if c == nil {
		return nil
	}
	return &common.StorageConfig{
		Root:      c.Root,
		Backend:   c.Backend,
		Pool:      c.Pool,
		AutoMount: c.AutoMount,
	}
}

// FromCommonStorageConfig converts common.StorageConfig to config.StorageConfig
func FromCommonStorageConfig(c *common.StorageConfig) *StorageConfig {
	if c == nil {
		return nil
	}
	return &StorageConfig{
		Root:      c.Root,
		Backend:   c.Backend,
		Pool:      c.Pool,
		AutoMount: c.AutoMount,
	}
}

// ToCommonCPUConfig converts config.CPUConfig to common.CPUConfig
func (c *CPUConfig) ToCommonCPUConfig() *common.CPUConfig {
	if c == nil {
		return nil
	}
	return &common.CPUConfig{
		Cores:  c.Cores,
		Shares: c.Shares,
		Quota:  c.Quota,
		Period: c.Period,
	}
}

// FromCommonResources converts common.CPUCOnfig and common.MemoryConfig to config.ResourceConfig
func FromCommonResources(c *common.CPUConfig, m *common.MemoryConfig) *ResourceConfig {
	if c == nil && m == nil {
		return nil
	}
	var shares, quota, period int64
	var memory, memorySwap string
	if c != nil {
		if c.Shares != nil {
			shares = *c.Shares
		}
		if c.Quota != nil {
			quota = *c.Quota
		}
		if c.Period != nil {
			period = *c.Period
		}
	}
	if m != nil {
		memory = m.Limit
		memorySwap = m.Swap
	}
	return &ResourceConfig{
		CPUShares:  shares,
		CPUQuota:   quota,
		CPUPeriod:  period,
		Memory:     memory,
		MemorySwap: memorySwap,
	}
}

// FromCommonCPUConfig converts common.CPUConfig to config.CPUConfig
func FromCommonCPUConfig(c *common.CPUConfig) *CPUConfig {
	if c == nil {
		return nil
	}
	return &CPUConfig{
		Cores:  c.Cores,
		Shares: c.Shares,
		Quota:  c.Quota,
		Period: c.Period,
	}
}

// ToCommonMemoryConfig converts config.MemoryConfig to common.MemoryConfig
func (c *MemoryConfig) ToCommonMemoryConfig() *common.MemoryConfig {
	if c == nil {
		return nil
	}
	return &common.MemoryConfig{
		Limit: c.Limit,
		Swap:  c.Swap,
	}
}

// FromCommonMemoryConfig converts common.MemoryConfig to config.MemoryConfig
func FromCommonMemoryConfig(c *common.MemoryConfig) *MemoryConfig {
	if c == nil {
		return nil
	}
	return &MemoryConfig{
		Limit: c.Limit,
		Swap:  c.Swap,
	}
}

// ToCommonDeviceConfigs converts []DeviceConfig to []common.DeviceConfig
func ToCommonDeviceConfigs(devices []DeviceConfig) []common.DeviceConfig {
	if devices == nil {
		return nil
	}
	commonDevices := make([]common.DeviceConfig, len(devices))
	for i, d := range devices {
		commonDevices[i] = common.DeviceConfig{
			Name:        d.Name,
			Type:        d.Type,
			Source:      d.Source,
			Destination: d.Destination,
			Options:     d.Options,
		}
	}
	return commonDevices
}

// FromCommonDeviceConfigs converts []common.DeviceConfig to []DeviceConfig
func FromCommonDeviceConfigs(devices []common.DeviceConfig) []DeviceConfig {
	if devices == nil {
		return nil
	}
	configDevices := make([]DeviceConfig, len(devices))
	for i, d := range devices {
		configDevices[i] = DeviceConfig{
			Name:        d.Name,
			Type:        d.Type,
			Source:      d.Source,
			Destination: d.Destination,
			Options:     d.Options,
		}
	}
	return configDevices
}

func (c *SecurityConfig) ToCommonSecurityConfig() *common.SecurityConfig {
	if c == nil {
		return nil
	}
	return &common.SecurityConfig{
		Isolation:       c.Isolation,
		Privileged:      c.Privileged,
		AppArmorProfile: c.AppArmorProfile,
		SeccompProfile:  c.SeccompProfile,
		SELinuxContext:  c.SELinuxContext,
		Capabilities:    c.Capabilities,
	}
}

func FromCommonSecurityConfig(c *common.SecurityConfig) *SecurityConfig {
	if c == nil {
		return nil
	}
	return &SecurityConfig{
		Isolation:       c.Isolation,
		Privileged:      c.Privileged,
		AppArmorProfile: c.AppArmorProfile,
		SeccompProfile:  c.SeccompProfile,
		SELinuxContext:  c.SELinuxContext,
		Capabilities:    c.Capabilities,
	}
}
