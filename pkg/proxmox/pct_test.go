package proxmox

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCreateArgs(t *testing.T) {
	opts := CreateOptions{
		Hostname:     "web",
		OSTemplate:   "local:vztmpl/alpine-3.19.tar.zst",
		Storage:      "local-lvm",
		RootFSSize:   8,
		Cores:        2,
		CPUUnits:     1024,
		MemoryMB:     2048,
		SwapMB:       1024,
		Unprivileged: true,
		Features:     "nesting=1",
		Nameservers:  []string{"1.1.1.1", "8.8.8.8"},
		Nets:         []string{"name=eth0,bridge=vmbr0,ip=dhcp"},
		Mounts:       []string{"local-lvm:4,mp=/data"},
		Start:        true,
	}
	args := createArgs(101, opts)
	joined := strings.Join(args, " ")

	for _, want := range []string{
		"create 101 local:vztmpl/alpine-3.19.tar.zst",
		"--hostname web",
		"--rootfs local-lvm:8",
		"--cores 2",
		"--cpuunits 1024",
		"--memory 2048",
		"--swap 1024",
		"--unprivileged 1",
		"--features nesting=1",
		"--nameserver 1.1.1.1 8.8.8.8",
		"--net0 name=eth0,bridge=vmbr0,ip=dhcp",
		"--mp0 local-lvm:4,mp=/data",
		"--start 1",
	} {
		if !strings.Contains(joined, want) {
			t.Errorf("args missing %q\n got: %s", want, joined)
		}
	}
}

func TestPCTCreateInvokesPct(t *testing.T) {
	recorded, restore := mockExec("", false)
	defer restore()

	b := NewPCTBackend()
	err := b.Create(100, CreateOptions{
		OSTemplate: "local:vztmpl/alpine.tar.zst",
		Storage:    "local-lvm",
		RootFSSize: 8,
	})
	if err != nil {
		t.Fatalf("Create error: %v", err)
	}
	if len(*recorded) != 1 {
		t.Fatalf("expected 1 command, got %d", len(*recorded))
	}
	cmd := (*recorded)[0]
	if cmd.name != "pct" {
		t.Errorf("command = %q, want pct", cmd.name)
	}
	if cmd.args[0] != "create" || cmd.args[1] != "100" {
		t.Errorf("args = %v", cmd.args)
	}
}

func TestPCTCreateRequiresTemplate(t *testing.T) {
	_, restore := mockExec("", false)
	defer restore()

	b := NewPCTBackend()
	if err := b.Create(100, CreateOptions{Storage: "local-lvm"}); err == nil {
		t.Error("expected error when OS template is empty")
	}
}

func TestPCTLifecycleCommands(t *testing.T) {
	recorded, restore := mockExec("", false)
	defer restore()

	b := NewPCTBackend()
	ops := []struct {
		fn   func(int) error
		verb string
	}{
		{b.Start, "start"},
		{b.Stop, "stop"},
		{b.Shutdown, "shutdown"},
		{b.Suspend, "suspend"},
		{b.Resume, "resume"},
		{b.Destroy, "destroy"},
	}
	for _, op := range ops {
		if err := op.fn(123); err != nil {
			t.Fatalf("%s error: %v", op.verb, err)
		}
	}

	if len(*recorded) != len(ops) {
		t.Fatalf("expected %d commands, got %d", len(ops), len(*recorded))
	}
	for i, op := range ops {
		cmd := (*recorded)[i]
		if cmd.args[0] != op.verb || cmd.args[1] != "123" {
			t.Errorf("command %d = %v, want %s 123", i, cmd.args, op.verb)
		}
	}
}

func TestPCTCommandFailure(t *testing.T) {
	_, restore := mockExec("", true)
	defer restore()

	b := NewPCTBackend()
	if err := b.Start(100); err == nil {
		t.Error("expected error when pct exits non-zero")
	}
}

func TestParseStatus(t *testing.T) {
	cases := map[string]Status{
		"status: running\n":   StatusRunning,
		"status: stopped\n":   StatusStopped,
		"status: suspended\n": StatusPaused,
		"status: weird\n":     StatusUnknown,
	}
	for in, want := range cases {
		if got := parseStatus(in); got != want {
			t.Errorf("parseStatus(%q) = %v, want %v", in, got, want)
		}
	}
}

func TestPCTStatus(t *testing.T) {
	_, restore := mockExec("status: running\n", false)
	defer restore()

	b := NewPCTBackend()
	st, err := b.Status(100)
	if err != nil {
		t.Fatal(err)
	}
	if st != StatusRunning {
		t.Errorf("status = %v, want running", st)
	}
}

func TestParseList(t *testing.T) {
	out := `VMID       Status     Lock         Name
100        running                 web
101        stopped                 db
`
	infos := parseList(out)
	if len(infos) != 2 {
		t.Fatalf("expected 2 containers, got %d", len(infos))
	}
	if infos[0].VMID != 100 || infos[0].Name != "web" || infos[0].Status != StatusRunning {
		t.Errorf("infos[0] = %+v", infos[0])
	}
	if infos[1].VMID != 101 || infos[1].Name != "db" || infos[1].Status != StatusStopped {
		t.Errorf("infos[1] = %+v", infos[1])
	}
}

func TestPCTVMIDs(t *testing.T) {
	out := `VMID       Status     Lock         Name
100        running                 web
105        stopped                 db
`
	_, restore := mockExec(out, false)
	defer restore()

	b := NewPCTBackend()
	ids, err := b.VMIDs()
	if err != nil {
		t.Fatal(err)
	}
	if len(ids) != 2 || ids[0] != 100 || ids[1] != 105 {
		t.Errorf("VMIDs = %v, want [100 105]", ids)
	}
}

func TestSetInitCommand(t *testing.T) {
	dir := t.TempDir()
	confPath := filepath.Join(dir, "950.conf")
	base := "arch: amd64\nhostname: web\nmemory: 256\n"
	if err := os.WriteFile(confPath, []byte(base), 0640); err != nil {
		t.Fatal(err)
	}

	b := &PCTBackend{binary: "pct", confDir: dir}

	// Empty path is a no-op.
	if err := b.SetInitCommand(950, ""); err != nil {
		t.Fatalf("empty SetInitCommand: %v", err)
	}
	if data, _ := os.ReadFile(confPath); string(data) != base {
		t.Error("empty init path should not modify config")
	}

	// Setting a command appends the raw lxc.init.cmd key.
	if err := b.SetInitCommand(950, "/usr/local/bin/lxc-compose-init.sh"); err != nil {
		t.Fatalf("SetInitCommand: %v", err)
	}
	data, _ := os.ReadFile(confPath)
	if !strings.Contains(string(data), "lxc.init.cmd: /usr/local/bin/lxc-compose-init.sh") {
		t.Errorf("config missing init cmd:\n%s", data)
	}

	// Calling again must not duplicate the key.
	if err := b.SetInitCommand(950, "/usr/local/bin/lxc-compose-init.sh"); err != nil {
		t.Fatal(err)
	}
	data, _ = os.ReadFile(confPath)
	if n := strings.Count(string(data), "lxc.init.cmd:"); n != 1 {
		t.Errorf("init cmd appears %d times, want 1", n)
	}
}

func TestSetInitCommandMissingConf(t *testing.T) {
	b := &PCTBackend{binary: "pct", confDir: t.TempDir()}
	if err := b.SetInitCommand(999, "/sbin/init"); err == nil {
		t.Error("expected error for missing config file")
	}
}
