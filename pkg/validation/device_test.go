package validation

import "testing"

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
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				} else if tt.errContains != "" && !contains(err.Error(), tt.errContains) {
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
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				} else if tt.errContains != "" && !contains(err.Error(), tt.errContains) {
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

func TestValidateDevicePath(t *testing.T) {
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
			path:     "/dev/sda",
			isSource: true,
			wantErr:  false,
		},
		{
			name:        "relative path",
			path:        "dev/sda",
			isSource:    true,
			wantErr:     true,
			errContains: "must be absolute",
		},
		{
			name:        "path with ..",
			path:        "/dev/../sda",
			isSource:    true,
			wantErr:     true,
			errContains: "must not contain '..'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateDevicePath(tt.path, tt.isSource)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				} else if tt.errContains != "" && !contains(err.Error(), tt.errContains) {
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
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				} else if tt.errContains != "" && !contains(err.Error(), tt.errContains) {
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

func TestValidateDevice(t *testing.T) {
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
			source:      "/dev/sda",
			destination: "/dev/sda",
			options:     []string{"ro"},
			wantErr:     false,
		},
		{
			name:        "empty device name",
			deviceName:  "",
			deviceType:  "disk",
			source:      "/dev/sda",
			destination: "/dev/sda",
			wantErr:     true,
			errContains: "name is required",
		},
		{
			name:        "invalid device type",
			deviceName:  "sda",
			deviceType:  "invalid",
			source:      "/dev/sda",
			destination: "/dev/sda",
			wantErr:     true,
			errContains: "unsupported device type",
		},
		{
			name:        "empty source path",
			deviceName:  "sda",
			deviceType:  "disk",
			source:      "",
			destination: "/dev/sda",
			wantErr:     true,
			errContains: "source path is required",
		},
		{
			name:        "relative source path",
			deviceName:  "sda",
			deviceType:  "disk",
			source:      "dev/sda",
			destination: "/dev/sda",
			wantErr:     true,
			errContains: "must be absolute",
		},
		{
			name:        "invalid options",
			deviceName:  "sda",
			deviceType:  "disk",
			source:      "/dev/sda",
			destination: "/dev/sda",
			options:     []string{"invalid"},
			wantErr:     true,
			errContains: "invalid device option",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateDevice(tt.deviceName, tt.deviceType, tt.source, tt.destination, tt.options)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				} else if tt.errContains != "" && !contains(err.Error(), tt.errContains) {
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
