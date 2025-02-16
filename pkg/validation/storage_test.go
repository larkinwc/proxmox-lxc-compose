package validation

import (
	"math"
	"strings"
	"testing"

	"github.com/larkinwc/proxmox-lxc-compose/pkg/common"
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
			name:      "terabytes",
			size:      "1T",
			wantBytes: 1024 * 1024 * 1024 * 1024,
			wantErr:   false,
		},
		{
			name:      "petabytes",
			size:      "1P",
			wantBytes: 1024 * 1024 * 1024 * 1024 * 1024,
			wantErr:   false,
		},
		{
			name:      "with B suffix",
			size:      "1GB",
			wantBytes: 1024 * 1024 * 1024,
			wantErr:   false,
		},
		{
			name:      "decimal value",
			size:      "1.5G",
			wantBytes: int64(1.5 * float64(1024*1024*1024)),
			wantErr:   false,
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
			name:        "too large",
			size:        "1024P",
			wantErr:     true,
			errContains: "size too large",
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
			expected: "0",
		},
		{
			name:     "bytes",
			bytes:    1023,
			expected: "1023",
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
			expected: "1T",
		},
		{
			name:     "exact petabytes",
			bytes:    1024 * 1024 * 1024 * 1024 * 1024,
			expected: "1P",
		},
		{
			name:     "non-exact value",
			bytes:    2560,
			expected: "2.5K",
		},
		{
			name:     "maximum value",
			bytes:    math.MaxInt64,
			expected: "8E",
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
		config      *common.StorageConfig
		wantErr     bool
		errContains string
	}{
		{
			name: "valid minimal config",
			config: &common.StorageConfig{
				Root:    "10G",
				Backend: "dir",
			},
			wantErr: false,
		},
		{
			name: "valid full config",
			config: &common.StorageConfig{
				Root:      "20G",
				Backend:   "zfs",
				Pool:      "lxc",
				AutoMount: true,
				Mounts: []common.Mount{
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
			name:        "nil config",
			config:      nil,
			wantErr:     true,
			errContains: "storage configuration is required",
		},
		{
			name: "missing root size",
			config: &common.StorageConfig{
				Backend: "dir",
			},
			wantErr:     true,
			errContains: "root storage size is required",
		},
		{
			name: "invalid root size",
			config: &common.StorageConfig{
				Root:    "invalid",
				Backend: "dir",
			},
			wantErr:     true,
			errContains: "invalid size format",
		},
		{
			name: "invalid backend",
			config: &common.StorageConfig{
				Root:    "10G",
				Backend: "invalid",
			},
			wantErr:     true,
			errContains: "invalid storage backend",
		},
		{
			name: "missing pool for zfs",
			config: &common.StorageConfig{
				Root:    "10G",
				Backend: "zfs",
			},
			wantErr:     true,
			errContains: "storage pool is required for zfs backend",
		},
		{
			name: "invalid mount",
			config: &common.StorageConfig{
				Root:    "10G",
				Backend: "dir",
				Mounts: []common.Mount{
					{
						Source: "", // Missing source
						Target: "/mnt",
						Type:   "bind",
					},
				},
			},
			wantErr:     true,
			errContains: "mount source is required",
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
