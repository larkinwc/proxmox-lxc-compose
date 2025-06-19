package config_test

import (
	"strings"
	"testing"

	"github.com/larkinwc/proxmox-lxc-compose/pkg/config"
	testing_internal "github.com/larkinwc/proxmox-lxc-compose/pkg/internal/testing"
)

func TestDefaultStorageConfig(t *testing.T) {
	tests := []struct {
		name     string
		input    *config.Container
		expected *config.StorageConfig
	}{
		{
			name:  "default config with no storage config",
			input: &config.Container{},
			expected: &config.StorageConfig{
				Root:      "10G",
				Backend:   "dir",
				AutoMount: true,
			},
		},
		{
			name: "custom storage config specified",
			input: &config.Container{
				Storage: &config.StorageConfig{
					Root:      "50GB",
					Backend:   "dir",
					AutoMount: true,
				},
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
				Storage: &config.StorageConfig{
					Root:      "20G",
					Backend:   "zfs",
					Pool:      "lxc",
					AutoMount: false,
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
			t.Logf("Test case: %s", tt.name)
			t.Logf("Input Storage: %+v", tt.input.Storage)
			t.Logf("Result: %+v", result)
			t.Logf("Expected: %+v", tt.expected)
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
			err := config.ValidateContainer(container)
			if (err != nil) != tt.wantErr {
				t.Errorf("Security validation error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && tt.errContains != "" && err != nil && !strings.Contains(err.Error(), tt.errContains) {
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
			err := config.ValidateContainer(tt.config)
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
		name        string
		config      *config.SecurityConfig
		wantErr     bool
		errContains string
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
			wantErr:     true,
			errContains: "invalid isolation level",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			container := &config.Container{
				Security: tt.config,
			}
			err := config.ValidateContainer(container)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				} else if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("error %q does not contain %q", err.Error(), tt.errContains)
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestValidateContainerConfig(t *testing.T) {
	container := &config.Container{
		Network: &config.NetworkConfig{
			Type:      "bridge",
			Bridge:    "br0",
			Interface: "eth0",
			IP:        "192.168.1.100/24",
			Gateway:   "192.168.1.1",
			DNS:       []string{"8.8.8.8"},
			DHCP:      false,
			Hostname:  "host1",
			MTU:       1500,
			MAC:       "00:11:22:33:44:55",
		},
	}
	err := config.ValidateContainer(container)
	if err != nil {
		t.Errorf("Container validation error = %v", err)
	}
}
