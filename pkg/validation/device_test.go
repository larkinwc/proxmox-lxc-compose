package validation

import (
	"path/filepath"
	"testing"

	"github.com/larkinwc/proxmox-lxc-compose/pkg/config"
	testing_internal "github.com/larkinwc/proxmox-lxc-compose/pkg/internal/testing"
)

func TestValidateDeviceType(t *testing.T) {
	tests := []struct {
		name        string
		deviceType  string
		wantErr     bool
		errContains string
	}{
		{
			name:        "empty type",
			deviceType:  "",
			wantErr:     true,
			errContains: "invalid device type",
		},
		{
			name:        "invalid type",
			deviceType:  "invalid",
			wantErr:     true,
			errContains: "invalid device type",
		},
		{
			name:        "unix-char not supported",
			deviceType:  "unix-char",
			wantErr:     true,
			errContains: "invalid device type",
		},
		{
			name:       "valid type - disk",
			deviceType: "disk",
			wantErr:    false,
		},
		{
			name:        "uppercase not supported",
			deviceType:  "DISK",
			wantErr:     true,
			errContains: "invalid device type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateDeviceType(tt.deviceType)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateDeviceType() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil && tt.errContains != "" && !testing_internal.Contains(err.Error(), tt.errContains) {
				t.Errorf("error %q does not contain %q", err.Error(), tt.errContains)
			}
		})
	}
}

func TestValidateDeviceName(t *testing.T) {
	tests := []struct {
		name        string
		deviceName  string
		wantErr     bool
		errContains string
	}{
		{
			name:        "empty name",
			deviceName:  "",
			wantErr:     true,
			errContains: "cannot be empty",
		},
		{
			name:       "valid name",
			deviceName: "dev0",
			wantErr:    false,
		},
		{
			name:       "valid name with hyphen",
			deviceName: "dev-0",
			wantErr:    false,
		},
		{
			name:       "valid name with underscore",
			deviceName: "dev_0",
			wantErr:    false,
		},
		{
			name:       "underscore start is valid",
			deviceName: "_dev0",
			wantErr:    false,
		},
		{
			name:        "invalid character",
			deviceName:  "dev@0",
			wantErr:     true,
			errContains: "invalid device name",
		},
		{
			name:       "long name is fine",
			deviceName: "a123456789012345678901234567890123456789012345678901234567890abcd",
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateDeviceName(tt.deviceName)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateDeviceName() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil && tt.errContains != "" && !testing_internal.Contains(err.Error(), tt.errContains) {
				t.Errorf("error %q does not contain %q", err.Error(), tt.errContains)
			}
		})
	}
}

func TestValidateDevicePath(t *testing.T) {
	// Create platform-agnostic paths
	absPath := filepath.Join(string(filepath.Separator), "dev", "sda")
	relPath := filepath.Join("dev", "sda")
	// Use raw string concatenation to prevent normalization
	dotPath := string(filepath.Separator) + "dev" + string(filepath.Separator) + ".." + string(filepath.Separator) + "sda"

	tests := []struct {
		name        string
		path        string
		wantErr     bool
		errContains string
	}{
		{
			name:        "empty path",
			path:        "",
			wantErr:     true,
			errContains: "cannot be empty",
		},
		{
			name:    "valid absolute path",
			path:    absPath,
			wantErr: false,
		},
		{
			name:        "relative path",
			path:        relPath,
			wantErr:     true,
			errContains: "must be absolute",
		},
		{
			name:    "path with .. is allowed by current implementation",
			path:    dotPath,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateDevicePath(tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateDevicePath() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil && tt.errContains != "" && !testing_internal.Contains(err.Error(), tt.errContains) {
				t.Errorf("error %q does not contain %q", err.Error(), tt.errContains)
			}
		})
	}
}

func TestValidateDeviceOptions(t *testing.T) {
	tests := []struct {
		name        string
		options     []string
		wantErr     bool
		errContains string
	}{
		{
			name:    "empty options",
			options: nil,
			wantErr: false,
		},
		{
			name:    "valid options",
			options: []string{"ro", "required"},
			wantErr: false,
		},
		{
			name:        "empty option in list",
			options:     []string{"ro", ""},
			wantErr:     true,
			errContains: "cannot be empty",
		},
		{
			name:    "all options are valid in current implementation",
			options: []string{"ro", "rw"},
			wantErr: false,
		},
		{
			name:    "conflicting options allowed in current implementation",
			options: []string{"required", "optional"},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateDeviceOptions(tt.options)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateDeviceOptions() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil && tt.errContains != "" && !testing_internal.Contains(err.Error(), tt.errContains) {
				t.Errorf("error %q does not contain %q", err.Error(), tt.errContains)
			}
		})
	}
}

func TestValidateDevice(t *testing.T) {
	// Create platform-agnostic paths
	absPath := filepath.Join(string(filepath.Separator), "dev", "sda")
	relPath := filepath.Join("dev", "sda")

	tests := []struct {
		name        string
		device      *config.DeviceConfig
		wantErr     bool
		errContains string
	}{
		{
			name: "valid disk device",
			device: &config.DeviceConfig{
				Name:        "disk0",
				Type:        "disk",
				Source:      absPath,
				Destination: absPath,
				Options:     []string{"ro"},
			},
			wantErr: false,
		},
		{
			name: "empty device name",
			device: &config.DeviceConfig{
				Name: "",
				Type: "disk",
			},
			wantErr:     true,
			errContains: "cannot be empty",
		},
		{
			name: "invalid device type",
			device: &config.DeviceConfig{
				Name: "test",
				Type: "invalid",
			},
			wantErr:     true,
			errContains: "invalid device type",
		},
		{
			name: "source path can be empty",
			device: &config.DeviceConfig{
				Name: "test",
				Type: "disk",
			},
			wantErr: false,
		},
		{
			name: "relative source path",
			device: &config.DeviceConfig{
				Name:   "test",
				Type:   "disk",
				Source: relPath,
			},
			wantErr:     true,
			errContains: "must be absolute",
		},
		{
			name: "empty option in list",
			device: &config.DeviceConfig{
				Name:    "test",
				Type:    "disk",
				Options: []string{"ro", ""},
			},
			wantErr:     true,
			errContains: "cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateDevice(tt.device)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateDevice() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil && tt.errContains != "" && !testing_internal.Contains(err.Error(), tt.errContains) {
				t.Errorf("error %q does not contain %q", err.Error(), tt.errContains)
			}
		})
	}
}
