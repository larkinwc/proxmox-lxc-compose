package config

import (
	"fmt"
	"os"
	"proxmox-lxc-compose/pkg/validation"

	"gopkg.in/yaml.v3"
)

// Load loads a configuration from a file
func Load(path string) (*Container, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var container Container
	if err := yaml.Unmarshal(data, &container); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	if err := Validate(&container); err != nil {
		return nil, err
	}

	return &container, nil
}

func toValidationNetworkConfig(cfg *NetworkConfig) *validation.NetworkConfig {
	if cfg == nil {
		return nil
	}

	// Convert to validation package's type
	return &validation.NetworkConfig{
		Type:      cfg.Type,
		Bridge:    cfg.Bridge,
		Interface: cfg.Interface,
		IP:        cfg.IP,
		Gateway:   cfg.Gateway,
		DNS:       cfg.DNS,
		DHCP:      cfg.DHCP,
		Hostname:  cfg.Hostname,
		MTU:       cfg.MTU,
		MAC:       cfg.MAC,
	}
}

func toValidationSecurityProfile(cfg *SecurityConfig) *validation.SecurityProfile {
	if cfg == nil {
		return nil
	}
	return &validation.SecurityProfile{
		Isolation:    cfg.Isolation,
		Privileged:   cfg.Privileged,
		Capabilities: cfg.Capabilities,
	}
}

// Validate validates a container configuration
func Validate(container *Container) error {
	if container == nil {
		return fmt.Errorf("container configuration is required")
	}

	// Validate storage configuration
	if container.Storage != nil {
		bytes, err := validation.ValidateStorageSize(container.Storage.Root)
		if err != nil {
			return fmt.Errorf("invalid storage configuration: %w", err)
		}
		// Format the size consistently
		container.Storage.Root = validation.FormatBytes(bytes)
	}

	// Validate network configuration
	if container.Network != nil {
		if err := validation.ValidateNetworkConfig(toValidationNetworkConfig(container.Network)); err != nil {
			return fmt.Errorf("invalid network configuration: %w", err)
		}
	}

	// Validate security configuration
	if container.Security != nil {
		if err := validation.ValidateSecurityProfile(toValidationSecurityProfile(container.Security)); err != nil {
			return fmt.Errorf("invalid security configuration: %w", err)
		}
	}

	return nil
}

// validateConfig performs basic validation of the configuration
func validateConfig(config *ComposeConfig) error {
	if config.Version == "" {
		return fmt.Errorf("version is required")
	}

	if len(config.Services) == 0 {
		return fmt.Errorf("at least one service must be defined")
	}

	for name, container := range config.Services {
		if err := validateContainer(name, &container); err != nil {
			return err
		}
	}

	return nil
}

// validateContainer validates a single container configuration
func validateContainer(name string, container *Container) error {
	if container.Image == "" {
		return fmt.Errorf("service '%s' must specify an image", name)
	}

	// Apply storage defaults
	if container.Storage == nil {
		container.Storage = container.DefaultStorageConfig()
	}

	// Validate container configuration
	if err := validateContainerConfig(container); err != nil {
		return fmt.Errorf("service '%s' has invalid configuration: %w", name, err)
	}

	return nil
}
