package config

import (
	"strings"
	"testing"

	. "proxmox-lxc-compose/pkg/internal/testing"
)

func TestDefaultStorageConfig(t *testing.T) {
	tests := []struct {
		name     string
		input    *Container
		expected *StorageConfig
	}{
		{
			name: "default config with no size",
			input: &Container{
				Image: "ubuntu:20.04",
			},
			expected: &StorageConfig{
				Root:      "10G",
				Backend:   "dir",
				AutoMount: true,
			},
		},
		{
			name: "custom size specified",
			input: &Container{
				Image: "ubuntu:20.04",
				Size:  "50GB",
			},
			expected: &StorageConfig{
				Root:      "50GB",
				Backend:   "dir",
				AutoMount: true,
			},
		},
		{
			name: "existing storage config",
			input: &Container{
				Image: "ubuntu:20.04",
				Storage: &StorageConfig{
					Root:    "20G",
					Backend: "zfs",
					Pool:    "lxc",
				},
			},
			expected: &StorageConfig{
				Root:    "20G",
				Backend: "zfs",
				Pool:    "lxc",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.input.DefaultStorageConfig()
			AssertEqual(t, tt.expected.Root, result.Root)
			AssertEqual(t, tt.expected.Backend, result.Backend)
			AssertEqual(t, tt.expected.AutoMount, result.AutoMount)
			AssertEqual(t, tt.expected.Pool, result.Pool)
		})
	}
}

func TestSecurityConfig(t *testing.T) {
	tests := []struct {
		name        string
		config      *SecurityConfig
		wantErr     bool
		errContains string
	}{
		{
			name: "valid default config",
			config: &SecurityConfig{
				Isolation: "default",
			},
			wantErr: false,
		},
		{
			name: "valid strict config",
			config: &SecurityConfig{
				Isolation:       "strict",
				AppArmorProfile: "lxc-container-default",
				Capabilities:    []string{"NET_ADMIN", "SYS_TIME"},
			},
			wantErr: false,
		},
		{
			name: "privileged config",
			config: &SecurityConfig{
				Isolation:  "privileged",
				Privileged: true,
			},
			wantErr: false,
		},
		{
			name: "invalid isolation",
			config: &SecurityConfig{
				Isolation: "invalid",
			},
			wantErr:     true,
			errContains: "invalid isolation level",
		},
		{
			name: "invalid privileged strict combination",
			config: &SecurityConfig{
				Isolation:  "strict",
				Privileged: true,
			},
			wantErr:     true,
			errContains: "cannot use privileged mode with strict isolation",
		},
		{
			name: "invalid capability",
			config: &SecurityConfig{
				Isolation:    "default",
				Capabilities: []string{"INVALID_CAP"},
			},
			wantErr:     true,
			errContains: "invalid capability",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateSecurityConfig(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateSecurityConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && !strings.Contains(err.Error(), tt.errContains) {
				t.Errorf("validateSecurityConfig() error = %v, want error containing %v", err, tt.errContains)
			}
		})
	}
}
