package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/larkinwc/proxmox-lxc-compose/pkg/common"
	"github.com/larkinwc/proxmox-lxc-compose/pkg/oci"
)

func runtimeTestCache(t *testing.T) {
	t.Helper()
	previous := templateCacheDir
	templateCacheDir = t.TempDir()
	t.Cleanup(func() { templateCacheDir = previous })
}

func seedImageRuntime(t *testing.T, image string, runtime oci.RuntimeConfig) string {
	t.Helper()
	root := t.TempDir()
	wrapper, err := oci.WriteInitWrapper(root, runtime)
	if err != nil {
		t.Fatal(err)
	}
	archive := ociTemplatePath(image)
	cmd := exec.Command("tar", "-czf", archive, "-C", root, ".")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("pack fixture: %v: %s", err, out)
	}
	if err := saveRuntime(&oci.ConvertResult{OutputPath: archive, Runtime: runtime, InitWrapperPath: wrapper}); err != nil {
		t.Fatal(err)
	}
	return archive
}

func executeCachedTemplate(t *testing.T, template preparedTemplate, output string) string {
	t.Helper()
	root := t.TempDir()
	archive := filepath.Join(templateCacheDir, strings.TrimPrefix(template.OSTemplate, "local:vztmpl/"))
	cmd := exec.Command("tar", "-xzf", archive, "-C", root)
	if data, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("unpack fixture: %v: %s", err, data)
	}
	bin := t.TempDir()
	if err := os.WriteFile(filepath.Join(bin, "ip"), []byte("#!/bin/sh\nexit 0\n"), 0755); err != nil {
		t.Fatal(err)
	}
	cmd = exec.Command("/bin/sh", filepath.Join(root, template.InitCmd))
	cmd.Env = []string{"PATH=" + bin, "OUTPUT=" + output}
	if data, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("execute cached wrapper: %v: %s", err, data)
	}
	data, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func TestServiceRuntimeCacheIsolationAndPrecedence(t *testing.T) {
	runtimeTestCache(t)
	image := "example/image:tag"
	defaults := oci.RuntimeConfig{Entrypoint: []string{"/bin/sh", "-c", `printf '%s|%s|%s' "$VALUE" "$1" "$2" >"$OUTPUT"`, "consumer"}, Command: []string{"image argument", "default two"}, Environment: map[string]string{"VALUE": "image"}, Network: []oci.RuntimeNetwork{}}
	base := seedImageRuntime(t, image, defaults)
	before, err := os.ReadFile(base)
	if err != nil {
		t.Fatal(err)
	}
	first := common.Container{Image: image, Command: []string{"spaces ' quotes", "$(not executed);"}, Env: map[string]string{"VALUE": "alias"}, Environment: map[string]string{"VALUE": "canonical"}}
	second := common.Container{Image: image, Env: map[string]string{"VALUE": "second"}}
	a, err := prepareServiceTemplate("first", first, false)
	if err != nil {
		t.Fatal(err)
	}
	b, err := prepareServiceTemplate("second", second, false)
	if err != nil {
		t.Fatal(err)
	}
	if a.OSTemplate == b.OSTemplate {
		t.Fatal("different effective runtimes share archive")
	}
	if got := executeCachedTemplate(t, a, filepath.Join(t.TempDir(), "first")); got != "canonical|spaces ' quotes|$(not executed);" {
		t.Fatalf("first runtime = %q", got)
	}
	if got := executeCachedTemplate(t, b, filepath.Join(t.TempDir(), "second")); got != "second|image argument|default two" {
		t.Fatalf("second runtime = %q", got)
	}
	// Equivalent effective configuration shares cache despite different service name/alias.
	first.Env = map[string]string{"VALUE": "ignored alias"}
	reused, err := prepareServiceTemplate("third", first, false)
	if err != nil {
		t.Fatal(err)
	}
	if reused != a {
		t.Fatal("equivalent runtime did not reuse cache")
	}
	if got := executeCachedTemplate(t, a, filepath.Join(t.TempDir(), "first-again")); got != "canonical|spaces ' quotes|$(not executed);" {
		t.Fatalf("other service mutated first cache: %q", got)
	}
	after, err := os.ReadFile(base)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before, after) {
		t.Fatal("service overrides mutated shared image template")
	}
}

func TestCachedMetadataRetainsEmptyInit(t *testing.T) {
	runtimeTestCache(t)
	seedImageRuntime(t, "no-command", oci.RuntimeConfig{})
	result, err := prepareServiceTemplate("native", common.Container{Image: "no-command"}, false)
	if err != nil {
		t.Fatal(err)
	}
	if result.InitCmd != "" {
		t.Fatalf("cache guessed wrapper %q", result.InitCmd)
	}
}

func TestServiceInvalidRuntimeDoesNotMutateHost(t *testing.T) {
	previous := templateCacheDir
	templateCacheDir = filepath.Join(t.TempDir(), "uncreated")
	t.Cleanup(func() { templateCacheDir = previous })
	for _, service := range []common.Container{
		{Image: "example", Environment: map[string]string{"BAD;KEY": "value"}},
		{Image: "local:vztmpl/archive.tar.gz", Command: []string{"bad\x00arg"}},
		{Image: "example", Network: &common.NetworkConfig{IP: "not an IP"}},
	} {
		if _, err := prepareServiceTemplate("invalid", service, false); err == nil {
			t.Fatal("accepted invalid runtime")
		}
		if _, err := os.Stat(templateCacheDir); !os.IsNotExist(err) {
			t.Fatal("invalid config mutated template cache")
		}
	}
}

func TestTemplateServiceOverridesDoNotMutateOriginal(t *testing.T) {
	runtimeTestCache(t)
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "etc"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "etc", "distro-init"), []byte("native"), 0644); err != nil {
		t.Fatal(err)
	}
	source := filepath.Join(t.TempDir(), "template.tar.gz")
	cmd := exec.Command("tar", "-czf", source, "-C", root, ".")
	if data, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("pack template: %v: %s", err, data)
	}
	before, err := os.ReadFile(source)
	if err != nil {
		t.Fatal(err)
	}
	previous := templatePathFn
	templatePathFn = func(string) (string, error) { return source, nil }
	t.Cleanup(func() { templatePathFn = previous })
	service := common.Container{Image: "local:vztmpl/template.tar.gz", Command: []string{"/bin/sh", "-c", `printf '%s|%s' "$VALUE" "$1" >"$OUTPUT"`, "consumer", "template argv"}, Environment: map[string]string{"VALUE": "template env"}}
	result, err := prepareServiceTemplate("template", service, false)
	if err != nil {
		t.Fatal(err)
	}
	if got := executeCachedTemplate(t, result, filepath.Join(t.TempDir(), "result")); got != "template env|template argv" {
		t.Fatalf("template process = %q", got)
	}
	after, err := os.ReadFile(source)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before, after) {
		t.Fatal("overrides mutated source CT template")
	}
	unchanged, err := prepareServiceTemplate("plain", common.Container{Image: service.Image}, false)
	if err != nil {
		t.Fatal(err)
	}
	if unchanged.OSTemplate != service.Image || unchanged.InitCmd != "" {
		t.Fatalf("plain template = %#v", unchanged)
	}
}
