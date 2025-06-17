package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Load loads the entire compose configuration from a file
func Load(configFile string) (*ComposeConfig, error) {
	data, err := os.ReadFile(configFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config ComposeConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return &config, nil
}

// LoadOne loads a single container configuration from a file
func LoadOne(path string) (*Container, error) {
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

		return container, nil
	}

	// Try to parse as a single container config
	var container Container
	if err := yaml.Unmarshal(data, &container); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return &container, nil
}

// Validate validates a container configuration
func Validate(container *Container) error {
	if container == nil {
		return fmt.Errorf("container configuration is required")
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

	return nil
}
