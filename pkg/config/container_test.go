package config

import (
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
