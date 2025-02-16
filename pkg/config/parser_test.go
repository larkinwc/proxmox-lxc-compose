package config_test

import (
	"path/filepath"
	"testing"

	"github.com/larkinwc/proxmox-lxc-compose/pkg/config"
	testing_internal "github.com/larkinwc/proxmox-lxc-compose/pkg/internal/testing"
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
			configPath: filepath.Join("testdata", "valid.yaml"),
			wantErr:    false,
		},
		{
			name:        "invalid config",
			configPath:  filepath.Join("testdata", "invalid.yaml"),
			wantErr:     true,
			errContains: "invalid configuration",
		},
		{
			name:        "non-existent file",
			configPath:  filepath.Join("testdata", "nonexistent.yaml"),
			wantErr:     true,
			errContains: "no such file",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := config.Load(tt.configPath)
			if tt.wantErr {
				testing_internal.AssertError(t, err)
				if tt.errContains != "" && !testing_internal.Contains(err.Error(), tt.errContains) {
					t.Errorf("error %q does not contain %q", err.Error(), tt.errContains)
				}
				return
			}
			testing_internal.AssertNoError(t, err)
			testing_internal.AssertNotNil(t, cfg)

			if tt.configPath == filepath.Join("testdata", "valid.yaml") {
				// Verify expected values from valid.yaml (app container)
				if cfg.Image != "ubuntu:20.04" {
					t.Errorf("expected image ubuntu:20.04, got %s", cfg.Image)
				}

				network := cfg.Network
				if network == nil {
					t.Fatal("expected network config, got nil")
				}
				if network.Type != "bridge" {
					t.Errorf("expected network type bridge, got %s", network.Type)
				}
				if network.IP != "10.0.3.100/24" {
					t.Errorf("expected IP 10.0.3.100/24, got %s", network.IP)
				}

				security := cfg.Security
				if security == nil {
					t.Fatal("expected security config, got nil")
				}
				if security.Isolation != "strict" {
					t.Errorf("expected isolation strict, got %s", security.Isolation)
				}
				if len(security.Capabilities) != 2 {
					t.Errorf("expected 2 capabilities, got %d", len(security.Capabilities))
				}

				storage := cfg.Storage
				if storage == nil {
					t.Fatal("expected storage config, got nil")
				}
				if storage.Root != "10G" {
					t.Errorf("expected root size 10G, got %s", storage.Root)
				}
			}
		})
	}
}
