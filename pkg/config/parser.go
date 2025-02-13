package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// LoadConfig loads and parses the lxc-compose.yml file
func LoadConfig(path string) (*ComposeConfig, error) {
	if path == "" {
		path = "lxc-compose.yml"
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path: %w", err)
	}

	data, err := os.ReadFile(absPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config ComposeConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	if err := validateConfig(&config); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	return &config, nil
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

	return nil
}
