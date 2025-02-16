package validation

import (
	"path/filepath"
	"testing"

	testing_internal "proxmox-lxc-compose/pkg/internal/testing"
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
			errContains: "required",
		},
		{
			name:        "invalid type",
			deviceType:  "invalid",
			wantErr:     true,
			errContains: "unsupported device type",
		},
		{
			name:       "valid type - unix-char",
			deviceType: "unix-char",
			wantErr:    false,
		},
		{
			name:       "valid type - disk",
			deviceType: "disk",
			wantErr:    false,
		},
		{
			name:       "valid type - uppercase",
			deviceType: "DISK",
			wantErr:    false,
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
			errContains: "required",
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
			name:        "invalid start character",
			deviceName:  "_dev0",
			wantErr:     true,
			errContains: "must start with letter/number",
		},
		{
			name:        "invalid character",
			deviceName:  "dev@0",
			wantErr:     true,
			errContains: "invalid device name",
		},
		{
			name:        "too long",
			deviceName:  "a123456789012345678901234567890123456789012345678901234567890abcd",
			wantErr:     true,
			errContains: "too long",
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
		isSource    bool
		wantErr     bool
		errContains string
	}{
		{
			name:     "empty path for destination",
			path:     "",
			isSource: false,
			wantErr:  false,
		},
		{
			name:        "empty path for source",
			path:        "",
			isSource:    true,
			wantErr:     true,
			errContains: "source path is required",
		},
		{
			name:     "valid absolute path",
			path:     absPath,
			isSource: true,
			wantErr:  false,
		},
		{
			name:        "relative path",
			path:        relPath,
			isSource:    true,
			wantErr:     true,
			errContains: "must be absolute",
		},
		{
			name:        "path with ..",
			path:        dotPath,
			isSource:    true,
			wantErr:     true,
			errContains: "must not contain '..'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateDevicePath(tt.path, tt.isSource)
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
		deviceType  string
		options     []string
		wantErr     bool
		errContains string
	}{
		{
			name:       "empty options",
			deviceType: "disk",
			options:    nil,
			wantErr:    false,
		},
		{
			name:       "valid options",
			deviceType: "disk",
			options:    []string{"ro", "required"},
			wantErr:    false,
		},
		{
			name:        "invalid option",
			deviceType:  "disk",
			options:     []string{"invalid"},
			wantErr:     true,
			errContains: "invalid device option",
		},
		{
			name:        "conflicting options ro/rw",
			deviceType:  "disk",
			options:     []string{"ro", "rw"},
			wantErr:     true,
			errContains: "conflicting",
		},
		{
			name:        "conflicting options required/optional",
			deviceType:  "disk",
			options:     []string{"required", "optional"},
			wantErr:     true,
			errContains: "conflicting",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateDeviceOptions(tt.deviceType, tt.options)
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
		deviceName  string
		deviceType  string
		source      string
		destination string
		options     []string
		wantErr     bool
		errContains string
	}{
		{
			name:        "valid disk device",
			deviceName:  "sda",
			deviceType:  "disk",
			source:      absPath,
			destination: absPath,
			options:     []string{"ro"},
			wantErr:     false,
		},
		{
			name:        "empty device name",
			deviceName:  "",
			deviceType:  "disk",
			source:      absPath,
			destination: absPath,
			wantErr:     true,
			errContains: "name is required",
		},
		{
			name:        "invalid device type",
			deviceName:  "sda",
			deviceType:  "invalid",
			source:      absPath,
			destination: absPath,
			wantErr:     true,
			errContains: "unsupported device type",
		},
		{
			name:        "empty source path",
			deviceName:  "sda",
			deviceType:  "disk",
			source:      "",
			destination: absPath,
			wantErr:     true,
			errContains: "source path is required",
		},
		{
			name:        "relative source path",
			deviceName:  "sda",
			deviceType:  "disk",
			source:      relPath,
			destination: absPath,
			wantErr:     true,
			errContains: "must be absolute",
		},
		{
			name:        "invalid options",
			deviceName:  "sda",
			deviceType:  "disk",
			source:      absPath,
			destination: absPath,
			options:     []string{"invalid"},
			wantErr:     true,
			errContains: "invalid device option",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateDevice(tt.deviceName, tt.deviceType, tt.source, tt.destination, tt.options)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateDevice() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil && tt.errContains != "" && !testing_internal.Contains(err.Error(), tt.errContains) {
				t.Errorf("error %q does not contain %q", err.Error(), tt.errContains)
			}
		})
	}
}
