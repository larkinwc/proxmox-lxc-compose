package oci

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestInspectImageParsing(t *testing.T) {
	orig := dockerExec
	defer func() { dockerExec = orig }()

	dockerExec = func(_ string, args ...string) *exec.Cmd {
		return helperCommand(t, "inspect-ok", args...)
	}

	entrypoint, cmd, err := inspectImage("nginx:alpine")
	if err != nil {
		t.Fatal(err)
	}
	if len(entrypoint) != 1 || entrypoint[0] != "/docker-entrypoint.sh" {
		t.Errorf("entrypoint = %v", entrypoint)
	}
	want := []string{"nginx", "-g", "daemon off;"}
	if strings.Join(cmd, "\x00") != strings.Join(want, "\x00") {
		t.Errorf("cmd = %v, want %v", cmd, want)
	}
}

// TestConvertOCIToLXCFlow exercises the full conversion orchestration with a
// mocked docker + tar so it runs without Docker or a Proxmox node. It verifies
// the output is renamed to .tar.gz, the rootfs is post-processed, and the
// captured command becomes an init wrapper.
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
	res, err := ConvertOCIToLXC("nginx:alpine", out)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(res.OutputPath, ".tar.gz") {
		t.Errorf("output not renamed to .tar.gz: %q", res.OutputPath)
	}
	if _, err := os.Stat(res.OutputPath); err != nil {
		t.Errorf("output file missing: %v", err)
	}
	if res.InitWrapperPath != InitWrapperPath {
		t.Errorf("init wrapper = %q", res.InitWrapperPath)
	}
	if res.PostProcess.Distro != "alpine" {
		t.Errorf("distro = %q", res.PostProcess.Distro)
	}
	if res.PostProcess.LogLinksFixed != 1 {
		t.Errorf("log links fixed = %d, want 1", res.PostProcess.LogLinksFixed)
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
		os.Stdout.WriteString(`[{"Config":{"Entrypoint":["/docker-entrypoint.sh"],"Cmd":["nginx","-g","daemon off;"]}}]`)
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
