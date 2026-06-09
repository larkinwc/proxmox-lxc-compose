package main

import (
	"os"
	"path/filepath"
	"strings"
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

	proxmoxBackendFactory = func() (proxmox.Backend, error) { return fake, nil }
	vmidStorePath = filepath.Join(t.TempDir(), "vmids.json")

	cleanup := func() {
		proxmoxBackendFactory = origFactory
		vmidStorePath = origStorePath
		configFile = ""
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
	lxcConfigPath = tmpDir
	os.Setenv("CONTAINER_CONFIG_PATH", tmpDir)

	mockCmd, cleanupMock := testutil.SetupMockCommand(&container.ExecCommand)
	mockCmd.SetDebug(false)

	return func() {
		cleanupMock()
		lxcConfigPath = origPath
		os.Unsetenv("CONTAINER_CONFIG_PATH")
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

// stubConverter replaces ociConvertFn with a recording stub and points the
// template cache at a temp dir. It returns a pointer to the converter call
// count and a cleanup function.
func stubConverter(t *testing.T) (*int, func()) {
	t.Helper()
	origConvert := ociConvertFn
	origCacheDir := templateCacheDir
	templateCacheDir = t.TempDir()
	calls := 0
	ociConvertFn = func(_, outPath string) (*oci.ConvertResult, error) {
		calls++
		// Materialize the cache file so cache-reuse logic can find it.
		if err := os.WriteFile(outPath, []byte("template"), 0644); err != nil {
			return nil, err
		}
		return &oci.ConvertResult{OutputPath: outPath, InitWrapperPath: oci.InitWrapperPath}, nil
	}
	return &calls, func() {
		ociConvertFn = origConvert
		templateCacheDir = origCacheDir
	}
}

const nginxCompose = `version: "1.0"
services:
  web:
    image: nginx:alpine
    storage:
      root: 2G
`

func TestUpAutoConvertsOCIImage(t *testing.T) {
	fake, cleanup := setupBackendTest(t)
	defer cleanup()
	calls, restore := stubConverter(t)
	defer restore()

	// An OCI reference (no ":vztmpl/") is auto-detected and converted, with no
	// env flags required.
	configFile = writeComposeFile(t, nginxCompose)
	if err := upCmdRunE(nil, nil); err != nil {
		t.Fatalf("up failed: %v", err)
	}

	if *calls != 1 {
		t.Errorf("converter called %d times, want 1", *calls)
	}
	opts, ok := fake.created[100]
	if !ok {
		t.Fatal("expected container 100 to be created")
	}
	if !strings.HasPrefix(opts.OSTemplate, "local:vztmpl/") {
		t.Errorf("OSTemplate = %q, want local:vztmpl/ prefix", opts.OSTemplate)
	}
	if fake.initCmds[100] != oci.InitWrapperPath {
		t.Errorf("init cmd = %q, want %q", fake.initCmds[100], oci.InitWrapperPath)
	}
}

func TestUpUsesVolidWithoutConvert(t *testing.T) {
	fake, cleanup := setupBackendTest(t)
	defer cleanup()
	calls, restore := stubConverter(t)
	defer restore()

	// A Proxmox template volid is used verbatim; the converter is not called.
	configFile = writeComposeFile(t, `version: "1.0"
services:
  web:
    image: local:vztmpl/alpine-3.22.tar.xz
    storage:
      root: 2G
`)
	if err := upCmdRunE(nil, nil); err != nil {
		t.Fatalf("up failed: %v", err)
	}
	if *calls != 0 {
		t.Errorf("converter should not be called for a volid, got %d calls", *calls)
	}
	if opts := fake.created[100]; opts.OSTemplate != "local:vztmpl/alpine-3.22.tar.xz" {
		t.Errorf("OSTemplate = %q, want the volid", opts.OSTemplate)
	}
	if fake.initCmds[100] != "" {
		t.Errorf("init cmd should be empty, got %q", fake.initCmds[100])
	}
}

func TestUpReusesCachedTemplate(t *testing.T) {
	fake, cleanup := setupBackendTest(t)
	defer cleanup()
	calls, restore := stubConverter(t)
	defer restore()

	// Pre-populate the cache so conversion should be skipped.
	if err := os.WriteFile(ociTemplatePath("nginx:alpine"), []byte("cached"), 0644); err != nil {
		t.Fatal(err)
	}

	configFile = writeComposeFile(t, nginxCompose)
	if err := upCmdRunE(nil, nil); err != nil {
		t.Fatalf("up failed: %v", err)
	}
	if *calls != 0 {
		t.Errorf("converter should reuse cache, got %d calls", *calls)
	}
	if opts := fake.created[100]; !strings.HasPrefix(opts.OSTemplate, "local:vztmpl/oci-") {
		t.Errorf("OSTemplate = %q, want cached oci- template", opts.OSTemplate)
	}
	// The init command is still applied from the cached (baked-in) wrapper.
	if fake.initCmds[100] != oci.InitWrapperPath {
		t.Errorf("init cmd = %q, want %q", fake.initCmds[100], oci.InitWrapperPath)
	}
}

func TestUpForceConvertBypassesCache(t *testing.T) {
	_, cleanup := setupBackendTest(t)
	defer cleanup()
	calls, restore := stubConverter(t)
	defer restore()

	// Cache present, but --force-convert must re-run conversion.
	if err := os.WriteFile(ociTemplatePath("nginx:alpine"), []byte("cached"), 0644); err != nil {
		t.Fatal(err)
	}

	configFile = writeComposeFile(t, nginxCompose)
	cmd := upCmdForTest()
	if err := cmd.Flags().Set("force-convert", "true"); err != nil {
		t.Fatal(err)
	}
	if err := upCmdRunE(cmd, nil); err != nil {
		t.Fatalf("up failed: %v", err)
	}
	if *calls != 1 {
		t.Errorf("converter called %d times with --force-convert, want 1", *calls)
	}
}

// upCmdForTest builds a cobra command carrying the up flags so flag-dependent
// behavior can be exercised without the full root command.
func upCmdForTest() *cobra.Command {
	cmd := &cobra.Command{Use: "up"}
	cmd.Flags().Bool("force-convert", false, "")
	cmd.Flags().Bool("pull", false, "")
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
