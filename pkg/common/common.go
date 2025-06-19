package common

import "github.com/larkinwc/proxmox-lxc-compose/pkg/config"

// Type aliases for backward compatibility with existing tests
type Container = config.Container
type ComposeConfig = config.ComposeConfig
type SecurityConfig = config.SecurityConfig
type StorageConfig = config.StorageConfig
type NetworkConfig = config.NetworkConfig
type NetworkInterface = config.NetworkInterface
type PortForward = config.PortForward
type BandwidthLimit = config.BandwidthLimit
type VPNConfig = config.VPNConfig
type Mount = config.Mount
type DeviceConfig = config.DeviceConfig

// Function aliases for backward compatibility
var Load = config.Load
