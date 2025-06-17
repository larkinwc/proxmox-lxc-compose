package container

import (
	"fmt"
	"os/exec"
)

// NewManager returns a Manager implementation based on the requested backend.
// backend may be "pct", "lxc", or "auto" (default). If "auto" is chosen, the
// function will prefer the Proxmox `pct` backend when the `pct` binary is
// available in PATH and fall back to the existing LXC implementation.
func NewManager(backend string, configPath string) (Manager, error) {
	switch backend {
	case "pct":
		if _, err := exec.LookPath("pct"); err != nil {
			return nil, fmt.Errorf("requested backend 'pct' but 'pct' binary not found in PATH")
		}
		return NewPCTManager(configPath)
	case "lxc":
		return NewLXCManager(configPath)
	case "", "auto":
		if _, err := exec.LookPath("pct"); err == nil {
			if mgr, err2 := NewPCTManager(configPath); err2 == nil {
				return mgr, nil
			}
			// If pct manager creation failed, fall back to lxc
		}
		return NewLXCManager(configPath)
	default:
		return nil, fmt.Errorf("unknown backend '%s' (valid: pct, lxc, auto)", backend)
	}
}
