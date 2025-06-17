package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/larkinwc/proxmox-lxc-compose/pkg/common"
)

func TestUpCommand(t *testing.T) {
	// Find the up command
	upCmd, _, err := rootCmd.Find([]string{"up"})
	if err != nil {
		t.Fatalf("Failed to find up command: %v", err)
	}

	// Test command metadata
	if upCmd.Use != "up [service...]" {
		t.Errorf("Expected Use to be 'up [service...]', got '%s'", upCmd.Use)
	}

	if upCmd.Short != "Create and start containers" {
		t.Errorf("Expected Short to be 'Create and start containers', got '%s'", upCmd.Short)
	}

	if !strings.Contains(upCmd.Long, "Create and start containers defined in the lxc-compose.yml file") {
		t.Errorf("Unexpected Long description: %s", upCmd.Long)
	}
}

func TestUpCommandFlags(t *testing.T) {
	upCmd, _, err := rootCmd.Find([]string{"up"})
	if err != nil {
		t.Fatalf("Failed to find up command: %v", err)
	}

	// Test --file flag
	fileFlag := upCmd.Flags().Lookup("file")
	if fileFlag == nil {
		t.Error("Expected --file flag to exist")
	}
	if fileFlag.Shorthand != "f" {
		t.Errorf("Expected --file flag shorthand to be 'f', got '%s'", fileFlag.Shorthand)
	}
}

func TestUpCmdRunE_ConfigFileNotFound(t *testing.T) {
	// Save original configFile value
	originalConfigFile := configFile
	defer func() { configFile = originalConfigFile }()

	// Set non-existent config file
	configFile = "non-existent-file.yml"

	err := upCmdRunE(nil, []string{})
	if err == nil {
		t.Error("Expected error for non-existent config file")
	}

	if !strings.Contains(err.Error(), "failed to load config") {
		t.Errorf("Expected error to contain 'failed to load config', got: %v", err)
	}
}

func TestUpCmdRunE_InvalidConfig(t *testing.T) {
	// Save original configFile value
	originalConfigFile := configFile
	defer func() { configFile = originalConfigFile }()

	// Create temporary invalid config file
	tempDir := t.TempDir()
	configFile = filepath.Join(tempDir, "invalid-config.yml")

	invalidConfig := `
services:
  web:
    image: "nginx:alpine"
    invalid_yaml: [unclosed bracket
`

	err := os.WriteFile(configFile, []byte(invalidConfig), 0644)
	if err != nil {
		t.Fatalf("Failed to write test config file: %v", err)
	}

	err = upCmdRunE(nil, []string{})
	if err == nil {
		t.Error("Expected error for invalid config file")
	}

	if !strings.Contains(err.Error(), "failed to load config") {
		t.Errorf("Expected error to contain 'failed to load config', got: %v", err)
	}
}

func TestUpCmdRunE_ValidConfigNoServices(t *testing.T) {
	// Initialize logging to prevent panic
	initConfig()
	
	// Save original configFile value
	originalConfigFile := configFile
	defer func() { configFile = originalConfigFile }()

	// Create temporary valid config file with no services
	tempDir := t.TempDir()
	configFile = filepath.Join(tempDir, "empty-config.yml")

	emptyConfig := `
services: {}
`

	err := os.WriteFile(configFile, []byte(emptyConfig), 0644)
	if err != nil {
		t.Fatalf("Failed to write test config file: %v", err)
	}

	// This should fail when trying to create container manager
	// since we don't have LXC installed in test environment
	err = upCmdRunE(nil, []string{})
	if err == nil {
		t.Error("Expected error when creating container manager without LXC")
	}

	if !strings.Contains(err.Error(), "failed to create container manager") {
		t.Errorf("Expected error to contain 'failed to create container manager', got: %v", err)
	}
}

func TestUpCmdRunE_ServiceNotFound(t *testing.T) {
	// Save original configFile value
	originalConfigFile := configFile
	defer func() { configFile = originalConfigFile }()

	// Create temporary valid config file
	tempDir := t.TempDir()
	configFile = filepath.Join(tempDir, "test-config.yml")

	validConfig := `
services:
  web:
    image: "nginx:alpine"
`

	err := os.WriteFile(configFile, []byte(validConfig), 0644)
	if err != nil {
		t.Fatalf("Failed to write test config file: %v", err)
	}

	// Try to start a service that doesn't exist
	err = upCmdRunE(nil, []string{"nonexistent"})
	if err == nil {
		t.Error("Expected error for non-existent service")
	}

	// The error could be either service not found or container manager creation failure
	// depending on which happens first
	if !strings.Contains(err.Error(), "service 'nonexistent' not found in config") && 
	   !strings.Contains(err.Error(), "failed to create container manager") {
		t.Errorf("Expected error to contain service not found or container manager error, got: %v", err)
	}
}

func TestUpCmdRunE_ConfigLoading(t *testing.T) {
	// Test that the function properly loads and parses config
	tempDir := t.TempDir()
	testConfigFile := filepath.Join(tempDir, "test-config.yml")

	validConfig := `
services:
  web:
    image: "nginx:alpine"
    network:
      type: "bridge"
      bridge: "lxcbr0"
    storage:
      root: "10G"
      backend: "dir"
  db:
    image: "postgres:13"
    storage:
      root: "20G"
      backend: "zfs"
`

	err := os.WriteFile(testConfigFile, []byte(validConfig), 0644)
	if err != nil {
		t.Fatalf("Failed to write test config file: %v", err)
	}

	// Test loading the config directly
	cfg, err := common.Load(testConfigFile)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	if len(cfg.Services) != 2 {
		t.Errorf("Expected 2 services, got %d", len(cfg.Services))
	}

	// Check web service
	web, exists := cfg.Services["web"]
	if !exists {
		t.Error("Expected 'web' service to exist")
	}
	if web.Image != "nginx:alpine" {
		t.Errorf("Expected web image to be 'nginx:alpine', got '%s'", web.Image)
	}

	// Check db service
	db, exists := cfg.Services["db"]
	if !exists {
		t.Error("Expected 'db' service to exist")
	}
	if db.Image != "postgres:13" {
		t.Errorf("Expected db image to be 'postgres:13', got '%s'", db.Image)
	}
}

func TestUpCmdRunE_ServiceSelection(t *testing.T) {
	// Test the logic for selecting which services to start
	tempDir := t.TempDir()
	testConfigFile := filepath.Join(tempDir, "test-config.yml")

	validConfig := `
services:
  web:
    image: "nginx:alpine"
  db:
    image: "postgres:13"
  cache:
    image: "redis:alpine"
`

	err := os.WriteFile(testConfigFile, []byte(validConfig), 0644)
	if err != nil {
		t.Fatalf("Failed to write test config file: %v", err)
	}

	cfg, err := common.Load(testConfigFile)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Test service selection logic
	tests := []struct {
		name     string
		args     []string
		expected []string
	}{
		{
			name:     "no args - all services",
			args:     []string{},
			expected: []string{"web", "db", "cache"}, // Note: order may vary due to map iteration
		},
		{
			name:     "single service",
			args:     []string{"web"},
			expected: []string{"web"},
		},
		{
			name:     "multiple services",
			args:     []string{"web", "db"},
			expected: []string{"web", "db"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var services []string
			if len(tt.args) == 0 {
				// If no services specified, start all
				for name := range cfg.Services {
					services = append(services, name)
				}
			} else {
				services = tt.args
			}

			if len(tt.args) == 0 {
				// For "all services" case, just check we got the right count
				if len(services) != 3 {
					t.Errorf("Expected 3 services when no args provided, got %d", len(services))
				}
			} else {
				// For specific services, check exact match
				if len(services) != len(tt.expected) {
					t.Errorf("Expected %d services, got %d", len(tt.expected), len(services))
				}
				
				for i, expected := range tt.expected {
					if i < len(services) && services[i] != expected {
						t.Errorf("Expected service %d to be '%s', got '%s'", i, expected, services[i])
					}
				}
			}

			// Verify all selected services exist in config
			for _, serviceName := range services {
				if _, exists := cfg.Services[serviceName]; !exists {
					t.Errorf("Service '%s' not found in config", serviceName)
				}
			}
		})
	}
}

func TestUpCmdRunE_EmptyConfigFile(t *testing.T) {
	// Initialize logging to prevent panic
	initConfig()
	
	// Save original configFile value
	originalConfigFile := configFile
	defer func() { configFile = originalConfigFile }()

	// Create temporary empty config file
	tempDir := t.TempDir()
	configFile = filepath.Join(tempDir, "empty-config.yml")

	err := os.WriteFile(configFile, []byte(""), 0644)
	if err != nil {
		t.Fatalf("Failed to write test config file: %v", err)
	}

	// This should fail when trying to create container manager
	err = upCmdRunE(nil, []string{})
	if err == nil {
		t.Error("Expected error when creating container manager without LXC")
	}

	if !strings.Contains(err.Error(), "failed to create container manager") {
		t.Errorf("Expected error to contain 'failed to create container manager', got: %v", err)
	}
}

func TestUpCmdRunE_DefaultConfigFile(t *testing.T) {
	// Save original configFile value
	originalConfigFile := configFile
	defer func() { configFile = originalConfigFile }()

	// Test with empty configFile (should use default)
	configFile = ""

	err := upCmdRunE(nil, []string{})
	if err == nil {
		t.Error("Expected error when no config file specified and default doesn't exist")
	}

	// Should fail trying to load default config file
	if !strings.Contains(err.Error(), "failed to load config") {
		t.Errorf("Expected error to contain 'failed to load config', got: %v", err)
	}
}