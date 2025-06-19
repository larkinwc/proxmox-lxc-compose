//go:build pcttest
// +build pcttest

package pctbackend_test

import (
	"fmt"
	"os/exec"
	"reflect"
	"testing"

	container "github.com/larkinwc/proxmox-lxc-compose/pkg/container"
	"github.com/larkinwc/proxmox-lxc-compose/pkg/logging"
)

func init() {
	logging.Init(logging.Config{Development: true, Level: "error"})
}

func TestPCTManager_List(t *testing.T) {
	original := container.ExecCommand
	defer func() { container.ExecCommand = original }()

	sample := "VMID NAME                STATUS\n100  alpine              running\n101  ubuntu              stopped\n"
	container.ExecCommand = func(name string, args ...string) *exec.Cmd {
		if name == "pct" && len(args) > 0 && args[0] == "list" {
			return exec.Command("bash", "-c", fmt.Sprintf("printf '%s'", sample))
		}
		return exec.Command("true")
	}

	m, _ := container.NewPCTManager("")
	got, err := m.List()
	if err != nil {
		t.Fatalf("List error: %v", err)
	}
	want := []container.Container{{Name: "alpine", State: "running"}, {Name: "ubuntu", State: "stopped"}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected containers:\ngot:  %+v\nwant: %+v", got, want)
	}
}

func TestPCTManager_Start_Stop(t *testing.T) {
	original := container.ExecCommand
	defer func() { container.ExecCommand = original }()

	calls := [][]string{}
	sample := "VMID NAME                STATUS\n123  test                stopped\n"
	container.ExecCommand = func(name string, args ...string) *exec.Cmd {
		if name == "pct" && len(args) > 0 && args[0] == "list" {
			return exec.Command("bash", "-c", fmt.Sprintf("printf '%s'", sample))
		}
		calls = append(calls, append([]string{name}, args...))
		return exec.Command("true")
	}

	m, _ := container.NewPCTManager("")
	if err := m.Start("test"); err != nil {
		t.Fatalf("Start error: %v", err)
	}
	if err := m.Stop("test"); err != nil {
		t.Fatalf("Stop error: %v", err)
	}

	foundStart, foundStop := false, false
	for _, c := range calls {
		if len(c) >= 2 && c[1] == "start" {
			foundStart = true
		}
		if len(c) >= 2 && c[1] == "stop" {
			foundStop = true
		}
	}
	if !foundStart || !foundStop {
		t.Fatalf("pct start/stop not invoked, calls: %+v", calls)
	}
}
