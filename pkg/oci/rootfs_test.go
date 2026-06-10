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

func TestWriteInitWrapper(t *testing.T) {
	dir := t.TempDir()
	path, err := WriteInitWrapper(dir, []string{"/docker-entrypoint.sh"}, []string{"nginx", "-g", "daemon off;"})
	if err != nil {
		t.Fatal(err)
	}
	if path != InitWrapperPath {
		t.Errorf("path = %q, want %q", path, InitWrapperPath)
	}
	data, err := os.ReadFile(filepath.Join(dir, "usr", "local", "bin", "lxc-compose-init.sh"))
	if err != nil {
		t.Fatal(err)
	}
	script := string(data)
	if !strings.Contains(script, "exec '/docker-entrypoint.sh' 'nginx' '-g' 'daemon off;'") {
		t.Errorf("wrapper exec line wrong:\n%s", script)
	}
	if !strings.Contains(script, "udhcpc") || !strings.Contains(script, "ip link set eth0 up") {
		t.Errorf("wrapper missing network bring-up:\n%s", script)
	}
	if !strings.Contains(script, "/sys/class/net/eth0") {
		t.Errorf("wrapper missing eth0 wait loop:\n%s", script)
	}
	// Executable bit set.
	info, _ := os.Stat(filepath.Join(dir, "usr", "local", "bin", "lxc-compose-init.sh"))
	if info.Mode()&0111 == 0 {
		t.Error("wrapper is not executable")
	}
}

func TestWriteInitWrapperEmpty(t *testing.T) {
	// No command -> no wrapper.
	path, err := WriteInitWrapper(t.TempDir(), nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if path != "" {
		t.Errorf("expected empty path, got %q", path)
	}
}

func TestShellQuote(t *testing.T) {
	cases := map[string]string{
		"simple":      "'simple'",
		"with space":  "'with space'",
		"it's":        `'it'\''s'`,
		"daemon off;": "'daemon off;'",
	}
	for in, want := range cases {
		if got := shellQuote(in); got != want {
			t.Errorf("shellQuote(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestPostProcessRootfs(t *testing.T) {
	dir := t.TempDir()
	// Minimal alpine-like rootfs with a std-stream log symlink.
	must(t, os.MkdirAll(filepath.Join(dir, "etc"), 0755))
	must(t, os.WriteFile(filepath.Join(dir, "etc", "os-release"), []byte(`NAME="Alpine Linux"`), 0644))
	must(t, os.MkdirAll(filepath.Join(dir, "var", "log", "nginx"), 0755))
	must(t, os.Symlink("/dev/stdout", filepath.Join(dir, "var", "log", "nginx", "access.log")))

	res, err := PostProcessRootfs(dir, nil, []string{"nginx", "-g", "daemon off;"})
	if err != nil {
		t.Fatal(err)
	}
	if res.Distro != "alpine" {
		t.Errorf("distro = %q", res.Distro)
	}
	if res.LogLinksFixed != 1 {
		t.Errorf("log links fixed = %d, want 1", res.LogLinksFixed)
	}
	if res.NetworkConfig == "" {
		t.Error("expected network config to be written")
	}
	if res.InitWrapperPath != InitWrapperPath {
		t.Errorf("init wrapper = %q", res.InitWrapperPath)
	}
}

func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}
