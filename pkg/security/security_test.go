package security_test

import (
	"os"
	"testing"

	"github.com/larkinwc/proxmox-lxc-compose/pkg/security"
)

func TestProfileValidation(t *testing.T) {
	// Save original PATH and restore after test
	originalPath := os.Getenv("PATH")
	defer os.Setenv("PATH", originalPath)

	// Set empty PATH to simulate environment without apparmor/selinux tools
	os.Setenv("PATH", "")

	tests := []struct {
		name    string
		profile *security.Profile
		wantErr bool
	}{
		{
			name: "valid default profile",
			profile: &security.Profile{
				Isolation: security.IsolationDefault,
			},
			wantErr: false,
		},
		{
			name: "valid strict profile",
			profile: &security.Profile{
				Isolation:    security.IsolationStrict,
				AppArmorName: "lxc-container-default",
				Capabilities: []string{"NET_ADMIN", "SYS_TIME"},
			},
			wantErr: false,
		},
		{
			name: "valid privileged profile",
			profile: &security.Profile{
				Isolation:  security.IsolationPrivileged,
				Privileged: true,
			},
			wantErr: false,
		},
		{
			name: "invalid isolation level",
			profile: &security.Profile{
				Isolation: "invalid",
			},
			wantErr: true,
		},
		{
			name: "strict with privileged",
			profile: &security.Profile{
				Isolation:  security.IsolationStrict,
				Privileged: true,
			},
			wantErr: true,
		},
		{
			name: "custom apparmor profile",
			profile: &security.Profile{
				Isolation:    security.IsolationStrict,
				AppArmorName: "custom-profile",
			},
			wantErr: false, // Should pass when apparmor tools not available
		},
		{
			name: "custom selinux context",
			profile: &security.Profile{
				Isolation:      security.IsolationStrict,
				SELinuxContext: "custom_u:custom_r:custom_t:s0",
			},
			wantErr: false, // Should pass when selinux tools not available
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.profile.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Profile.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestProfileApplication(t *testing.T) {
	// Save original PATH and restore after test
	originalPath := os.Getenv("PATH")
	defer os.Setenv("PATH", originalPath)

	// Set empty PATH to simulate environment without apparmor/selinux tools
	os.Setenv("PATH", "")

	tests := []struct {
		name          string
		profile       *security.Profile
		containerName string
		wantErr       bool
	}{
		{
			name: "apply default profile",
			profile: &security.Profile{
				Isolation: security.IsolationDefault,
			},
			containerName: "test-container",
			wantErr:       false,
		},
		{
			name: "apply strict profile",
			profile: &security.Profile{
				Isolation:      security.IsolationStrict,
				AppArmorName:   "lxc-container-default",
				SELinuxContext: "unconfined_u:unconfined_r:unconfined_t:s0",
			},
			containerName: "test-container",
			wantErr:       false,
		},
		{
			name: "apply privileged profile",
			profile: &security.Profile{
				Isolation:  security.IsolationPrivileged,
				Privileged: true,
			},
			containerName: "test-container",
			wantErr:       false,
		},
		{
			name: "apply with custom apparmor",
			profile: &security.Profile{
				Isolation:    security.IsolationStrict,
				AppArmorName: "custom-profile",
			},
			containerName: "test-container",
			wantErr:       false, // Should pass when apparmor tools not available
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.profile.Apply(tt.containerName)
			if (err != nil) != tt.wantErr {
				t.Errorf("Profile.Apply() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
