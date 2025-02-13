package config

import "testing"

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
			if result.Root != tt.expected.Root {
				t.Errorf("Root size mismatch: got %v, want %v", result.Root, tt.expected.Root)
			}
			if result.Backend != tt.expected.Backend {
				t.Errorf("Backend mismatch: got %v, want %v", result.Backend, tt.expected.Backend)
			}
			if result.AutoMount != tt.expected.AutoMount {
				t.Errorf("AutoMount mismatch: got %v, want %v", result.AutoMount, tt.expected.AutoMount)
			}
			if result.Pool != tt.expected.Pool {
				t.Errorf("Pool mismatch: got %v, want %v", result.Pool, tt.expected.Pool)
			}
		})
	}
}
