package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/larkinwc/proxmox-lxc-compose/pkg/common"
	"github.com/larkinwc/proxmox-lxc-compose/pkg/container"
	"github.com/larkinwc/proxmox-lxc-compose/pkg/logging"
	"github.com/larkinwc/proxmox-lxc-compose/pkg/oci"
	"github.com/larkinwc/proxmox-lxc-compose/pkg/proxmox"
	"github.com/larkinwc/proxmox-lxc-compose/pkg/testutil"

	"github.com/spf13/cobra"
)

func init() {
	_ = logging.Init(logging.Config{Level: "error", Development: true})
}

var commonUbuntu = common.Container{Image: "ubuntu:20.04"}

// setupBackendTest installs an in-memory Proxmox backend and a temp VMID store
// so lifecycle commands never invoke the real pct binary.
func setupBackendTest(t *testing.T) (*fakeBackend, func()) {
	t.Helper()

	fake := newFakeBackend()
	origFactory := proxmoxBackendFactory
	origStorePath := vmidStorePath
	origConfigFile := configFile
	origConvertFn := ociConvertFn
	origCacheDir := templateCacheDir
	origReadiness := templateReadinessFn

	proxmoxBackendFactory = func() (proxmox.Backend, error) { return fake, nil }
	vmidStorePath = filepath.Join(t.TempDir(), "vmids.json")

	// Lifecycle tests use valid local archives, without Docker or host storage.
	templateCacheDir = t.TempDir()
	templateReadinessFn = func(string) error { return nil }
	ociConvertFn = func(image, _ string, _ oci.RuntimeOverrides) (*oci.ConvertResult, error) {
		runtime := oci.RuntimeConfig{Command: []string{"/bin/sh"}, Network: []oci.RuntimeNetwork{}}
		archive := seedImageRuntime(t, image, runtime)
		return &oci.ConvertResult{OutputPath: archive, Runtime: runtime, InitWrapperPath: oci.InitWrapperPath}, nil
	}

	cleanup := func() {
		proxmoxBackendFactory = origFactory
		vmidStorePath = origStorePath
		configFile = origConfigFile
		ociConvertFn = origConvertFn
		templateCacheDir = origCacheDir
		templateReadinessFn = origReadiness
	}
	return fake, cleanup
}

// setupManagerTest installs a mocked container.ExecCommand + temp LXC root for
// the template/manager-based commands.
func setupManagerTest(t *testing.T) func() {
	t.Helper()

	tmpDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(tmpDir, "state"), 0755); err != nil {
		t.Fatal(err)
	}
	origPath := lxcConfigPath
	origEnv, hadEnv := os.LookupEnv("CONTAINER_CONFIG_PATH")
	lxcConfigPath = tmpDir
	os.Setenv("CONTAINER_CONFIG_PATH", tmpDir)

	mockCmd, cleanupMock := testutil.SetupMockCommand(&container.ExecCommand)
	mockCmd.SetDebug(false)

	return func() {
		cleanupMock()
		lxcConfigPath = origPath
		if hadEnv {
			_ = os.Setenv("CONTAINER_CONFIG_PATH", origEnv)
		} else {
			_ = os.Unsetenv("CONTAINER_CONFIG_PATH")
		}
	}
}

func writeComposeFile(t *testing.T, contents string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "lxc-compose.yml")
	if err := os.WriteFile(path, []byte(contents), 0644); err != nil {
		t.Fatal(err)
	}
	return path
}

const multiServiceCompose = `version: "1.0"
services:
  web:
    image: ubuntu:20.04
    storage:
      root: 2G
  db:
    image: ubuntu:20.04
    storage:
      root: 2G
`

func TestUpCreatesAllServices(t *testing.T) {
	fake, cleanup := setupBackendTest(t)
	defer cleanup()

	configFile = writeComposeFile(t, multiServiceCompose)

	if err := upCmdRunE(nil, nil); err != nil {
		t.Fatalf("up failed: %v", err)
	}

	// Both services should be created and running (VMIDs 100 and 101).
	infos, _ := fake.List()
	if len(infos) != 2 {
		t.Fatalf("expected 2 containers, got %d", len(infos))
	}
	for _, info := range infos {
		if info.Status != proxmox.StatusRunning {
			t.Errorf("vmid %d status = %v, want running", info.VMID, info.Status)
		}
	}
}

func TestUpAssignsStableVMIDs(t *testing.T) {
	fake, cleanup := setupBackendTest(t)
	defer cleanup()

	configFile = writeComposeFile(t, multiServiceCompose)
	if err := upCmdRunE(nil, []string{"web"}); err != nil {
		t.Fatalf("up web failed: %v", err)
	}

	store, _ := newVMIDStore()
	vmid, ok := store.Get("web")
	if !ok || vmid != 100 {
		t.Errorf("web vmid = (%d, %v), want (100, true)", vmid, ok)
	}
	if _, err := fake.Status(100); err != nil {
		t.Errorf("expected vmid 100 to exist: %v", err)
	}
}

func TestUpSingleNamedService(t *testing.T) {
	fake, cleanup := setupBackendTest(t)
	defer cleanup()

	configFile = writeComposeFile(t, multiServiceCompose)
	if err := upCmdRunE(nil, []string{"web"}); err != nil {
		t.Fatalf("up web failed: %v", err)
	}

	infos, _ := fake.List()
	if len(infos) != 1 {
		t.Fatalf("expected 1 container, got %d", len(infos))
	}
}

func TestUpUnknownServiceErrors(t *testing.T) {
	_, cleanup := setupBackendTest(t)
	defer cleanup()

	configFile = writeComposeFile(t, multiServiceCompose)
	if err := upCmdRunE(nil, []string{"nope"}); err == nil {
		t.Fatal("expected error for unknown service, got nil")
	}
}

func TestDownStopsService(t *testing.T) {
	fake, cleanup := setupBackendTest(t)
	defer cleanup()

	configFile = writeComposeFile(t, multiServiceCompose)
	if err := upCmdRunE(nil, []string{"web"}); err != nil {
		t.Fatalf("up failed: %v", err)
	}
	if err := downCmdRunE(nil, []string{"web"}); err != nil {
		t.Fatalf("down failed: %v", err)
	}

	st, err := fake.Status(100)
	if err != nil {
		t.Fatalf("status failed: %v", err)
	}
	if st != proxmox.StatusStopped {
		t.Errorf("expected stopped, got %v", st)
	}
}

func TestDownRemoveDestroysAndUnmaps(t *testing.T) {
	fake, cleanup := setupBackendTest(t)
	defer cleanup()

	configFile = writeComposeFile(t, multiServiceCompose)
	if err := upCmdRunE(nil, []string{"web"}); err != nil {
		t.Fatalf("up failed: %v", err)
	}

	removeContainers = true
	defer func() { removeContainers = false }()
	if err := downCmdRunE(nil, []string{"web"}); err != nil {
		t.Fatalf("down --rm failed: %v", err)
	}

	if _, err := fake.Status(100); err == nil {
		t.Error("expected vmid 100 to be destroyed")
	}
	store, _ := newVMIDStore()
	if _, ok := store.Get("web"); ok {
		t.Error("expected web mapping to be removed")
	}
	state, err := loadDeploymentState()
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := state.services["web"]; ok {
		t.Error("expected web desired state to be removed")
	}
}

func TestPauseResumeService(t *testing.T) {
	fake, cleanup := setupBackendTest(t)
	defer cleanup()

	configFile = writeComposeFile(t, multiServiceCompose)
	if err := upCmdRunE(nil, []string{"web"}); err != nil {
		t.Fatalf("up failed: %v", err)
	}

	if err := pauseCmdRunE(nil, []string{"web"}); err != nil {
		t.Fatalf("pause failed: %v", err)
	}
	if st, _ := fake.Status(100); st != proxmox.StatusPaused {
		t.Errorf("expected paused, got %v", st)
	}

	if err := unpauseCmdRunE(nil, []string{"web"}); err != nil {
		t.Fatalf("unpause failed: %v", err)
	}
	if st, _ := fake.Status(100); st != proxmox.StatusRunning {
		t.Errorf("expected running, got %v", st)
	}
}

const nginxCompose = `version: "1.0"
services:
  web:
    image: nginx:alpine
    storage:
      root: 2G
`

// upCmdForTest builds a cobra command carrying the up flags so flag-dependent
// behavior can be exercised without the full root command.
func upCmdForTest() *cobra.Command {
	cmd := &cobra.Command{Use: "up"}
	cmd.Flags().Bool("force-convert", false, "")
	cmd.Flags().Bool("pull", false, "")
	cmd.Flags().Bool("recreate", false, "")
	return cmd
}

func TestTemplateLifecycle(t *testing.T) {
	cleanup := setupManagerTest(t)
	defer cleanup()

	// Create a container directly via the manager (templates operate on the
	// local LXCManager state, not the Proxmox backend).
	manager, err := newManager()
	if err != nil {
		t.Fatal(err)
	}
	if err := manager.Create("web", &commonUbuntu); err != nil {
		t.Fatalf("create failed: %v", err)
	}

	createCmd := templateCreateCmd()
	if err := createCmd.RunE(createCmd, []string{"web", "web-tmpl"}); err != nil {
		t.Fatalf("template create failed: %v", err)
	}

	templates, err := manager.ListTemplates()
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, tmpl := range templates {
		if tmpl.Name == "web-tmpl" {
			found = true
		}
	}
	if !found {
		t.Fatal("expected template 'web-tmpl' to be listed")
	}

	applyCmd := templateApplyCmd()
	if err := applyCmd.RunE(applyCmd, []string{"web-tmpl", "web2"}); err != nil {
		t.Fatalf("template apply failed: %v", err)
	}
	if !manager.ContainerExists("web2") {
		t.Error("expected container 'web2' from template")
	}

	delCmd := templateDeleteCmd()
	if err := delCmd.RunE(delCmd, []string{"web-tmpl"}); err != nil {
		t.Fatalf("template rm failed: %v", err)
	}
	if _, err := manager.GetTemplate("web-tmpl"); err == nil {
		t.Error("expected template to be deleted")
	}
	if err := delCmd.RunE(delCmd, []string{"web-tmpl"}); err == nil {
		t.Error("expected error deleting non-existent template")
	}
}
