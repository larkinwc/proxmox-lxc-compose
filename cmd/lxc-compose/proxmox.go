package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/larkinwc/proxmox-lxc-compose/pkg/oci"
	"github.com/larkinwc/proxmox-lxc-compose/pkg/proxmox"
)

// proxmoxBackendFactory builds the Proxmox backend. It is a package var so
// tests can substitute a fake backend without invoking the real pct binary.
var proxmoxBackendFactory = func() (proxmox.Backend, error) {
	return proxmox.NewPCTBackend(), nil
}

// vmidStorePath is the location of the persisted name<->VMID mapping.
var vmidStorePath = defaultVMIDStorePath()

func defaultVMIDStorePath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ".lxc-compose-vmids.json"
	}
	return filepath.Join(home, ".lxc-compose", "vmids.json")
}

func newBackend() (proxmox.Backend, error) {
	return proxmoxBackendFactory()
}

func newVMIDStore() (*proxmox.VMIDStore, error) {
	base := proxmox.DefaultVMIDBase
	if v := os.Getenv("PROXMOX_VMID_BASE"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			base = n
		}
	}
	return proxmox.NewVMIDStore(vmidStorePath, base)
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// translateOptions assembles node-level defaults from the environment. Proxmox
// concepts (storage IDs, bridges, template volids) have no compose equivalent,
// so they are sourced from env vars with sensible defaults.
func translateOptions(name string) proxmox.TranslateOptions {
	rootGB := 8
	if v := os.Getenv("PROXMOX_ROOTFS_GB"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			rootGB = n
		}
	}
	return proxmox.TranslateOptions{
		Hostname:        name,
		DefaultStorage:  envOr("PROXMOX_STORAGE", "local-lvm"),
		DefaultBridge:   envOr("PROXMOX_BRIDGE", "vmbr0"),
		DefaultRootFSGB: rootGB,
	}
}

// templateCacheDir is where converted OCI templates are placed so Proxmox's
// `local` storage can reference them as local:vztmpl/<name>.
var templateCacheDir = "/var/lib/vz/template/cache"

// ociConvertFn is overridable in tests so `up` wiring can be exercised without
// Docker or a real Proxmox node.
var ociConvertFn = oci.ConvertOCIToLXC

// preparedTemplate is the result of resolving a service's image into something
// pct can provision from.
type preparedTemplate struct {
	// OSTemplate is the volid to pass to pct (e.g. local:vztmpl/foo.tar.gz).
	OSTemplate string
	// InitCmd is the in-container init command captured from an OCI image
	// (empty when using a standard CT template).
	InitCmd string
}

// isProxmoxVolid reports whether an image reference is already a Proxmox CT
// template volid (e.g. "local:vztmpl/alpine-3.22.tar.xz") rather than an OCI
// image reference. Template volids always contain a ":vztmpl/" segment, which
// OCI references (including registry-with-port forms like "registry:5000/img")
// never do.
func isProxmoxVolid(image string) bool {
	return strings.Contains(image, ":vztmpl/")
}

// ociTemplatePath returns the cache path for a converted OCI image.
func ociTemplatePath(image string) string {
	safe := strings.NewReplacer("/", "_", ":", "-").Replace(image)
	return filepath.Join(templateCacheDir, fmt.Sprintf("oci-%s.tar.gz", safe))
}

// prepareTemplate resolves a service's image into a pct-usable template,
// auto-detecting intent from the image reference:
//
//   - A Proxmox template volid (contains ":vztmpl/") is used verbatim.
//   - Any other reference is treated as an OCI image and converted into the
//     local template cache. Conversion is cached: an existing converted
//     template is reused unless `force` is set.
//
// An empty image leaves OSTemplate unset so Translate can derive a best-effort
// volid (and pct will fail clearly if it doesn't exist).
func prepareTemplate(name, image string, force bool) (preparedTemplate, error) {
	if image == "" {
		return preparedTemplate{}, nil
	}
	if isProxmoxVolid(image) {
		return preparedTemplate{OSTemplate: image}, nil
	}

	outPath := ociTemplatePath(image)
	volid := "local:vztmpl/" + filepath.Base(outPath)

	// Reuse a previously converted template unless a refresh was requested.
	// The init wrapper is baked into the cached rootfs at a deterministic path.
	if !force {
		if _, err := os.Stat(outPath); err == nil {
			fmt.Printf("Using cached template for image '%s' (service '%s')\n", image, name)
			return preparedTemplate{OSTemplate: volid, InitCmd: oci.InitWrapperPath}, nil
		}
	}

	fmt.Printf("Converting OCI image '%s' for service '%s'...\n", image, name)
	result, err := ociConvertFn(image, outPath)
	if err != nil {
		return preparedTemplate{}, fmt.Errorf("failed to convert image %q: %w", image, err)
	}

	return preparedTemplate{
		OSTemplate: "local:vztmpl/" + filepath.Base(result.OutputPath),
		InitCmd:    result.InitWrapperPath,
	}, nil
}

// backendVMIDs returns the VMIDs currently present on the node (best effort).
func backendVMIDs(b proxmox.Backend) []int {
	infos, err := b.List()
	if err != nil {
		return nil
	}
	ids := make([]int, 0, len(infos))
	for _, info := range infos {
		ids = append(ids, info.VMID)
	}
	return ids
}
