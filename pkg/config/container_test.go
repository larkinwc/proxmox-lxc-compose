package config_test

import (
	"proxmox-lxc-compose/pkg/config"
	testing_internal "proxmox-lxc-compose/pkg/internal/testing"
	"strings"
	"testing"
)

func TestDefaultStorageConfig(t *testing.T) {
	tests := []struct {
		name     string
		input    *config.Container
		expected *config.StorageConfig
	}{
		{
			name: "default config with no size",
			input: &config.Container{
				Image: "ubuntu:20.04",
			},
			expected: &config.StorageConfig{
				Root:      "10G",
				Backend:   "dir",
				AutoMount: true,
			},
		},
		{
			name: "custom size specified",
			input: &config.Container{
				Image: "ubuntu:20.04",
				Size:  "50GB",
			},
			expected: &config.StorageConfig{
				Root:      "50GB",
				Backend:   "dir",
				AutoMount: true,
			},
		},
		{
			name: "existing storage config",
			input: &config.Container{
				Image: "ubuntu:20.04",
				Storage: &config.StorageConfig{
					Root:    "20G",
					Backend: "zfs",
					Pool:    "lxc",
				},
			},
			expected: &config.StorageConfig{
				Root:    "20G",
				Backend: "zfs",
				Pool:    "lxc",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.input.DefaultStorageConfig()
			testing_internal.AssertEqual(t, tt.expected.Root, result.Root)
			testing_internal.AssertEqual(t, tt.expected.Backend, result.Backend)
			testing_internal.AssertEqual(t, tt.expected.AutoMount, result.AutoMount)
			testing_internal.AssertEqual(t, tt.expected.Pool, result.Pool)
		})
	}
}

func TestSecurityConfig(t *testing.T) {
	tests := []struct {
		name        string
		config      *config.SecurityConfig
		wantErr     bool
		errContains string
	}{
		{
			name: "valid default config",
			config: &config.SecurityConfig{
				Isolation: "default",
			},
			wantErr: false,
		},
		{
			name: "valid strict config",
			config: &config.SecurityConfig{
				Isolation:       "strict",
				AppArmorProfile: "lxc-container-default",
				Capabilities:    []string{"NET_ADMIN", "SYS_TIME"},
			},
			wantErr: false,
		},
		{
			name: "privileged config",
			config: &config.SecurityConfig{
				Isolation:  "privileged",
				Privileged: true,
			},
			wantErr: false,
		},
		{
			name: "invalid isolation",
			config: &config.SecurityConfig{
				Isolation: "invalid",
			},
			wantErr:     true,
			errContains: "invalid isolation level",
		},
		{
			name: "invalid privileged strict combination",
			config: &config.SecurityConfig{
				Isolation:  "strict",
				Privileged: true,
			},
			wantErr:     true,
			errContains: "cannot use privileged mode with strict isolation",
		},
		{
			name: "invalid capability",
			config: &config.SecurityConfig{
				Isolation:    "default",
				Capabilities: []string{"INVALID_CAP"},
			},
			wantErr:     true,
			errContains: "invalid capability",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			container := &config.Container{
				Security: tt.config,
			}
			err := config.Validate(container)
			if (err != nil) != tt.wantErr {
				t.Errorf("Security validation error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && !strings.Contains(err.Error(), tt.errContains) {
				t.Errorf("Security validation error = %v, want error containing %v", err, tt.errContains)
			}
		})
	}
}

func TestContainerConfig(t *testing.T) {
	tests := []struct {
		name    string
		config  *config.Container
		wantErr bool
	}{
		{
			name: "valid config",
			config: &config.Container{
				Storage: &config.StorageConfig{
					Root: "10G",
				},
				Security: &config.SecurityConfig{
					Isolation: "strict",
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := config.Validate(tt.config)
			if tt.wantErr {
				testing_internal.AssertError(t, err)
			} else {
				testing_internal.AssertNoError(t, err)
			}
		})
	}
}

func TestContainerValidation(t *testing.T) {
	tests := []struct {
		name    string
		config  *config.SecurityConfig
		wantErr bool
	}{
		{
			name: "valid config",
			config: &config.SecurityConfig{
				Isolation: "strict",
			},
			wantErr: false,
		},
		{
			name: "invalid isolation",
			config: &config.SecurityConfig{
				Isolation: "invalid",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			container := &config.Container{
				Security: tt.config,
			}
			err := config.Validate(container)
			if tt.wantErr {
				testing_internal.AssertError(t, err)
			} else {
				testing_internal.AssertNoError(t, err)
			}
		})
	}
}
