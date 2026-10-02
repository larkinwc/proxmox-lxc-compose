package main

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/larkinwc/proxmox-lxc-compose/pkg/common"
	"github.com/larkinwc/proxmox-lxc-compose/pkg/oci"
)

func serviceNetwork(network *common.NetworkConfig) []oci.RuntimeNetwork {
	if network == nil {
		return nil
	}
	interfaces := network.Interfaces
	if len(interfaces) == 0 && (network.Type != "" || network.Bridge != "" || network.IP != "" || network.DHCP) {
		// Match the legacy translation path, which uses the default eth0 name.
		interfaces = []common.NetworkInterface{{IP: network.IP, Gateway: network.Gateway, DHCP: network.DHCP}}
	}
	if len(interfaces) == 0 {
		return nil
	}
	result := make([]oci.RuntimeNetwork, len(interfaces))
	for i, iface := range interfaces {
		name := iface.Interface
		if name == "" {
			name = fmt.Sprintf("eth%d", i)
		}
		result[i] = oci.RuntimeNetwork{Name: name, IP: iface.IP, Gateway: iface.Gateway, DHCP: iface.DHCP}
	}
	return result
}

// templatePathFn resolves a volume using the node's configured storage backend.
var templatePathFn = func(volid string) (string, error) {
	out, err := exec.Command("pvesm", "path", volid).Output()
	if err != nil {
		return "", fmt.Errorf("resolve template %q: %w", volid, err)
	}
	path := strings.TrimSpace(string(out))
	if !filepath.IsAbs(path) {
		return "", fmt.Errorf("pvesm returned non-absolute template path %q", path)
	}
	return path, nil
}

func cachePath(identity interface{}) string {
	data, _ := json.Marshal(identity) // all identities are JSON-safe structs
	digest := sha256.Sum256(data)
	return filepath.Join(templateCacheDir, fmt.Sprintf("oci-%x.tar.gz", digest))
}

const runtimeCacheVersion = 3

func ociTemplatePath(image string) string {
	return cachePath(struct {
		Version int
		Image   string
	}{runtimeCacheVersion, image})
}

type runtimeCache struct {
	Version int
	Result  oci.ConvertResult
}

func cachedRuntime(path string) (*oci.ConvertResult, bool) {
	info, err := os.Stat(path)
	if err != nil || !info.Mode().IsRegular() {
		return nil, false
	}
	data, err := os.ReadFile(path + ".json")
	if err != nil {
		return nil, false
	}
	var metadata runtimeCache
	if json.Unmarshal(data, &metadata) != nil || metadata.Version != runtimeCacheVersion || metadata.Result.OutputPath != path {
		return nil, false
	}
	return &metadata.Result, true
}

func saveRuntime(result *oci.ConvertResult) error {
	data, err := json.Marshal(runtimeCache{Version: runtimeCacheVersion, Result: *result})
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(result.OutputPath), ".runtime-metadata-")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), result.OutputPath+".json")
}

func prepared(result *oci.ConvertResult) preparedTemplate {
	return preparedTemplate{OSTemplate: "local:vztmpl/" + filepath.Base(result.OutputPath), InitCmd: result.InitWrapperPath}
}

// prepareServiceTemplate isolates each effective runtime configuration in its own
// archive. Empty env maps add no values; neither nil nor empty clears image Env.
func prepareServiceTemplate(name string, svc common.Container, force bool) (preparedTemplate, error) {
	overrides := oci.RuntimeOverrides{Command: svc.Command, Entrypoint: svc.Entrypoint, Env: svc.Env, Environment: svc.Environment}
	if err := overrides.Validate(); err != nil {
		return preparedTemplate{}, fmt.Errorf("service %q: %w", name, err)
	}
	network := serviceNetwork(svc.Network)
	if _, err := oci.ResolveRuntime(oci.RuntimeConfig{Network: network}, overrides); err != nil {
		return preparedTemplate{}, fmt.Errorf("service %q: %w", name, err)
	}
	hasOverrides := svc.Command != nil || svc.Entrypoint != nil || len(svc.Env) != 0 || len(svc.Environment) != 0 || (!isProxmoxVolid(svc.Image) && network != nil)
	if svc.Image == "" {
		if hasOverrides {
			return preparedTemplate{}, fmt.Errorf("service %q requires an image for runtime overrides", name)
		}
		return preparedTemplate{}, nil
	}
	if isProxmoxVolid(svc.Image) && !hasOverrides {
		return preparedTemplate{OSTemplate: svc.Image}, nil
	}
	if err := os.MkdirAll(templateCacheDir, 0755); err != nil {
		return preparedTemplate{}, err
	}

	var defaults oci.RuntimeConfig
	var source string
	if isProxmoxVolid(svc.Image) {
		var err error
		source, err = templatePathFn(svc.Image)
		if err != nil {
			return preparedTemplate{}, err
		}
	} else {
		basePath := ociTemplatePath(svc.Image)
		base, ok := cachedRuntime(basePath)
		if force || !ok {
			fmt.Printf("Converting OCI image '%s' for service '%s'...\n", svc.Image, name)
			var err error
			base, err = ociConvertFn(svc.Image, basePath, oci.RuntimeOverrides{})
			if err != nil {
				return preparedTemplate{}, fmt.Errorf("convert image %q: %w", svc.Image, err)
			}
			if err := saveRuntime(base); err != nil {
				return preparedTemplate{}, err
			}
		}
		if !hasOverrides {
			return prepared(base), nil
		}
		defaults, source = base.Runtime, base.OutputPath
	}
	defaults.Network = network
	runtime, err := oci.ResolveRuntime(defaults, overrides)
	if err != nil {
		return preparedTemplate{}, err
	}
	// Include the source bytes, not just its volid: a force-refreshed image cannot
	// reuse an override archive derived from a previous image revision.
	sourceFile, err := os.Open(source)
	if err != nil {
		return preparedTemplate{}, err
	}
	digest := sha256.New()
	_, err = io.Copy(digest, sourceFile)
	closeErr := sourceFile.Close()
	if err != nil {
		return preparedTemplate{}, err
	}
	if closeErr != nil {
		return preparedTemplate{}, closeErr
	}
	outPath := cachePath(struct {
		Version int
		Image   string
		Source  string
		Runtime oci.RuntimeConfig
	}{runtimeCacheVersion, svc.Image, fmt.Sprintf("%x", digest.Sum(nil)), runtime})
	if !force {
		if result, ok := cachedRuntime(outPath); ok {
			return prepared(result), nil
		}
	}
	result, err := oci.ConvertTemplateRuntime(source, outPath, runtime)
	if err != nil {
		return preparedTemplate{}, fmt.Errorf("service %q runtime: %w", name, err)
	}
	if err := saveRuntime(result); err != nil {
		return preparedTemplate{}, err
	}
	return prepared(result), nil
}
