package container

import (
	"os"
	"path/filepath"
	"testing"

	"proxmox-lxc-compose/pkg/config"
	. "proxmox-lxc-compose/pkg/internal/testing"
)

func TestApplySecurityConfig(t *testing.T) {
	dir, cleanup := TempDir(t)
	defer cleanup()

	// Create test container directory
	containerName := "test-container"
	containerDir := filepath.Join(dir, containerName)
	err := os.MkdirAll(containerDir, 0755)
	AssertNoError(t, err)

	// Create manager
	manager := &LXCManager{
		configPath: dir,
	}

	tests := []struct {
		name   string
		config *config.SecurityConfig
		verify func(t *testing.T, configPath string)
	}{
		{
			name:   "default security config",
			config: nil,
			verify: func(t *testing.T, configPath string) {
				data, err := os.ReadFile(configPath)
				AssertNoError(t, err)
				content := string(data)
				AssertContains(t, content, "lxc.apparmor.profile = lxc-container-default")
			},
		},
		{
			name: "privileged config",
			config: &config.SecurityConfig{
				Isolation:  "privileged",
				Privileged: true,
			},
			verify: func(t *testing.T, configPath string) {
				data, err := os.ReadFile(configPath)
				AssertNoError(t, err)
				content := string(data)
				AssertContains(t, content, "lxc.apparmor.profile = unconfined")
				AssertContains(t, content, "lxc.cap.drop = ")
			},
		},
		{
			name: "strict isolation with capabilities",
			config: &config.SecurityConfig{
				Isolation:       "strict",
				AppArmorProfile: "lxc-container-default-restricted",
				Capabilities:    []string{"NET_ADMIN", "SYS_TIME"},
			},
			verify: func(t *testing.T, configPath string) {
				data, err := os.ReadFile(configPath)
				AssertNoError(t, err)
				content := string(data)
				AssertContains(t, content, "lxc.apparmor.profile = lxc-container-default-restricted")
				AssertContains(t, content, "lxc.cap.drop = all")
				AssertContains(t, content, "lxc.cap.keep = NET_ADMIN SYS_TIME")
			},
		},
		{
			name: "selinux context",
			config: &config.SecurityConfig{
				Isolation:      "default",
				SELinuxContext: "system_u:system_r:container_t:s0",
			},
			verify: func(t *testing.T, configPath string) {
				data, err := os.ReadFile(configPath)
				AssertNoError(t, err)
				content := string(data)
				AssertContains(t, content, "lxc.selinux.context = system_u:system_r:container_t:s0")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			configPath := filepath.Join(containerDir, "config")
			os.Remove(configPath) // Clean up from previous test

			f, err := os.Create(configPath)
			AssertNoError(t, err)
			defer f.Close()

			err = manager.applySecurityConfig(f, tt.config)
			AssertNoError(t, err)

			if tt.verify != nil {
				tt.verify(t, configPath)
			}
		})
	}
}
