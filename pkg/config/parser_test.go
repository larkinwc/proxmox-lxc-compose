package config

import (
	"testing"

	"proxmox-lxc-compose/pkg/testutil"
)

func TestLoadConfig(t *testing.T) {
	t.Run("loads valid configuration", func(t *testing.T) {
		dir, cleanup := testutil.TempDir(t)
		defer cleanup()

		configContent := `
version: "1.0"
services:
  web:
    image: ubuntu:20.04
    cpu:
      cores: 2
      shares: 1024
    memory:
      limit: 2G
      swap: 1G
    network:
      type: bridge
      bridge: vmbr0
      ip: 192.168.1.100
    storage:
      root: 10G
      mounts:
        - source: /path/to/data
          target: /data
    environment:
      DB_HOST: db
      DB_PORT: "5432"
    command: ["nginx", "-g", "daemon off;"]`

		configPath := testutil.WriteFile(t, dir, "lxc-compose.yml", configContent)

		cfg, err := LoadConfig(configPath)
		testutil.AssertNoError(t, err)

		// Verify version
		testutil.AssertEqual(t, "1.0", cfg.Version)

		// Verify web service
		web, ok := cfg.Services["web"]
		if !ok {
			t.Fatal("web service not found")
		}

		testutil.AssertEqual(t, "ubuntu:20.04", web.Image)

		// Verify CPU config
		if web.CPU == nil {
			t.Fatal("CPU config is nil")
		}
		cores := 2
		testutil.AssertEqual(t, &cores, web.CPU.Cores)
		shares := int64(1024)
		testutil.AssertEqual(t, &shares, web.CPU.Shares)

		// Verify memory config
		if web.Memory == nil {
			t.Fatal("Memory config is nil")
		}
		testutil.AssertEqual(t, "2G", web.Memory.Limit)
		testutil.AssertEqual(t, "1G", web.Memory.Swap)

		// Verify network config
		if web.Network == nil {
			t.Fatal("Network config is nil")
		}
		testutil.AssertEqual(t, "bridge", web.Network.Type)
		testutil.AssertEqual(t, "vmbr0", web.Network.Bridge)
		testutil.AssertEqual(t, "192.168.1.100", web.Network.IP)

		// Verify storage config
		if web.Storage == nil {
			t.Fatal("Storage config is nil")
		}
		testutil.AssertEqual(t, "10G", web.Storage.Root)
		if len(web.Storage.Mounts) != 1 {
			t.Fatal("expected 1 mount")
		}
		testutil.AssertEqual(t, "/path/to/data", web.Storage.Mounts[0].Source)
		testutil.AssertEqual(t, "/data", web.Storage.Mounts[0].Target)

		// Verify environment
		testutil.AssertEqual(t, "db", web.Environment["DB_HOST"])
		testutil.AssertEqual(t, "5432", web.Environment["DB_PORT"])

		// Verify command
		if len(web.Command) != 3 {
			t.Fatal("expected 3 command parts")
		}
		testutil.AssertEqual(t, "nginx", web.Command[0])
		testutil.AssertEqual(t, "-g", web.Command[1])
		testutil.AssertEqual(t, "daemon off;", web.Command[2])
	})

	t.Run("validates required fields", func(t *testing.T) {
		dir, cleanup := testutil.TempDir(t)
		defer cleanup()

		// Missing version
		configContent := `
services:
  web:
    image: ubuntu:20.04`

		configPath := testutil.WriteFile(t, dir, "lxc-compose.yml", configContent)
		_, err := LoadConfig(configPath)
		testutil.AssertError(t, err)

		// Missing services
		configContent = `version: "1.0"`
		configPath = testutil.WriteFile(t, dir, "lxc-compose.yml", configContent)
		_, err = LoadConfig(configPath)
		testutil.AssertError(t, err)

		// Missing image
		configContent = `
version: "1.0"
services:
  web: {}`
		configPath = testutil.WriteFile(t, dir, "lxc-compose.yml", configContent)
		_, err = LoadConfig(configPath)
		testutil.AssertError(t, err)
	})

	t.Run("handles invalid YAML", func(t *testing.T) {
		dir, cleanup := testutil.TempDir(t)
		defer cleanup()

		configContent := `invalid: [yaml`
		configPath := testutil.WriteFile(t, dir, "lxc-compose.yml", configContent)
		_, err := LoadConfig(configPath)
		testutil.AssertError(t, err)
	})

	t.Run("handles non-existent file", func(t *testing.T) {
		_, err := LoadConfig("nonexistent.yml")
		testutil.AssertError(t, err)
	})
}
