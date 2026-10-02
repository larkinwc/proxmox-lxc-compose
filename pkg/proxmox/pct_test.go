package proxmox

import (
	"os"
	"os/exec"
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

func TestParseLXCState(t *testing.T) {
	cases := []struct {
		out  string
		want Status
	}{
		{"State:          RUNNING\n", StatusRunning},
		{"State: FROZEN\n", StatusPaused},
		{"State:\tSTOPPED\n", StatusStopped},
		{"", StatusUnknown},
		{"RUNNING\n", StatusUnknown},
		{"State: FREEZING\n", StatusUnknown},
		{"State: THAWED\n", StatusUnknown},
		{"State: RUNNING\nunexpected output", StatusUnknown},
	}
	for _, tc := range cases {
		got, err := parseLXCState(tc.out)
		if got != tc.want || (err != nil) != (tc.want == StatusUnknown) {
			t.Errorf("parseLXCState(%q) = %v, %v; want %v", tc.out, got, err, tc.want)
		}
	}
}

func TestPCTRuntimeState(t *testing.T) {
	cases := []struct {
		name        string
		pctStatus   string
		lxcState    string
		failCommand string
		want        Status
		wantError   string
	}{
		{"running", "running", "RUNNING", "", StatusRunning, ""},
		{"frozen", "running", "FROZEN", "", StatusPaused, ""},
		{"stopped", "stopped", "", "", StatusStopped, ""},
		{"stopped during inspection", "running", "STOPPED", "", StatusStopped, ""},
		{"pct paused", "paused", "FROZEN", "", StatusPaused, ""},
		{"unknown pct", "unexpected", "", "", StatusUnknown, "unknown pct status"},
		{"unknown lxc", "running", "FREEZING", "", StatusUnknown, "unrecognized lxc-info"},
		{"empty lxc", "running", "", "", StatusUnknown, "unrecognized lxc-info"},
		{"inspection failure", "running", "RUNNING", "lxc-info", StatusUnknown, "inspection denied"},
		{"pct failure", "running", "RUNNING", "pct", StatusUnknown, "inspection denied"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			orig := execCommand
			t.Cleanup(func() { execCommand = orig })
			execCommand = func(name string, args ...string) *exec.Cmd {
				if name == tc.failCommand {
					return exec.Command("sh", "-c", "printf 'inspection denied'; exit 1")
				}
				if name == "lxc-info" {
					if tc.pctStatus == "stopped" {
						t.Fatal("stopped container must not require LXC inspection")
					}
					out := ""
					if tc.lxcState != "" {
						out = "State: " + tc.lxcState + "\n"
					}
					return exec.Command("printf", "%s", out)
				}
				out := "status: " + tc.pctStatus + "\n"
				if len(args) > 0 && args[0] == "list" {
					out = "VMID Status Lock Name\n100 " + tc.pctStatus + " web\n"
				}
				return exec.Command("printf", "%s", out)
			}
			b := NewPCTBackend()
			state, err := b.Status(100)
			if state != tc.want {
				t.Errorf("Status = %v, want %v", state, tc.want)
			}
			assertStateError(t, err, tc.wantError)
			infos, err := b.List()
			assertStateError(t, err, tc.wantError)
			if tc.wantError == "" {
				if len(infos) != 1 || infos[0].VMID != 100 || infos[0].Name != "web" || infos[0].Status != tc.want {
					t.Errorf("List = %+v, want container 100/web with status %v", infos, tc.want)
				}
			} else if infos != nil {
				t.Errorf("List returned partial inventory on inspection failure: %+v", infos)
			}
		})
	}
}

func assertStateError(t *testing.T, err error, want string) {
	t.Helper()
	if want == "" {
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	} else if err == nil || !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want containing %q", err, want)
	}
}

func TestPCTFreezerFailure(t *testing.T) {
	_, restore := mockExec("", true)
	defer restore()
	b := NewPCTBackend()
	if err := b.Suspend(100); err == nil {
		t.Fatal("Suspend must report freezer failure")
	}
	if err := b.Resume(100); err == nil {
		t.Fatal("Resume must report unfreezer failure")
	}
}

func TestPCTListMixedStates(t *testing.T) {
	orig := execCommand
	t.Cleanup(func() { execCommand = orig })
	execCommand = func(name string, args ...string) *exec.Cmd {
		out := "VMID Status Lock Name\n100 running web\n101 stopped db\n102 running worker\n"
		if name == "lxc-info" {
			switch args[1] {
			case "100":
				out = "State: FROZEN\n"
			case "102":
				out = "State: RUNNING\n"
			default:
				t.Fatalf("unexpected inspection of container %s", args[1])
			}
		}
		return exec.Command("printf", "%s", out)
	}
	infos, err := NewPCTBackend().List()
	if err != nil {
		t.Fatal(err)
	}
	want := []ContainerInfo{
		{VMID: 100, Name: "web", Status: StatusPaused},
		{VMID: 101, Name: "db", Status: StatusStopped},
		{VMID: 102, Name: "worker", Status: StatusRunning},
	}
	if len(infos) != len(want) {
		t.Fatalf("List = %+v, want %+v", infos, want)
	}
	for i := range want {
		if infos[i] != want[i] {
			t.Errorf("List[%d] = %+v, want %+v", i, infos[i], want[i])
		}
	}
}
