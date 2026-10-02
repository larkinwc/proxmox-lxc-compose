package main

import (
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
	// InitCmd is the generated guest runtime wrapper, empty when the service
	// boots distro init directly.
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
