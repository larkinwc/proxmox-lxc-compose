package security_test

import (
	"proxmox-lxc-compose/pkg/security"
	"testing"
)

func TestProfileValidation(t *testing.T) {
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
			name: "invalid isolation level",
			profile: &security.Profile{
				Isolation: "invalid",
			},
			wantErr: true,
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
