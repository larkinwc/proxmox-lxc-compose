package validation

import (
	"math"
	"strings"
	"testing"

	"github.com/larkinwc/proxmox-lxc-compose/pkg/config"
)

func TestValidateStorageSize(t *testing.T) {
	tests := []struct {
		name        string
		size        string
		wantBytes   int64
		wantErr     bool
		errContains string
	}{
		{
			name:      "bytes only",
			size:      "1024",
			wantBytes: 1024,
			wantErr:   false,
		},
		{
			name:      "kilobytes",
			size:      "1K",
			wantBytes: 1024,
			wantErr:   false,
		},
		{
			name:      "megabytes",
			size:      "1M",
			wantBytes: 1024 * 1024,
			wantErr:   false,
		},
		{
			name:      "gigabytes",
			size:      "1G",
			wantBytes: 1024 * 1024 * 1024,
			wantErr:   false,
		},
		{
			name:        "terabytes not supported",
			size:        "1T",
			wantErr:     true,
			errContains: "invalid size format",
		},
		{
			name:        "petabytes not supported",
			size:        "1P",
			wantErr:     true,
			errContains: "invalid size format",
		},
		{
			name:      "with B suffix",
			size:      "1GB",
			wantBytes: 1024 * 1024 * 1024,
			wantErr:   false,
		},
		{
			name:        "decimal value not supported",
			size:        "1.5G",
			wantErr:     true,
			errContains: "invalid size format",
		},
		{
			name:      "lowercase unit",
			size:      "1gb",
			wantBytes: 1024 * 1024 * 1024,
			wantErr:   false,
		},
		{
			name:        "invalid format",
			size:        "invalid",
			wantErr:     true,
			errContains: "invalid size format",
		},
		{
			name:        "negative value",
			size:        "-1G",
			wantErr:     true,
			errContains: "invalid size format",
		},
		{
			name:        "invalid unit",
			size:        "1X",
			wantErr:     true,
			errContains: "invalid size format",
		},
		{
			name:      "large value with supported unit",
			size:      "1024G",
			wantBytes: 1024 * 1024 * 1024 * 1024,
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ValidateStorageSize(tt.size)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateStorageSize() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil {
				if tt.errContains != "" && !contains(err.Error(), tt.errContains) {
					t.Errorf("error %q does not contain %q", err.Error(), tt.errContains)
				}
				return
			}
			if got != tt.wantBytes {
				t.Errorf("ValidateStorageSize() = %v, want %v", got, tt.wantBytes)
			}
		})
	}
}

func TestFormatBytes(t *testing.T) {
	tests := []struct {
		name     string
		bytes    int64
		expected string
	}{
		{
			name:     "zero bytes",
			bytes:    0,
			expected: "0B",
		},
		{
			name:     "bytes",
			bytes:    1023,
			expected: "1023B",
		},
		{
			name:     "exact kilobytes",
			bytes:    1024,
			expected: "1K",
		},
		{
			name:     "exact megabytes",
			bytes:    1024 * 1024,
			expected: "1M",
		},
		{
			name:     "exact gigabytes",
			bytes:    1024 * 1024 * 1024,
			expected: "1G",
		},
		{
			name:     "exact terabytes",
			bytes:    1024 * 1024 * 1024 * 1024,
			expected: "1024G",
		},
		{
			name:     "exact petabytes",
			bytes:    1024 * 1024 * 1024 * 1024 * 1024,
			expected: "1048576G",
		},
		{
			name:     "non-exact value",
			bytes:    2560,
			expected: "2K",
		},
		{
			name:     "maximum value",
			bytes:    math.MaxInt64,
			expected: "8589934591G",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatBytes(tt.bytes)
			if got != tt.expected {
				t.Errorf("FormatBytes() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestValidateStorageConfig(t *testing.T) {
	tests := []struct {
		name        string
		config      *config.StorageConfig
		wantErr     bool
		errContains string
	}{
		{
			name: "valid minimal config",
			config: &config.StorageConfig{
				Root:    "10G",
				Backend: "dir",
			},
			wantErr: false,
		},
		{
			name: "valid full config",
			config: &config.StorageConfig{
				Root:      "20G",
				Backend:   "zfs",
				Pool:      "lxc",
				AutoMount: true,
				Mounts: []config.MountConfig{
					{
						Source: "/tmp",
						Target: "/mnt/tmp",
						Type:   "bind",
					},
				},
			},
			wantErr: false,
		},
		{
			name:    "nil config",
			config:  nil,
			wantErr: false,
		},
		{
			name: "missing root size",
			config: &config.StorageConfig{
				Backend: "dir",
			},
			wantErr: false,
		},
		{
			name: "invalid root size",
			config: &config.StorageConfig{
				Root:    "invalid",
				Backend: "dir",
			},
			wantErr:     true,
			errContains: "invalid size format",
		},
		{
			name: "backend not validated by ValidateStorageConfig",
			config: &config.StorageConfig{
				Root:    "10G",
				Backend: "invalid",
			},
			wantErr: false,
		},
		{
			name: "pool not validated by ValidateStorageConfig",
			config: &config.StorageConfig{
				Root:    "10G",
				Backend: "zfs",
			},
			wantErr: false,
		},
		{
			name: "mounts not validated by ValidateStorageConfig",
			config: &config.StorageConfig{
				Root:    "10G",
				Backend: "dir",
				Mounts: []config.MountConfig{
					{
						Source: "", // Missing source
						Target: "/mnt",
						Type:   "bind",
					},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateStorageConfig(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateStorageConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil && tt.errContains != "" && !contains(err.Error(), tt.errContains) {
				t.Errorf("error %q does not contain %q", err.Error(), tt.errContains)
			}
		})
	}
}

func contains(s, substr string) bool {
	return s != "" && substr != "" && strings.Contains(s, substr)
}
