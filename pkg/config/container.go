package config

import (
	"fmt"

	"proxmox-lxc-compose/pkg/validation"
)

func validateContainerConfig(container *Container) error {
	if container == nil {
		return fmt.Errorf("container configuration is required")
	}

	// Validate storage size
	if container.Storage != nil {
		bytes, err := validation.ValidateStorageSize(container.Storage.Root)
		if err != nil {
			return fmt.Errorf("invalid storage size: %w", err)
		}
		container.Storage.Root = validation.FormatBytes(bytes)
	}

	// Validate network configuration
	if container.Network != nil {
		err := validation.ValidateNetworkConfig(
			container.Network.Type,
			container.Network.Bridge,
			container.Network.Interface,
			container.Network.IP,
			container.Network.Gateway,
			container.Network.DNS,
			container.Network.DHCP,
			container.Network.Hostname,
			container.Network.MTU,
			container.Network.MAC,
		)
		if err != nil {
			return fmt.Errorf("invalid network configuration: %w", err)
		}
	}

	// Validate device configuration
	if len(container.Devices) > 0 {
		for _, device := range container.Devices {
			err := validation.ValidateDevice(device.Name, device.Type, device.Source, device.Destination, device.Options)
			if err != nil {
				return fmt.Errorf("invalid device configuration: %w", err)
			}
		}
	}

	return nil
}
