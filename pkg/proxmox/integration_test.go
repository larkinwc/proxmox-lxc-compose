//go:build integration
// +build integration

package proxmox_test

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
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
	for _, binary := range []string{"lxc-info", "lxc-freeze", "lxc-unfreeze"} {
		if _, err := exec.LookPath(binary); err != nil {
			t.Fatalf("integration tests require %s: %v", binary, err)
		}
	}
	if os.Geteuid() != 0 {
		t.Skip("Proxmox integration tests require root")
	}
}

func testVMID(t *testing.T) int {
	t.Helper()
	v := os.Getenv("PROXMOX_TEST_VMID")
	if v == "" {
		t.Fatal("set PROXMOX_TEST_VMID to a dedicated disposable VMID; these tests stop/destroy it")
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

	// A real guest process updates a counter. Read it through the guest init's
	// root so observing a frozen guest never requires executing inside it.
	pid := integrationInitPID(t, vmid)
	counterPath := fmt.Sprintf("/proc/%d/root/tmp/lxc-compose-freeze-counter", pid)
	activity := exec.Command("pct", "exec", strconv.Itoa(vmid), "--", "sh", "-c",
		"i=0; while :; do i=$((i+1)); echo \"$i\" > /tmp/lxc-compose-freeze-counter.next; mv /tmp/lxc-compose-freeze-counter.next /tmp/lxc-compose-freeze-counter; sleep 1; done")
	if err := activity.Start(); err != nil {
		t.Fatalf("start guest activity: %v", err)
	}
	t.Cleanup(func() {
		_ = b.Resume(vmid)
		_ = activity.Process.Kill()
		_ = activity.Wait()
	})
	readCounter := func() int {
		data, err := os.ReadFile(counterPath)
		if err != nil {
			if os.IsNotExist(err) {
				return 0
			}
			t.Fatalf("read guest counter: %v", err)
		}
		n, err := strconv.Atoi(strings.TrimSpace(string(data)))
		if err != nil {
			t.Fatalf("parse guest counter %q: %v", data, err)
		}
		return n
	}
	waitForActivity := func(after int) int {
		deadline := time.Now().Add(15 * time.Second)
		for time.Now().Before(deadline) {
			if n := readCounter(); n > after {
				return n
			}
			time.Sleep(100 * time.Millisecond)
		}
		t.Fatalf("guest counter did not advance beyond %d", after)
		return 0
	}
	initial := waitForActivity(0)
	waitForActivity(initial)
	if err := b.Suspend(vmid); err != nil {
		t.Fatalf("freeze failed: %v", err)
	}
	assertIntegrationState(t, b, vmid, proxmox.StatusPaused)
	frozen := readCounter()
	if frozen == 0 {
		t.Fatal("guest froze during a counter write; cannot prove activity stopped")
	}
	time.Sleep(2500 * time.Millisecond)
	if got := readCounter(); got != frozen {
		t.Fatalf("frozen guest counter changed: %d -> %d", frozen, got)
	}
	if err := b.Resume(vmid); err != nil {
		t.Fatalf("unfreeze failed: %v", err)
	}
	assertIntegrationState(t, b, vmid, proxmox.StatusRunning)
	if got := integrationInitPID(t, vmid); got != pid {
		t.Fatalf("unfreeze replaced guest init: PID %d -> %d", pid, got)
	}
	waitForActivity(frozen)

	if err := b.Stop(vmid); err != nil {
		t.Fatalf("stop failed: %v", err)
	}
	assertIntegrationState(t, b, vmid, proxmox.StatusStopped)
}

func integrationInitPID(t *testing.T, vmid int) int {
	t.Helper()
	out, err := exec.Command("lxc-info", "-n", strconv.Itoa(vmid), "-p").CombinedOutput()
	if err != nil {
		t.Fatalf("inspect guest init PID: %v: %s", err, out)
	}
	fields := strings.Fields(string(out))
	if len(fields) != 2 || fields[0] != "PID:" {
		t.Fatalf("unexpected lxc-info PID output: %q", out)
	}
	pid, err := strconv.Atoi(fields[1])
	if err != nil || pid <= 0 {
		t.Fatalf("invalid guest init PID: %q", out)
	}
	return pid
}

func assertIntegrationState(t *testing.T, b proxmox.Backend, vmid int, want proxmox.Status) {
	t.Helper()
	got, err := b.Status(vmid)
	if err != nil || got != want {
		t.Fatalf("Status(%d) = %v, %v; want %v", vmid, got, err, want)
	}
	infos, err := b.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	for _, info := range infos {
		if info.VMID == vmid {
			if info.Status != want {
				t.Fatalf("List container %d status = %v; want %v", vmid, info.Status, want)
			}
			return
		}
	}
	t.Fatalf("container %d absent from List", vmid)
}
