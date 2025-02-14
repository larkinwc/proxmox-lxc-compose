package config_test

import (
	"proxmox-lxc-compose/pkg/config"
	testing_internal "proxmox-lxc-compose/pkg/internal/testing"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	tests := []struct {
		name        string
		configPath  string
		wantErr     bool
		errContains string
	}{
		{
			name:       "valid config",
			configPath: "testdata/valid.yaml",
		},
		{
			name:        "invalid config",
			configPath:  "testdata/invalid.yaml",
			wantErr:     true,
			errContains: "invalid configuration",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := config.Load(tt.configPath)
			if tt.wantErr {
				testing_internal.AssertError(t, err)
				testing_internal.AssertContains(t, err.Error(), tt.errContains)
			} else {
				testing_internal.AssertNoError(t, err)
				testing_internal.AssertNotNil(t, cfg)
			}
		})
	}
}

// Other test functions...
