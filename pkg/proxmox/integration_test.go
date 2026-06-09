//go:build integration
// +build integration

package proxmox_test

import (
	"os"
	"os/exec"
	"strconv"
	"testing"
	"time"

	"github.com/larkinwc/proxmox-lxc-compose/pkg/proxmox"
)

// These tests exercise the real `pct` CLI and therefore must run on a Proxmox
// node as root. They are gated behind both the `integration` build tag and the
// PROXMOX_INTEGRATION=1 environment variable so they never run in CI by default.
//
// Run with:
//
//	PROXMOX_INTEGRATION=1 PROXMOX_TEST_TEMPLATE="local:vztmpl/alpine-3.19-default_20240207_amd64.tar.xz" \
//	  PROXMOX_TEST_STORAGE=local-lvm PROXMOX_TEST_VMID=999 \
//	  go test -tags integration ./pkg/proxmox/ -run Integration -v

func requireIntegration(t *testing.T) {
	t.Helper()
	if os.Getenv("PROXMOX_INTEGRATION") != "1" {
		t.Skip("set PROXMOX_INTEGRATION=1 to run Proxmox integration tests")
	}
	if _, err := exec.LookPath("pct"); err != nil {
		t.Skip("pct binary not found; integration tests require a Proxmox node")
	}
	if os.Geteuid() != 0 {
		t.Skip("Proxmox integration tests require root")
	}
}

func testVMID(t *testing.T) int {
	t.Helper()
	v := os.Getenv("PROXMOX_TEST_VMID")
	if v == "" {
		v = "999"
	}
	vmid, err := strconv.Atoi(v)
	if err != nil {
		t.Fatalf("invalid PROXMOX_TEST_VMID: %v", err)
	}
	return vmid
}

func TestIntegrationContainerLifecycle(t *testing.T) {
	requireIntegration(t)

	template := os.Getenv("PROXMOX_TEST_TEMPLATE")
	if template == "" {
		t.Skip("set PROXMOX_TEST_TEMPLATE to a valid vztmpl volid")
	}
	storage := os.Getenv("PROXMOX_TEST_STORAGE")
	if storage == "" {
		storage = "local-lvm"
	}

	b := proxmox.NewPCTBackend()
	vmid := testVMID(t)

	// Best-effort cleanup of any leftover from a previous run.
	_ = b.Stop(vmid)
	_ = b.Destroy(vmid)

	opts := proxmox.CreateOptions{
		Hostname:     "lxc-compose-it",
		OSTemplate:   template,
		Storage:      storage,
		RootFSSize:   2,
		MemoryMB:     256,
		Unprivileged: true,
		Nets:         []string{"name=eth0,bridge=vmbr0,ip=dhcp"},
	}

	if err := b.Create(vmid, opts); err != nil {
		t.Fatalf("create failed: %v", err)
	}
	t.Cleanup(func() {
		_ = b.Stop(vmid)
		_ = b.Destroy(vmid)
	})

	if err := b.Start(vmid); err != nil {
		t.Fatalf("start failed: %v", err)
	}

	// Give the container a moment to report running.
	deadline := time.Now().Add(15 * time.Second)
	var status proxmox.Status
	for time.Now().Before(deadline) {
		st, err := b.Status(vmid)
		if err != nil {
			t.Fatalf("status failed: %v", err)
		}
		status = st
		if st == proxmox.StatusRunning {
			break
		}
		time.Sleep(time.Second)
	}
	if status != proxmox.StatusRunning {
		t.Fatalf("container not running, got %v", status)
	}

	infos, err := b.List()
	if err != nil {
		t.Fatalf("list failed: %v", err)
	}
	found := false
	for _, info := range infos {
		if info.VMID == vmid {
			found = true
		}
	}
	if !found {
		t.Errorf("vmid %d not present in list", vmid)
	}

	if err := b.Stop(vmid); err != nil {
		t.Fatalf("stop failed: %v", err)
	}
}
