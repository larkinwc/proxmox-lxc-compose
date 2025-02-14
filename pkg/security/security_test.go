package security

import (
	"testing"
)

func TestSecurityProfileValidation(t *testing.T) {
	tests := []struct {
		name    string
		profile SecurityProfile
		wantErr bool
	}{
		{
			name: "valid default profile",
			profile: SecurityProfile{
				Isolation: IsolationDefault,
			},
			wantErr: false,
		},
		{
			name: "valid strict profile",
			profile: SecurityProfile{
				Isolation:       IsolationStrict,
				AppArmorProfile: "lxc-container-default",
				Capabilities:    []string{"NET_ADMIN", "SYS_TIME"},
			},
			wantErr: false,
		},
		{
			name: "invalid isolation level",
			profile: SecurityProfile{
				Isolation: "invalid",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.profile.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("SecurityProfile.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSecurityProfileApplication(t *testing.T) {
	tests := []struct {
		name          string
		profile       SecurityProfile
		containerName string
		wantErr       bool
	}{
		{
			name: "apply default profile",
			profile: SecurityProfile{
				Isolation: IsolationDefault,
			},
			containerName: "test-container",
			wantErr:       false,
		},
		{
			name: "apply strict profile",
			profile: SecurityProfile{
				Isolation:       IsolationStrict,
				AppArmorProfile: "lxc-container-default",
				SELinuxContext:  "unconfined_u:unconfined_r:unconfined_t:s0",
			},
			containerName: "test-container",
			wantErr:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.profile.Apply(tt.containerName)
			if (err != nil) != tt.wantErr {
				t.Errorf("SecurityProfile.Apply() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
