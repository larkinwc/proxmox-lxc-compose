package oci

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDistroFamily(t *testing.T) {
	cases := map[string]string{
		`NAME="Alpine Linux"`: "alpine",
		`ID=debian`:           "debian",
		`NAME="Ubuntu"`:       "debian",
		`NAME="Void"`:         "unknown",
	}
	for content, want := range cases {
		dir := t.TempDir()
		if err := os.MkdirAll(filepath.Join(dir, "etc"), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "etc", "os-release"), []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
		if got := DistroFamily(dir); got != want {
			t.Errorf("DistroFamily(%q) = %q, want %q", content, got, want)
		}
	}

	// Missing os-release -> unknown.
	if got := DistroFamily(t.TempDir()); got != "unknown" {
		t.Errorf("missing os-release: got %q, want unknown", got)
	}
}

func TestFixLogSymlinks(t *testing.T) {
	dir := t.TempDir()
	logDir := filepath.Join(dir, "var", "log", "nginx")
	if err := os.MkdirAll(logDir, 0755); err != nil {
		t.Fatal(err)
	}
	// The classic OCI pattern: logs symlinked to std streams.
	if err := os.Symlink("/dev/stdout", filepath.Join(logDir, "access.log")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("/dev/stderr", filepath.Join(logDir, "error.log")); err != nil {
		t.Fatal(err)
	}
	// A normal symlink that must be left alone.
	if err := os.Symlink("/etc/hosts", filepath.Join(logDir, "other.log")); err != nil {
		t.Fatal(err)
	}

	count, err := FixLogSymlinks(dir)
	if err != nil {
		t.Fatal(err)
	}
	if count != 2 {
		t.Errorf("fixed %d symlinks, want 2", count)
	}

	// access.log and error.log should now be regular, writable files.
	for _, name := range []string{"access.log", "error.log"} {
		info, err := os.Lstat(filepath.Join(logDir, name))
		if err != nil {
			t.Fatal(err)
		}
		if info.Mode()&os.ModeSymlink != 0 {
			t.Errorf("%s is still a symlink", name)
		}
	}
	// The unrelated symlink must be untouched.
	info, _ := os.Lstat(filepath.Join(logDir, "other.log"))
	if info.Mode()&os.ModeSymlink == 0 {
		t.Error("other.log should remain a symlink")
	}
}

func TestFixLogSymlinksNoLogDir(t *testing.T) {
	// No /var/log present -> no error, zero fixed.
	count, err := FixLogSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Errorf("count = %d, want 0", count)
	}
}

func TestWriteNetworkConfig(t *testing.T) {
	dir := t.TempDir()
	rel, err := WriteNetworkConfig(dir, "debian")
	if err != nil {
		t.Fatal(err)
	}
	if rel != filepath.Join("etc", "network", "interfaces") {
		t.Errorf("rel = %q", rel)
	}
	data, err := os.ReadFile(filepath.Join(dir, rel))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "iface eth0 inet dhcp") {
		t.Errorf("interfaces missing dhcp stanza:\n%s", data)
	}
}

func TestWriteInitWrapperEmpty(t *testing.T) {
	// No command -> no wrapper.
	path, err := WriteInitWrapper(t.TempDir(), RuntimeConfig{})
	if err != nil {
		t.Fatal(err)
	}
	if path != "" {
		t.Errorf("expected empty path, got %q", path)
	}
}

func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}
