package main

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/larkinwc/proxmox-lxc-compose/pkg/oci"
	"github.com/larkinwc/proxmox-lxc-compose/pkg/proxmox"
	"github.com/spf13/cobra"
)

// guardedLifecycleBackend rejects transitions which must not occur and injects
// provisioning failures without prescribing pct commands or their ordering.
type guardedLifecycleBackend struct {
	*fakeBackend
	blockCreate  bool
	blockStart   bool
	blockStop    bool
	blockDestroy bool
}

func (b *guardedLifecycleBackend) Create(id int, opts proxmox.CreateOptions) error {
	if b.blockCreate {
		return fmt.Errorf("create rejected")
	}
	return b.fakeBackend.Create(id, opts)
}
func (b *guardedLifecycleBackend) Start(id int) error {
	if b.blockStart {
		return fmt.Errorf("start rejected")
	}
	return b.fakeBackend.Start(id)
}
func (b *guardedLifecycleBackend) Stop(id int) error {
	if b.blockStop {
		return fmt.Errorf("stop rejected")
	}
	return b.fakeBackend.Stop(id)
}
func (b *guardedLifecycleBackend) Shutdown(id int) error { return b.Stop(id) }
func (b *guardedLifecycleBackend) Destroy(id int) error {
	if b.blockDestroy {
		return fmt.Errorf("destroy rejected")
	}
	return b.fakeBackend.Destroy(id)
}

func provisionWeb(t *testing.T) (*fakeBackend, deploymentRecord) {
	t.Helper()
	fake, cleanup := setupBackendTest(t)
	t.Cleanup(cleanup)
	configFile = writeComposeFile(t, nginxCompose)
	if err := upCmdRunE(nil, []string{"web"}); err != nil {
		t.Fatal(err)
	}
	state, err := loadDeploymentState()
	if err != nil {
		t.Fatal(err)
	}
	record, ok := state.services["web"]
	if !ok || record.Digest == "" {
		t.Fatal("successful provisioning did not save desired state")
	}
	return fake, record
}

func recreationCommand(t *testing.T) *cobra.Command {
	t.Helper()
	cmd := upCmdForTest()
	if err := cmd.Flags().Set("recreate", "true"); err != nil {
		t.Fatal(err)
	}
	return cmd
}

func assertDeploymentRecord(t *testing.T, want deploymentRecord) {
	t.Helper()
	state, err := loadDeploymentState()
	if err != nil {
		t.Fatal(err)
	}
	if got := state.services["web"]; !reflect.DeepEqual(got, want) {
		t.Fatalf("deployment state = %+v, want %+v", got, want)
	}
}

func TestRepeatUpRetainsRunningAndPausedContainers(t *testing.T) {
	for _, status := range []proxmox.Status{proxmox.StatusRunning, proxmox.StatusPaused} {
		t.Run(string(status), func(t *testing.T) {
			fake, record := provisionWeb(t)
			fake.status[record.VMID] = status
			fake.initCmds[record.VMID] = "existing-init"
			guard := &guardedLifecycleBackend{fakeBackend: fake, blockCreate: true, blockStart: true, blockStop: true, blockDestroy: true}
			proxmoxBackendFactory = func() (proxmox.Backend, error) { return guard, nil }
			ociConvertFn = func(string, string, oci.RuntimeOverrides) (*oci.ConvertResult, error) {
				return nil, fmt.Errorf("repeat up must not resolve image again")
			}
			if err := upCmdRunE(nil, []string{"web"}); err != nil {
				t.Fatal(err)
			}
			if fake.status[record.VMID] != status || fake.initCmds[record.VMID] != "existing-init" {
				t.Fatal("existing container was modified")
			}
			assertDeploymentRecord(t, record)
		})
	}
}

func TestRepeatUpStartsStoppedContainerWithoutReprovisioning(t *testing.T) {
	fake, record := provisionWeb(t)
	fake.status[record.VMID] = proxmox.StatusStopped
	fake.initCmds[record.VMID] = "existing-init"
	guard := &guardedLifecycleBackend{fakeBackend: fake, blockCreate: true, blockStop: true, blockDestroy: true}
	proxmoxBackendFactory = func() (proxmox.Backend, error) { return guard, nil }
	if err := upCmdRunE(nil, []string{"web"}); err != nil {
		t.Fatal(err)
	}
	if fake.status[record.VMID] != proxmox.StatusRunning || fake.initCmds[record.VMID] != "existing-init" {
		t.Fatal("stopped container was not resumed unchanged")
	}
	assertDeploymentRecord(t, record)
}

func TestRepeatUpRecreatesMissingContainerAtMappedVMID(t *testing.T) {
	fake, record := provisionWeb(t)
	if err := fake.Destroy(record.VMID); err != nil {
		t.Fatal(err)
	}
	// An unrelated lower ID must not change the stable mapping.
	if err := fake.Create(99, proxmox.CreateOptions{Hostname: "unrelated"}); err != nil {
		t.Fatal(err)
	}
	if err := upCmdRunE(nil, []string{"web"}); err != nil {
		t.Fatal(err)
	}
	if fake.status[record.VMID] != proxmox.StatusRunning || fake.names[record.VMID] != "web" {
		t.Fatal("missing service was not provisioned at its original VMID")
	}
	if fake.names[99] != "unrelated" {
		t.Fatal("unrelated container modified")
	}
	assertDeploymentRecord(t, record)
}

func TestChangedDesiredConfigRequiresExplicitRecreation(t *testing.T) {
	for _, suffix := range []string{
		"    memory:\n      limit: 512M\n",
		"    volumes: [\"/host:/guest\"]\n",
		"    env: {TOKEN: changed}\n",
		"    environment: {TOKEN: changed}\n",
		"    command: []\n",
		"    entrypoint: []\n",
		"    security:\n      isolation: default\n      apparmor_profile: custom\n",
	} {
		t.Run(strings.ReplaceAll(suffix, "\n", ""), func(t *testing.T) {
			fake, record := provisionWeb(t)
			before := fake.created[record.VMID]
			configFile = writeComposeFile(t, nginxCompose+suffix)
			err := upCmdRunE(nil, []string{"web"})
			if err == nil || !strings.Contains(err.Error(), "--recreate") {
				t.Fatalf("expected recreation instruction, got %v", err)
			}
			if fake.status[record.VMID] != proxmox.StatusRunning || !reflect.DeepEqual(before, fake.created[record.VMID]) {
				t.Fatal("config drift modified existing service")
			}
			assertDeploymentRecord(t, record)
		})
	}
}

func TestEffectiveNodeDefaultChangeRequiresRecreation(t *testing.T) {
	for _, key := range []string{"PROXMOX_STORAGE", "PROXMOX_BRIDGE"} {
		t.Run(key, func(t *testing.T) {
			fake, record := provisionWeb(t)
			t.Setenv(key, "different-node-default")
			err := upCmdRunE(nil, []string{"web"})
			if err == nil || !strings.Contains(err.Error(), "--recreate") {
				t.Fatalf("expected recreation instruction, got %v", err)
			}
			if fake.status[record.VMID] != proxmox.StatusRunning {
				t.Fatal("node default drift stopped container")
			}
			assertDeploymentRecord(t, record)
		})
	}
}

func TestCanonicalServiceConfigDoesNotDependOnMapOrder(t *testing.T) {
	fake, cleanup := setupBackendTest(t)
	defer cleanup()
	configFile = writeComposeFile(t, nginxCompose+"    env: {FIRST: a, SECOND: b}\n")
	if err := upCmdRunE(nil, []string{"web"}); err != nil {
		t.Fatal(err)
	}
	configFile = writeComposeFile(t, nginxCompose+"    env: {SECOND: b, FIRST: a}\n")
	guard := &guardedLifecycleBackend{fakeBackend: fake, blockCreate: true, blockStart: true}
	proxmoxBackendFactory = func() (proxmox.Backend, error) { return guard, nil }
	if err := upCmdRunE(nil, []string{"web"}); err != nil {
		t.Fatal(err)
	}
}

func TestLegacyMappingRequiresRecreationEvenWhenContainerMissing(t *testing.T) {
	for _, present := range []bool{false, true} {
		t.Run(fmt.Sprint(present), func(t *testing.T) {
			fake, record := provisionWeb(t)
			if err := os.Remove(vmidStorePath + ".state.json"); err != nil {
				t.Fatal(err)
			}
			if !present {
				if err := fake.Destroy(record.VMID); err != nil {
					t.Fatal(err)
				}
			}
			err := upCmdRunE(nil, []string{"web"})
			if err == nil || !strings.Contains(err.Error(), "--recreate") {
				t.Fatalf("expected legacy recreation instruction, got %v", err)
			}
			if present && fake.status[record.VMID] != proxmox.StatusRunning {
				t.Fatal("legacy container modified without authorization")
			}
			if err := upCmdRunE(recreationCommand(t), []string{"web"}); err != nil {
				t.Fatal(err)
			}
			if fake.status[record.VMID] != proxmox.StatusRunning {
				t.Fatal("legacy recreation did not retain VMID")
			}
			assertDeploymentRecord(t, record)
		})
	}
}

func TestRecreateAppliesChangedConfigAndKeepsVMID(t *testing.T) {
	fake, before := provisionWeb(t)
	fake.initCmds[before.VMID] = "old-init"
	configFile = writeComposeFile(t, nginxCompose+"    memory:\n      limit: 512M\n")
	if err := upCmdRunE(recreationCommand(t), []string{"web"}); err != nil {
		t.Fatal(err)
	}
	if fake.created[before.VMID].MemoryMB != 512 || fake.status[before.VMID] != proxmox.StatusRunning || fake.initCmds[before.VMID] == "old-init" {
		t.Fatal("recreation did not replace the container at its original VMID")
	}
	state, err := loadDeploymentState()
	if err != nil {
		t.Fatal(err)
	}
	if state.services["web"].VMID != before.VMID || state.services["web"].Digest == before.Digest {
		t.Fatal("recreation did not commit updated desired config at stable VMID")
	}
	if err := upCmdRunE(nil, []string{"web"}); err != nil {
		t.Fatalf("repeat up after recreation: %v", err)
	}
}

func TestImageRefreshOnExistingContainerRequiresRecreation(t *testing.T) {
	for _, flag := range []string{"pull", "force-convert"} {
		t.Run(flag, func(t *testing.T) {
			fake, record := provisionWeb(t)
			cmd := upCmdForTest()
			if err := cmd.Flags().Set(flag, "true"); err != nil {
				t.Fatal(err)
			}
			err := upCmdRunE(cmd, []string{"web"})
			if err == nil || !strings.Contains(err.Error(), "--recreate") {
				t.Fatalf("expected recreation requirement, got %v", err)
			}
			if fake.status[record.VMID] != proxmox.StatusRunning {
				t.Fatal("refresh stopped existing container")
			}
			assertDeploymentRecord(t, record)
		})
	}
}

func TestRecreateValidationFailurePreservesExistingDeployment(t *testing.T) {
	for _, failure := range []string{"config", "template", "stop", "destroy"} {
		t.Run(failure, func(t *testing.T) {
			fake, record := provisionWeb(t)
			guard := &guardedLifecycleBackend{fakeBackend: fake}
			proxmoxBackendFactory = func() (proxmox.Backend, error) { return guard, nil }
			cmd := recreationCommand(t)
			switch failure {
			case "config":
				configFile = writeComposeFile(t, nginxCompose+"    memory:\n      limit: invalid\n")
			case "template":
				if err := cmd.Flags().Set("pull", "true"); err != nil {
					t.Fatal(err)
				}
				ociConvertFn = func(string, string, oci.RuntimeOverrides) (*oci.ConvertResult, error) {
					return nil, fmt.Errorf("template unavailable")
				}
			case "stop":
				guard.blockStop = true
			case "destroy":
				guard.blockDestroy = true
			}
			if err := upCmdRunE(cmd, []string{"web"}); err == nil {
				t.Fatal("expected recreation failure")
			}
			if _, exists := fake.created[record.VMID]; !exists {
				t.Fatal("failure destroyed original container")
			}
			if failure != "destroy" && fake.status[record.VMID] != proxmox.StatusRunning {
				t.Fatal("validation/stop failure changed runtime state")
			}
			assertDeploymentRecord(t, record)
		})
	}
}

func TestProvisioningFailureDoesNotRecordSuccessfulDesiredState(t *testing.T) {
	for _, failure := range []string{"create", "init", "start"} {
		t.Run(failure, func(t *testing.T) {
			fake, record := provisionWeb(t)
			guard := &guardedLifecycleBackend{fakeBackend: fake, blockCreate: failure == "create", blockStart: failure == "start"}
			var backend proxmox.Backend = guard
			if failure == "init" {
				backend = &failingInitBackend{guard}
			}
			proxmoxBackendFactory = func() (proxmox.Backend, error) { return backend, nil }
			if err := upCmdRunE(recreationCommand(t), []string{"web"}); err == nil {
				t.Fatal("expected provisioning failure")
			}
			state, err := loadDeploymentState()
			if err != nil {
				t.Fatal(err)
			}
			if _, recorded := state.services["web"]; recorded {
				t.Fatal("failed replacement inherited successful desired state")
			}
			store, err := newVMIDStore()
			if err != nil {
				t.Fatal(err)
			}
			if id, ok := store.Get("web"); !ok || id != record.VMID {
				t.Fatal("failure lost stable VMID")
			}
			if failure != "create" {
				err := upCmdRunE(nil, []string{"web"})
				if err == nil || !strings.Contains(err.Error(), "--recreate") {
					t.Fatalf("partial provisioning was silently accepted: %v", err)
				}
			}
		})
	}
}

type failingInitBackend struct{ *guardedLifecycleBackend }

func (b *failingInitBackend) SetInitCommand(int, string) error { return fmt.Errorf("init rejected") }

func TestNamedServicesAreValidatedBeforeLifecycleMutation(t *testing.T) {
	for _, operation := range []string{"up", "recreate", "down"} {
		t.Run(operation, func(t *testing.T) {
			fake, record := provisionWeb(t)
			args := []string{"web", "missing"}
			var err error
			switch operation {
			case "up":
				err = upCmdRunE(nil, args)
			case "recreate":
				err = upCmdRunE(recreationCommand(t), args)
			case "down":
				err = downCmdRunE(nil, args)
			}
			if err == nil || !strings.Contains(err.Error(), "not found") {
				t.Fatalf("expected unknown service error, got %v", err)
			}
			if fake.status[record.VMID] != proxmox.StatusRunning {
				t.Fatal("earlier service was changed before unknown name rejected")
			}
			assertDeploymentRecord(t, record)
		})
	}
}

func TestMappedUnrelatedContainerIsNeverModified(t *testing.T) {
	for _, operation := range []string{"up", "recreate", "down"} {
		t.Run(operation, func(t *testing.T) {
			fake, record := provisionWeb(t)
			fake.names[record.VMID] = "unrelated"
			var err error
			switch operation {
			case "up":
				err = upCmdRunE(nil, []string{"web"})
			case "recreate":
				err = upCmdRunE(recreationCommand(t), []string{"web"})
			case "down":
				removeContainers = true
				t.Cleanup(func() { removeContainers = false })
				err = downCmdRunE(nil, []string{"web"})
			}
			if err == nil || !strings.Contains(err.Error(), "unrelated") {
				t.Fatalf("expected identity mismatch rejection, got %v", err)
			}
			if fake.status[record.VMID] != proxmox.StatusRunning || fake.names[record.VMID] != "unrelated" {
				t.Fatal("unrelated mapped CT was modified")
			}
			assertDeploymentRecord(t, record)
		})
	}
}

func TestOverriddenNodeDefaultChangeLeavesContainerUnchanged(t *testing.T) {
	fake, record := provisionWeb(t)
	t.Setenv("PROXMOX_ROOTFS_GB", "64")
	guard := &guardedLifecycleBackend{fakeBackend: fake, blockCreate: true, blockStart: true}
	proxmoxBackendFactory = func() (proxmox.Backend, error) { return guard, nil }
	if err := upCmdRunE(nil, []string{"web"}); err != nil {
		t.Fatal(err)
	}
	if fake.created[record.VMID].RootFSSize != 2 {
		t.Fatal("explicit rootfs size was overridden")
	}
	assertDeploymentRecord(t, record)
}

func TestAllRecreationConfigsValidatedBeforeFirstContainerStopped(t *testing.T) {
	fake, record := provisionWeb(t)
	configFile = writeComposeFile(t, nginxCompose+"  db:\n    image: ubuntu:20.04\n    memory:\n      limit: invalid\n")
	if err := upCmdRunE(recreationCommand(t), []string{"web", "db"}); err == nil {
		t.Fatal("expected invalid later service error")
	}
	if fake.status[record.VMID] != proxmox.StatusRunning {
		t.Fatal("first service stopped before later config validated")
	}
	assertDeploymentRecord(t, record)
}

func TestAllRecreationTemplatesPreparedBeforeFirstContainerStopped(t *testing.T) {
	fake, record := provisionWeb(t)
	configFile = writeComposeFile(t, nginxCompose+"  db:\n    image: unavailable:image\n")
	ociConvertFn = func(string, string, oci.RuntimeOverrides) (*oci.ConvertResult, error) {
		return nil, fmt.Errorf("template unavailable")
	}
	if err := upCmdRunE(recreationCommand(t), []string{"web", "db"}); err == nil {
		t.Fatal("expected later service template error")
	}
	if fake.status[record.VMID] != proxmox.StatusRunning {
		t.Fatal("first service stopped before later template ready")
	}
	assertDeploymentRecord(t, record)
}

func TestFailedInitialStartKeepsMappingWithoutSuccessfulDesiredState(t *testing.T) {
	fake, cleanup := setupBackendTest(t)
	defer cleanup()
	configFile = writeComposeFile(t, nginxCompose)
	guard := &guardedLifecycleBackend{fakeBackend: fake, blockStart: true}
	proxmoxBackendFactory = func() (proxmox.Backend, error) { return guard, nil }
	if err := upCmdRunE(nil, []string{"web"}); err == nil {
		t.Fatal("expected start failure")
	}
	store, err := newVMIDStore()
	if err != nil {
		t.Fatal(err)
	}
	id, mapped := store.Get("web")
	if !mapped || fake.status[id] != proxmox.StatusStopped {
		t.Fatal("failed initial start lost allocated stopped container")
	}
	state, err := loadDeploymentState()
	if err != nil {
		t.Fatal(err)
	}
	if _, recorded := state.services["web"]; recorded {
		t.Fatal("failed initial start recorded successful desired state")
	}
	guard.blockStart = false
	if err := upCmdRunE(recreationCommand(t), []string{"web"}); err != nil {
		t.Fatal(err)
	}
	if fake.status[id] != proxmox.StatusRunning {
		t.Fatal("explicit recovery did not retain failed provisioning VMID")
	}
}

func TestDownRemovalFailurePreservesMappingAndDesiredState(t *testing.T) {
	fake, record := provisionWeb(t)
	guard := &guardedLifecycleBackend{fakeBackend: fake, blockDestroy: true}
	proxmoxBackendFactory = func() (proxmox.Backend, error) { return guard, nil }
	removeContainers = true
	t.Cleanup(func() { removeContainers = false })
	if err := downCmdRunE(nil, []string{"web"}); err == nil {
		t.Fatal("expected remove failure")
	}
	store, err := newVMIDStore()
	if err != nil {
		t.Fatal(err)
	}
	if id, mapped := store.Get("web"); !mapped || id != record.VMID {
		t.Fatal("failed removal lost mapping")
	}
	if _, exists := fake.created[record.VMID]; !exists {
		t.Fatal("failed removal lost container")
	}
	assertDeploymentRecord(t, record)
}

func TestDownRemovalClearsMissingContainerState(t *testing.T) {
	fake, record := provisionWeb(t)
	if err := fake.Destroy(record.VMID); err != nil {
		t.Fatal(err)
	}
	removeContainers = true
	t.Cleanup(func() { removeContainers = false })
	if err := downCmdRunE(nil, []string{"web"}); err != nil {
		t.Fatal(err)
	}
	store, err := newVMIDStore()
	if err != nil {
		t.Fatal(err)
	}
	if _, mapped := store.Get("web"); mapped {
		t.Fatal("missing CT mapping retained after down --rm")
	}
	state, err := loadDeploymentState()
	if err != nil {
		t.Fatal(err)
	}
	if _, recorded := state.services["web"]; recorded {
		t.Fatal("missing CT desired state retained after down --rm")
	}
}

func TestNativeTemplateReadinessFailurePreservesDeployment(t *testing.T) {
	for _, failure := range []string{"missing", "directory", "relative", "empty", "unreadable"} {
		t.Run(failure, func(t *testing.T) {
			fake, record := provisionWeb(t)
			configFile = writeComposeFile(t, strings.ReplaceAll(nginxCompose, "nginx:alpine", "local:vztmpl/native.tar.xz"))
			dir := t.TempDir()
			path := filepath.Join(dir, "missing.tar.xz")
			switch failure {
			case "directory":
				path = dir
			case "relative":
				path = "relative.tar.xz"
			case "empty":
				path = ""
			}
			binary := filepath.Join(dir, "pvesm")
			if err := os.WriteFile(binary, []byte("#!/bin/sh\nprintf '%s\\n' \"$TEST_TEMPLATE_PATH\"\n"), 0755); err != nil {
				t.Fatal(err)
			}
			t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
			t.Setenv("TEST_TEMPLATE_PATH", path)
			templateReadinessFn = validateTemplateReadiness
			if failure == "unreadable" {
				templateReadinessFn = func(string) error { return os.ErrPermission }
			}
			err := upCmdRunE(recreationCommand(t), []string{"web"})
			if err == nil || !strings.Contains(err.Error(), "not ready") {
				t.Fatalf("expected readiness rejection, got %v", err)
			}
			if fake.status[record.VMID] != proxmox.StatusRunning || fake.names[record.VMID] != "web" {
				t.Fatal("unavailable native template changed usable CT")
			}
			assertDeploymentRecord(t, record)
		})
	}
}

func TestNativeTemplateReadinessAllowsExistingRegularFile(t *testing.T) {
	fake, record := provisionWeb(t)
	configFile = writeComposeFile(t, strings.ReplaceAll(nginxCompose, "nginx:alpine", "local:vztmpl/native.tar.xz"))
	dir := t.TempDir()
	path := filepath.Join(dir, "native.tar.xz")
	if err := os.WriteFile(path, []byte("native-template"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "pvesm"), []byte("#!/bin/sh\nprintf '%s\\n' \"$TEST_TEMPLATE_PATH\"\n"), 0755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("TEST_TEMPLATE_PATH", path)
	templateReadinessFn = validateTemplateReadiness
	if err := upCmdRunE(recreationCommand(t), []string{"web"}); err != nil {
		t.Fatal(err)
	}
	if fake.status[record.VMID] != proxmox.StatusRunning || fake.created[record.VMID].OSTemplate != "local:vztmpl/native.tar.xz" {
		t.Fatal("ready native template was not provisioned at stable VMID")
	}
}

func TestCorruptVMIDMappingsAreRejectedBeforeDestruction(t *testing.T) {
	for _, corruption := range []string{"duplicate", "different-recorded-id"} {
		for _, operation := range []string{"recreate", "down"} {
			t.Run(corruption+"/"+operation, func(t *testing.T) {
				fake, record := provisionWeb(t)
				mapping := fmt.Sprintf("{\"web\":%d,\"db\":%d}", record.VMID, record.VMID)
				if corruption == "different-recorded-id" {
					otherID := record.VMID + 1
					if err := fake.Create(otherID, proxmox.CreateOptions{Hostname: "web"}); err != nil {
						t.Fatal(err)
					}
					if err := fake.Start(otherID); err != nil {
						t.Fatal(err)
					}
					mapping = fmt.Sprintf("{\"web\":%d}", otherID)
				}
				if err := os.WriteFile(vmidStorePath, []byte(mapping), 0644); err != nil {
					t.Fatal(err)
				}
				var err error
				if operation == "recreate" {
					err = upCmdRunE(recreationCommand(t), []string{"web"})
				} else {
					removeContainers = true
					t.Cleanup(func() { removeContainers = false })
					err = downCmdRunE(nil, []string{"web"})
				}
				if err == nil || !strings.Contains(err.Error(), "refusing") {
					t.Fatalf("expected corrupt mapping refusal, got %v", err)
				}
				for _, status := range fake.status {
					if status != proxmox.StatusRunning {
						t.Fatal("mapping corruption stopped an existing CT")
					}
				}
				if _, exists := fake.created[record.VMID]; !exists {
					t.Fatal("mapping corruption removed original CT")
				}
				assertDeploymentRecord(t, record)
			})
		}
	}
}

func TestConfigDriftReportsSortedChangedFieldsWithoutSecrets(t *testing.T) {
	fake, cleanup := setupBackendTest(t)
	defer cleanup()
	configFile = writeComposeFile(t, nginxCompose+"    environment: {TOKEN: secret-before}\n")
	if err := upCmdRunE(nil, []string{"web"}); err != nil {
		t.Fatal(err)
	}
	state, err := loadDeploymentState()
	if err != nil {
		t.Fatal(err)
	}
	record := state.services["web"]
	configFile = writeComposeFile(t, nginxCompose+"    environment: {TOKEN: secret-after}\n    command: []\n    memory:\n      limit: 512M\n")
	err = upCmdRunE(nil, []string{"web"})
	want := "changed fields: command, effective node options, environment, memory"
	if err == nil || !strings.Contains(err.Error(), want) || !strings.Contains(err.Error(), "--recreate") {
		t.Fatalf("expected sorted field-level drift report, got %v", err)
	}
	if strings.Contains(err.Error(), "secret-before") || strings.Contains(err.Error(), "secret-after") {
		t.Fatal("drift error disclosed environment values")
	}
	data, err := os.ReadFile(vmidStorePath + ".state.json")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "secret-before") || strings.Contains(string(data), "secret-after") {
		t.Fatal("deployment metadata persisted secret values instead of hashes")
	}
	if fake.status[record.VMID] != proxmox.StatusRunning {
		t.Fatal("drift reporting modified existing CT")
	}
	assertDeploymentRecord(t, record)
}

func TestLegacyDigestWithoutFieldHashesDoesNotGuessDifferences(t *testing.T) {
	fake, record := provisionWeb(t)
	state, err := loadDeploymentState()
	if err != nil {
		t.Fatal(err)
	}
	record.Fields = nil
	state.services["web"] = record
	if err := state.save(); err != nil {
		t.Fatal(err)
	}
	if err := upCmdRunE(nil, []string{"web"}); err != nil {
		t.Fatalf("matching legacy digest should still be accepted: %v", err)
	}
	configFile = writeComposeFile(t, nginxCompose+"    memory:\n      limit: 512M\n")
	err = upCmdRunE(nil, []string{"web"})
	if err == nil || !strings.Contains(err.Error(), "field-level differences unavailable in legacy deployment state") || !strings.Contains(err.Error(), "--recreate") {
		t.Fatalf("expected honest legacy difference report, got %v", err)
	}
	if fake.status[record.VMID] != proxmox.StatusRunning {
		t.Fatal("legacy drift reporting modified existing CT")
	}
	assertDeploymentRecord(t, record)
}
