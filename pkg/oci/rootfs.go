package oci

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// DistroFamily inspects an extracted rootfs and returns a coarse distro family
// ("alpine", "debian", or "unknown") based on /etc/os-release.
func DistroFamily(rootfs string) string {
	path, err := guestPath(rootfs, "/etc/os-release")
	if err != nil {
		return "unknown"
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "unknown"
	}
	content := strings.ToLower(string(data))
	switch {
	case strings.Contains(content, "alpine"):
		return "alpine"
	case strings.Contains(content, "debian"), strings.Contains(content, "ubuntu"):
		return "debian"
	default:
		return "unknown"
	}
}

// FixLogSymlinks replaces symlinks under /var/log that point at /dev/stdout or
// /dev/stderr (the common OCI logging pattern) with empty regular files.
//
// In an unprivileged LXC container those device targets are not writable the
// way they are under Docker, which causes daemons like nginx to abort with
// "Permission denied". Returns the number of symlinks rewritten.
func FixLogSymlinks(rootfs string) (int, error) {
	logDir, err := guestPath(rootfs, "/var/log")
	if err != nil {
		return 0, err
	}
	count := 0

	err = filepath.Walk(logDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			// /var/log may not exist; that's fine.
			if os.IsNotExist(err) {
				return filepath.SkipDir
			}
			return err
		}
		if info.Mode()&os.ModeSymlink == 0 {
			return nil
		}
		target, err := os.Readlink(path)
		if err != nil {
			return nil // skip unreadable links
		}
		if target != "/dev/stdout" && target != "/dev/stderr" {
			return nil
		}
		if err := os.Remove(path); err != nil {
			return fmt.Errorf("failed to remove log symlink %s: %w", path, err)
		}
		f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return fmt.Errorf("failed to create log file %s: %w", path, err)
		}
		_ = f.Close()
		count++
		return nil
	})
	if err != nil && !os.IsNotExist(err) {
		return count, err
	}
	return count, nil
}

// WriteNetworkConfig writes a DHCP network configuration for eth0 into the
// rootfs so that, when booted with the distro's own init, the primary
// interface comes up automatically. This is the documented fallback for the
// init-command mechanism. Returns the path written (relative to rootfs).
func WriteNetworkConfig(rootfs string, distro string) (string, error) {
	switch distro {
	case "alpine", "debian", "unknown":
		// Both Alpine (ifupdown) and Debian use /etc/network/interfaces.
		dir, err := guestPath(rootfs, "/etc/network")
		if err != nil {
			return "", err
		}
		if err := os.MkdirAll(dir, 0755); err != nil {
			return "", fmt.Errorf("failed to create network dir: %w", err)
		}
		path, err := guestPath(rootfs, "/etc/network/interfaces")
		if err != nil {
			return "", err
		}
		content := "auto lo\niface lo inet loopback\n\nauto eth0\niface eth0 inet dhcp\n"
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			return "", fmt.Errorf("failed to write interfaces file: %w", err)
		}
		return filepath.Join("etc", "network", "interfaces"), nil
	default:
		return "", nil
	}
}

// shellQuote single-quotes a string for safe embedding in a /bin/sh command.
func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

// PostProcessResult summarizes the rootfs mutations applied during conversion.
type PostProcessResult struct {
	Distro          string
	LogLinksFixed   int
	NetworkConfig   string
	InitWrapperPath string
}

// PostProcessRootfs applies OCI compatibility fixes and the effective runtime.
func PostProcessRootfs(rootfs string, runtime RuntimeConfig) (PostProcessResult, error) {
	if err := validateRuntime(runtime, nil); err != nil {
		return PostProcessResult{}, err
	}
	var res PostProcessResult
	res.Distro = DistroFamily(rootfs)

	fixed, err := FixLogSymlinks(rootfs)
	if err != nil {
		return res, err
	}
	res.LogLinksFixed = fixed

	netPath, err := WriteNetworkConfig(rootfs, res.Distro)
	if err != nil {
		return res, err
	}
	res.NetworkConfig = netPath

	wrapper, err := WriteInitWrapper(rootfs, runtime)
	if err != nil {
		return res, err
	}
	res.InitWrapperPath = wrapper

	return res, nil
}
