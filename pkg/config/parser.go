package config

import (
	"fmt"
	"os"

	"github.com/larkinwc/proxmox-lxc-compose/pkg/validation"

	"gopkg.in/yaml.v3"
)

// Load loads a configuration from a file
func Load(path string) (*Container, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// First try to parse as a compose config
	var composeConfig struct {
		Version  string                `yaml:"version"`
		Services map[string]*Container `yaml:"services"`
	}

	if err := yaml.Unmarshal(data, &composeConfig); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	if len(composeConfig.Services) > 0 {
		// Get the "app" container if it exists, otherwise get the first container
		container, exists := composeConfig.Services["app"]
		if !exists {
			// Get the first container
			for _, c := range composeConfig.Services {
				container = c
				break
			}
		}

		if container == nil {
			return nil, fmt.Errorf("invalid configuration: service is empty")
		}

		if err := validateContainer("app", container); err != nil {
			return nil, fmt.Errorf("invalid configuration: %w", err)
		}

		return container, nil
	}

	// Try to parse as a single container config
	var container Container
	if err := yaml.Unmarshal(data, &container); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	if err := validateContainer("default", &container); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
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
		fmt.Printf("DEBUG: Validating security config: %+v\n", container.Security)
		if err := validation.ValidateSecurityProfile(toValidationSecurityProfile(container.Security)); err != nil {
			return fmt.Errorf("invalid security configuration: %w", err)
		}
	}

	return nil
}

// ValidateConfig performs basic validation of the configuration
func ValidateConfig(config *ComposeConfig) error {
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
