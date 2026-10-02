package oci

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"os"
	"path/filepath"
	"testing"
)

func testArchive(t *testing.T, headers []*tar.Header) []byte {
	t.Helper()
	var buffer bytes.Buffer
	tw := tar.NewWriter(&buffer)
	for _, header := range headers {
		must(t, tw.WriteHeader(header))
		if header.Typeflag == tar.TypeReg {
			_, err := tw.Write(bytes.Repeat([]byte("x"), int(header.Size)))
			must(t, err)
		}
	}
	must(t, tw.Close())
	return buffer.Bytes()
}

func TestArchiveRejectsPathAndLinkEscapes(t *testing.T) {
	host := t.TempDir()
	sentinel := filepath.Join(host, "sentinel")
	must(t, os.WriteFile(sentinel, []byte("untouched"), 0600))
	cases := [][]*tar.Header{
		{{Name: "../sentinel", Typeflag: tar.TypeReg, Size: 1, Mode: 0600}},
		{{Name: sentinel, Typeflag: tar.TypeReg, Size: 1, Mode: 0600}},
		{{Name: "link", Typeflag: tar.TypeLink, Linkname: sentinel}},
		{{Name: "link", Typeflag: tar.TypeLink, Linkname: "../sentinel"}},
		{{Name: "usr", Typeflag: tar.TypeSymlink, Linkname: "../../outside"}, {Name: "usr/file", Typeflag: tar.TypeReg, Size: 1, Mode: 0600}},
	}
	for _, headers := range cases {
		if err := extractTar(bytes.NewReader(testArchive(t, headers)), t.TempDir()); err == nil {
			t.Fatalf("accepted escape %#v", headers)
		}
		data, err := os.ReadFile(sentinel)
		must(t, err)
		if string(data) != "untouched" {
			t.Fatal("archive changed host sentinel")
		}
	}
}

func TestArchiveGuestSymlinksAndPostprocessingStayConfined(t *testing.T) {
	host := t.TempDir()
	must(t, os.MkdirAll(filepath.Join(host, "network"), 0755))
	must(t, os.WriteFile(filepath.Join(host, "network", "interfaces"), []byte("host network"), 0600))
	must(t, os.WriteFile(filepath.Join(host, "lxc-compose-init.sh"), []byte("host init"), 0600))
	root := t.TempDir()
	archive := testArchive(t, []*tar.Header{
		{Name: "etc", Typeflag: tar.TypeSymlink, Linkname: host},
		{Name: "usr/local/bin", Typeflag: tar.TypeSymlink, Linkname: host},
		{Name: "var/log", Typeflag: tar.TypeSymlink, Linkname: host},
		{Name: "bin", Typeflag: tar.TypeSymlink, Linkname: "/usr/bin"},
		{Name: "bin/guest-file", Typeflag: tar.TypeReg, Size: 1, Mode: 0644},
	})
	must(t, extractTar(bytes.NewReader(archive), root))
	_, err := PostProcessRootfs(root, RuntimeConfig{Command: []string{"/bin/true"}})
	must(t, err)
	data, err := os.ReadFile(filepath.Join(host, "network", "interfaces"))
	must(t, err)
	if string(data) != "host network" {
		t.Fatal("network fix escaped rootfs")
	}
	data, err = os.ReadFile(filepath.Join(host, "lxc-compose-init.sh"))
	must(t, err)
	if string(data) != "host init" {
		t.Fatal("runtime wrapper escaped rootfs")
	}
	data, err = os.ReadFile(filepath.Join(root, "usr", "bin", "guest-file"))
	must(t, err)
	if string(data) != "x" {
		t.Fatal("guest absolute bin symlink was not resolved in rootfs")
	}
	// The generated wrapper lives beneath root, at the guest-mapped target.
	if _, err := os.Stat(filepath.Join(root, host, "lxc-compose-init.sh")); err != nil {
		t.Fatal(err)
	}
}

func TestTemplateRuntimePreservesSourceAndReplacesProcess(t *testing.T) {
	root := t.TempDir()
	must(t, os.MkdirAll(filepath.Join(root, "etc"), 0755))
	must(t, os.WriteFile(filepath.Join(root, "etc", "distribution-state"), []byte("native init config"), 0644))
	source := filepath.Join(t.TempDir(), "source.tar.gz")
	must(t, packGzip(root, source))
	before, err := os.ReadFile(source)
	must(t, err)
	output := filepath.Join(t.TempDir(), "derived.tar.gz")
	runtime := RuntimeConfig{Entrypoint: []string{os.Args[0], "-test.run=TestRuntimeCapture", "--"}, Command: []string{"template override", "it's safe"}, Environment: map[string]string{"A": "template env"}, Network: []RuntimeNetwork{}}
	result, err := ConvertTemplateRuntime(source, output, runtime)
	must(t, err)
	guest := t.TempDir()
	must(t, extractArchive(output, guest))
	capture := captureWrapper(t, guest, result.InitWrapperPath)
	if len(capture.Args) != 2 || capture.Args[0] != "template override" || capture.Args[1] != "it's safe" || capture.Env["A"] != "template env" {
		t.Fatalf("template runtime = %#v", capture)
	}
	data, err := os.ReadFile(filepath.Join(guest, "etc", "distribution-state"))
	must(t, err)
	if string(data) != "native init config" {
		t.Fatal("changed distro configuration")
	}
	after, err := os.ReadFile(source)
	must(t, err)
	if !bytes.Equal(before, after) {
		t.Fatal("mutated original template")
	}
}

func TestTemplateClearingRemovesPreviousWrapper(t *testing.T) {
	root := t.TempDir()
	_, err := WriteInitWrapper(root, RuntimeConfig{Command: []string{"old-process"}})
	must(t, err)
	source := filepath.Join(t.TempDir(), "source.tar.gz")
	must(t, packGzip(root, source))
	output := filepath.Join(t.TempDir(), "cleared.tar.gz")
	result, err := ConvertTemplateRuntime(source, output, RuntimeConfig{})
	must(t, err)
	if result.InitWrapperPath != "" {
		t.Fatal("cleared template still advertises wrapper")
	}
	guest := t.TempDir()
	must(t, extractArchive(output, guest))
	if _, err := os.Stat(filepath.Join(guest, InitWrapperPath)); !os.IsNotExist(err) {
		t.Fatal("old wrapper still in cleared rootfs")
	}
}

func TestTemplateRejectsUnsafeArchiveBeforePublishing(t *testing.T) {
	source := filepath.Join(t.TempDir(), "unsafe.tar.gz")
	f, err := os.Create(source)
	must(t, err)
	gz := gzip.NewWriter(f)
	_, err = gz.Write(testArchive(t, []*tar.Header{{Name: "../../escape", Typeflag: tar.TypeReg, Size: 1, Mode: 0600}}))
	must(t, err)
	must(t, gz.Close())
	must(t, f.Close())
	output := filepath.Join(t.TempDir(), "must-not-exist.tar.gz")
	if _, err := ConvertTemplateRuntime(source, output, RuntimeConfig{Command: []string{"/bin/true"}}); err == nil {
		t.Fatal("unsafe source was accepted")
	}
	if _, err := os.Stat(output); !os.IsNotExist(err) {
		t.Fatal("published unsafe template")
	}
}
