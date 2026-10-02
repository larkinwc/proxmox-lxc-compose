package oci

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

type runtimeCapture struct {
	Args []string
	Env  map[string]string
	Cwd  string
}

func TestRuntimeCapture(_ *testing.T) {
	output := os.Getenv("LXC_RUNTIME_CAPTURE")
	if output == "" {
		return
	}
	args := os.Args
	for i, arg := range args {
		if arg == "--" {
			args = args[i+1:]
			break
		}
	}
	cwd, err := os.Getwd()
	if err != nil {
		os.Exit(2)
	}
	data, err := json.Marshal(runtimeCapture{Args: args, Cwd: cwd, Env: map[string]string{"A": os.Getenv("A"), "B": os.Getenv("B"), "C": os.Getenv("C")}})
	if err != nil || os.WriteFile(output, data, 0600) != nil {
		os.Exit(2)
	}
	os.Exit(0)
}

func captureWrapper(t *testing.T, rootfs, wrapper string) runtimeCapture {
	t.Helper()
	output := filepath.Join(t.TempDir(), "capture.json")
	bin := t.TempDir()
	// Suppress host network mutation; assertions concern the real executed process.
	must(t, os.WriteFile(filepath.Join(bin, "ip"), []byte("#!/bin/sh\nexit 0\n"), 0755))
	wrapperFile := filepath.Join(rootfs, wrapper)
	cmd := exec.Command("/bin/sh", wrapperFile)
	cmd.Env = []string{"PATH=" + bin, "LXC_RUNTIME_CAPTURE=" + output}
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("execute wrapper: %v: %s", err, out)
	}
	data, err := os.ReadFile(output)
	must(t, err)
	var capture runtimeCapture
	must(t, json.Unmarshal(data, &capture))
	return capture
}

func TestRuntimeInheritanceAndQuoting(t *testing.T) {
	entrypoint := []string{os.Args[0], "-test.run=TestRuntimeCapture", "--"}
	cwd := t.TempDir()
	defaults := RuntimeConfig{Entrypoint: entrypoint, Command: []string{"image default", "it's quoted"}, Environment: map[string]string{"A": "image", "B": "image b"}, WorkingDir: cwd, Network: []RuntimeNetwork{}}
	cases := []struct {
		name     string
		override RuntimeOverrides
		want     []string
	}{
		{"inherit", RuntimeOverrides{}, defaults.Command},
		{"clear-command", RuntimeOverrides{Command: []string{}}, []string{}},
		{"override-entrypoint-suppresses-cmd", RuntimeOverrides{Entrypoint: entrypoint}, []string{}},
		{"explicit-command", RuntimeOverrides{Entrypoint: entrypoint, Command: []string{"new space", "a'b", "$(touch injected); $HOME\nnext"}}, []string{"new space", "a'b", "$(touch injected); $HOME\nnext"}},
		{"clear-entrypoint-use-command", RuntimeOverrides{Entrypoint: []string{}, Command: append(append([]string{}, entrypoint...), "only command")}, []string{"only command"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tc.override.Env = map[string]string{"A": "alias", "C": "space ' quote; $(bad)\nline"}
			tc.override.Environment = map[string]string{"A": "canonical", "B": ""}
			runtime, err := ResolveRuntime(defaults, tc.override)
			must(t, err)
			root := t.TempDir()
			wrapper, err := WriteInitWrapper(root, runtime)
			must(t, err)
			capture := captureWrapper(t, root, wrapper)
			if !reflect.DeepEqual(capture.Args, tc.want) {
				t.Fatalf("argv = %#v, want %#v", capture.Args, tc.want)
			}
			wantEnv := map[string]string{"A": "canonical", "B": "", "C": tc.override.Env["C"]}
			if !reflect.DeepEqual(capture.Env, wantEnv) {
				t.Fatalf("env = %#v", capture.Env)
			}
			if capture.Cwd != cwd {
				t.Fatalf("cwd = %q", capture.Cwd)
			}
		})
	}
	if defaults.Environment["A"] != "image" {
		t.Fatal("resolution mutated image defaults")
	}
}

func TestRuntimeEmptyMapsRetainImageEnv(t *testing.T) {
	defaults := RuntimeConfig{Entrypoint: []string{os.Args[0], "-test.run=TestRuntimeCapture", "--"}, Environment: map[string]string{"A": "image"}, Network: []RuntimeNetwork{}}
	for _, overrides := range []RuntimeOverrides{{}, {Env: map[string]string{}, Environment: map[string]string{}}} {
		runtime, err := ResolveRuntime(defaults, overrides)
		must(t, err)
		root := t.TempDir()
		wrapper, err := WriteInitWrapper(root, runtime)
		must(t, err)
		if got := captureWrapper(t, root, wrapper).Env["A"]; got != "image" {
			t.Fatalf("image env = %q", got)
		}
	}
}

func TestRuntimeClearProcessUsesNativeInit(t *testing.T) {
	runtime, err := ResolveRuntime(RuntimeConfig{Entrypoint: []string{"old"}, Command: []string{"old command"}}, RuntimeOverrides{Entrypoint: []string{}, Command: []string{}})
	must(t, err)
	root := t.TempDir()
	wrapper, err := WriteInitWrapper(root, runtime)
	must(t, err)
	if wrapper != "" {
		t.Fatal("cleared process should use native init")
	}
}

func TestRuntimeRejectsInvalidBeforeWrites(t *testing.T) {
	cases := []RuntimeOverrides{
		{Env: map[string]string{"bad-name": "value"}}, {Environment: map[string]string{"9BAD": "value"}},
		{Environment: map[string]string{"A": "bad\x00value"}}, {Command: []string{"bad\x00arg"}},
		{Entrypoint: []string{"bad\x00entry"}}, {Env: map[string]string{"": "value"}},
	}
	for _, overrides := range cases {
		if _, err := ResolveRuntime(RuntimeConfig{}, overrides); err == nil {
			t.Fatalf("accepted invalid config %#v", overrides)
		}
		root := filepath.Join(t.TempDir(), "uncreated")
		if _, err := ConvertOCIToLXC("irrelevant", filepath.Join(root, "image.tar.gz"), overrides); err == nil {
			t.Fatal("converter accepted invalid runtime")
		}
		if _, err := os.Stat(root); !os.IsNotExist(err) {
			t.Fatal("invalid input mutated host")
		}
	}
	root := filepath.Join(t.TempDir(), "uncreated")
	if _, err := WriteInitWrapper(root, RuntimeConfig{Environment: map[string]string{"bad;name": "x"}}); err == nil {
		t.Fatal("wrapper accepted invalid environment")
	}
	if _, err := os.Stat(root); !os.IsNotExist(err) {
		t.Fatal("invalid wrapper mutated host")
	}
	if _, err := WriteInitWrapper(root, RuntimeConfig{Network: []RuntimeNetwork{{Name: "eth0;bad", IP: "not-ip"}}}); err == nil || !strings.Contains(err.Error(), "network") {
		t.Fatalf("invalid network: %v", err)
	}
}
