package oci

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"testing"
)

func TestInspectImageParsing(t *testing.T) {
	orig := dockerExec
	defer func() { dockerExec = orig }()

	dockerExec = func(_ string, args ...string) *exec.Cmd {
		return helperCommand(t, "inspect-ok", args...)
	}

	runtime, err := inspectImage("nginx:alpine")
	if err != nil {
		t.Fatal(err)
	}
	runtime.Network = []RuntimeNetwork{}
	root := t.TempDir()
	wrapper, err := WriteInitWrapper(root, runtime)
	if err != nil {
		t.Fatal(err)
	}
	capture := captureWrapper(t, root, wrapper)
	if !reflect.DeepEqual(capture.Args, []string{"image default", "it's quoted"}) || capture.Env["A"] != "image env" || capture.Cwd != "/" {
		t.Fatalf("inspected image runtime = %#v", capture)
	}
}

// The docker fixture provides real archive bytes; the converted template is
// extracted and its generated shell wrapper executes a real consumer process.
func TestConvertOCIToLXCFlow(t *testing.T) {
	if _, err := exec.LookPath("docker"); err != nil {
		t.Skip("docker not in PATH; LookPath guard would fail before mock")
	}
	orig := dockerExec
	defer func() { dockerExec = orig }()

	dockerExec = func(name string, args ...string) *exec.Cmd {
		switch {
		case name == "docker" && len(args) > 0 && args[0] == "pull":
			return helperCommand(t, "noop", args...)
		case name == "docker" && len(args) > 0 && args[0] == "inspect":
			return helperCommand(t, "inspect-ok", args...)
		case name == "docker" && len(args) > 0 && args[0] == "create":
			return helperCommand(t, "create-ok", args...)
		case name == "docker" && len(args) > 0 && args[0] == "rm":
			return helperCommand(t, "noop", args...)
		case name == "docker" && len(args) > 0 && args[0] == "export":
			return helperCommand(t, "export-rootfs", args...)
		case name == "tar":
			// Use the real tar for extract/pack against the mocked stream.
			return exec.Command(name, args...)
		default:
			// Fail loudly so an unexpected command surfaces as a test failure
			// instead of silently passing.
			return helperCommand(t, "unexpected", append([]string{name}, args...)...)
		}
	}

	out := filepath.Join(t.TempDir(), "nginx") // no extension on purpose
	res, err := ConvertOCIToLXC("nginx:alpine", out, RuntimeOverrides{Command: []string{"override arg"}, Environment: map[string]string{"A": "service env"}})
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	if err := extractArchive(res.OutputPath, root); err != nil {
		t.Fatal(err)
	}
	capture := captureWrapper(t, root, res.InitWrapperPath)
	if !reflect.DeepEqual(capture.Args, []string{"override arg"}) || capture.Env["A"] != "service env" || capture.Cwd != "/" {
		t.Fatalf("converted runtime = %#v", capture)
	}
	data, err := os.ReadFile(filepath.Join(root, "var", "log", "nginx", "access.log"))
	if err != nil || len(data) != 0 {
		t.Fatalf("converted access log: %q, %v", data, err)
	}
}

// helperCommand builds an exec.Cmd that re-invokes the test binary as a fake
// external command. The standard Go pattern for mocking exec.
func helperCommand(t *testing.T, mode string, args ...string) *exec.Cmd {
	t.Helper()
	cs := append([]string{"-test.run=TestHelperProcess", "--", mode}, args...)
	cmd := exec.Command(os.Args[0], cs...)
	cmd.Env = append(os.Environ(), "GO_WANT_HELPER_PROCESS=1")
	return cmd
}

// TestHelperProcess is not a real test; it stands in for docker/tar when
// invoked by helperCommand.
func TestHelperProcess(_ *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	args := os.Args
	for i, a := range args {
		if a == "--" {
			args = args[i+1:]
			break
		}
	}
	if len(args) == 0 {
		os.Exit(0)
	}
	mode := args[0]

	switch mode {
	case "noop":
		os.Exit(0)
	case "inspect-ok":
		data, err := json.Marshal([]interface{}{map[string]interface{}{"Config": map[string]interface{}{
			"Entrypoint": []string{os.Args[0], "-test.run=TestRuntimeCapture", "--"},
			"Cmd":        []string{"image default", "it's quoted"},
			"Env":        []string{"A=image env", "B=image b"}, "WorkingDir": "/",
		}}})
		if err != nil {
			os.Exit(1)
		}
		_, _ = os.Stdout.Write(data)
		os.Exit(0)
	case "create-ok":
		os.Stdout.WriteString("deadbeefcafe\n")
		os.Exit(0)
	case "export-rootfs":
		// Emit a tar stream of a minimal alpine-like rootfs with a std-stream
		// log symlink, so post-processing has something to operate on.
		if err := emitRootfsTar(); err != nil {
			os.Exit(1)
		}
		os.Exit(0)
	case "unexpected":
		os.Stderr.WriteString("unexpected command path in test mock\n")
		os.Exit(1)
	default:
		os.Exit(0)
	}
}

// emitRootfsTar writes a tar archive to stdout containing a minimal rootfs.
func emitRootfsTar() error {
	// Build the rootfs in a temp dir then tar it to stdout using the system tar.
	dir, err := os.MkdirTemp("", "fake-rootfs-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(dir)

	if err := os.MkdirAll(filepath.Join(dir, "etc"), 0755); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(dir, "etc", "os-release"), []byte(`NAME="Alpine Linux"`), 0644); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Join(dir, "var", "log", "nginx"), 0755); err != nil {
		return err
	}
	if err := os.Symlink("/dev/stdout", filepath.Join(dir, "var", "log", "nginx", "access.log")); err != nil {
		return err
	}

	cmd := exec.Command("tar", "-c", "-C", dir, ".")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
