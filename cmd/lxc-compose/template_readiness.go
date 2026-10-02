package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// templateReadinessFn can be replaced by lifecycle tests without a PVE host.
var templateReadinessFn = validateTemplateReadiness

func validateTemplateReadiness(volid string) error {
	if volid == "" {
		return fmt.Errorf("no template specified")
	}
	out, err := exec.Command("pvesm", "path", volid).Output()
	if err != nil {
		return fmt.Errorf("cannot resolve template %q with pvesm: %w", volid, err)
	}
	path := strings.TrimSpace(string(out))
	if !filepath.IsAbs(path) || strings.ContainsAny(path, "\r\n") {
		return fmt.Errorf("pvesm returned an invalid template path %q for %q", path, volid)
	}
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("template %q is unavailable: %w", volid, err)
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("template %q at %q is not a regular file", volid, path)
	}
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("template %q is unreadable: %w", volid, err)
	}
	return file.Close()
}
