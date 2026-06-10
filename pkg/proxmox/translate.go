package proxmox

import (
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"

	"github.com/larkinwc/proxmox-lxc-compose/pkg/common"
)

// TranslateOptions carries node/environment defaults that aren't expressible in
// the compose file but are needed to build a Proxmox container.
type TranslateOptions struct {
	// Hostname is the container hostname, normally the compose service name.
	Hostname string
	// OSTemplate is the Proxmox template volid to provision from. If empty,
	// Translate derives a best-effort guess from the image name, which the
	// caller should validate against `pveam available`.
	OSTemplate string
	// DefaultStorage is the storage ID used for the rootfs when the compose
	// file doesn't specify a pool (e.g. "local-lvm").
	DefaultStorage string
	// DefaultBridge is used for interfaces that don't specify a bridge.
	DefaultBridge string
	// DefaultRootFSGB is used when storage.root is unset.
	DefaultRootFSGB int
}

var sizeRegex = regexp.MustCompile(`(?i)^\s*(\d+(?:\.\d+)?)\s*([KMGTP]?B?)?\s*$`)

// parseSizeToMB converts a human size string ("2G", "512M", "1024") to MB.
// A bare number is interpreted as MB (Proxmox's native unit for memory).
func parseSizeToMB(s string) (int, error) {
	m := sizeRegex.FindStringSubmatch(s)
	if m == nil {
		return 0, fmt.Errorf("invalid size %q", s)
	}
	val, err := strconv.ParseFloat(m[1], 64)
	if err != nil {
		return 0, fmt.Errorf("invalid size %q: %w", s, err)
	}
	unit := strings.ToUpper(strings.TrimSuffix(m[2], "B"))
	switch unit {
	case "", "M":
		// already MB
	case "K":
		val /= 1024
	case "G":
		val *= 1024
	case "T":
		val *= 1024 * 1024
	case "P":
		val *= 1024 * 1024 * 1024
	default:
		return 0, fmt.Errorf("unknown size unit in %q", s)
	}
	return int(val + 0.5), nil
}

// parseSizeToGB converts a human size string to GB, rounding up so a container
// never gets less disk than requested.
func parseSizeToGB(s string) (int, error) {
	mb, err := parseSizeToMB(s)
	if err != nil {
		return 0, err
	}
	gb := mb / 1024
	if mb%1024 != 0 {
		gb++
	}
	if gb < 1 {
		gb = 1
	}
	return gb, nil
}

// deriveTemplateFromImage makes a best-effort Proxmox volid from a docker-style
// image reference. This is only a fallback; real templates should be provided
// explicitly via TranslateOptions.OSTemplate.
func deriveTemplateFromImage(image string) string {
	if image == "" {
		return ""
	}
	// Strip any registry path, keep the final component.
	parts := strings.Split(image, "/")
	last := parts[len(parts)-1]
	name := strings.ReplaceAll(last, ":", "-")
	return fmt.Sprintf("local:vztmpl/%s.tar.zst", name)
}

// firstInterface returns the effective primary interface, accounting for the
// legacy top-level network fields.
func firstInterface(net *common.NetworkConfig) *common.NetworkInterface {
	if net == nil {
		return nil
	}
	if len(net.Interfaces) > 0 {
		return &net.Interfaces[0]
	}
	if net.Type != "" || net.Bridge != "" || net.IP != "" || net.DHCP {
		return &common.NetworkInterface{
			Type:     net.Type,
			Bridge:   net.Bridge,
			IP:       net.IP,
			Gateway:  net.Gateway,
			DNS:      net.DNS,
			DHCP:     net.DHCP,
			Hostname: net.Hostname,
			MTU:      net.MTU,
			MAC:      net.MAC,
		}
	}
	return nil
}

// translateNet builds a single pct --netN value from an interface.
func translateNet(index int, iface common.NetworkInterface, defaultBridge string) string {
	var parts []string
	name := iface.Interface
	if name == "" {
		name = fmt.Sprintf("eth%d", index)
	}
	parts = append(parts, "name="+name)

	bridge := iface.Bridge
	if bridge == "" {
		bridge = defaultBridge
	}
	if bridge != "" {
		parts = append(parts, "bridge="+bridge)
	}
	if iface.MAC != "" {
		parts = append(parts, "hwaddr="+iface.MAC)
	}
	if iface.DHCP {
		parts = append(parts, "ip=dhcp")
	} else if iface.IP != "" {
		parts = append(parts, "ip="+iface.IP)
		if iface.Gateway != "" {
			parts = append(parts, "gw="+iface.Gateway)
		}
	}
	if iface.MTU > 0 {
		parts = append(parts, fmt.Sprintf("mtu=%d", iface.MTU))
	}
	return strings.Join(parts, ",")
}

// translateMount builds a pct --mpN value from a mount config.
func translateMount(m common.Mount, storage string) string {
	// For bind mounts (host path source), Proxmox uses the raw path as the
	// volume; for managed volumes it uses storage:size. We treat an absolute
	// source path as a bind mount.
	vol := m.Source
	isBind := strings.HasPrefix(m.Source, "/") ||
		strings.HasPrefix(m.Source, "./") ||
		strings.HasPrefix(m.Source, "../")
	if !isBind && storage != "" {
		vol = storage + ":" + m.Source
	}
	parts := []string{vol, "mp=" + m.Target}
	for _, opt := range m.Options {
		switch opt {
		case "ro", "readonly":
			parts = append(parts, "ro=1")
		}
	}
	return strings.Join(parts, ",")
}

// Translate converts a compose container definition into Proxmox CreateOptions.
func Translate(c *common.Container, opts TranslateOptions) (CreateOptions, error) {
	if c == nil {
		return CreateOptions{}, fmt.Errorf("container configuration is required")
	}

	co := CreateOptions{
		Hostname:     opts.Hostname,
		OSTemplate:   opts.OSTemplate,
		Storage:      opts.DefaultStorage,
		Unprivileged: true, // safe Proxmox default
		Extra:        map[string]string{},
	}

	if co.OSTemplate == "" {
		co.OSTemplate = deriveTemplateFromImage(c.Image)
	}

	// Storage / rootfs.
	rootGB := opts.DefaultRootFSGB
	if c.Storage != nil {
		if c.Storage.Pool != "" {
			co.Storage = c.Storage.Pool
		}
		if c.Storage.Root != "" {
			gb, err := parseSizeToGB(c.Storage.Root)
			if err != nil {
				return CreateOptions{}, fmt.Errorf("invalid storage.root: %w", err)
			}
			rootGB = gb
		}
		for _, m := range c.Storage.Mounts {
			co.Mounts = append(co.Mounts, translateMount(m, co.Storage))
		}
	}
	co.RootFSSize = rootGB

	// CPU.
	if c.CPU != nil {
		if c.CPU.Cores != nil {
			co.Cores = *c.CPU.Cores
		}
		if c.CPU.Shares != nil {
			co.CPUUnits = int(*c.CPU.Shares)
		}
		// Proxmox cpulimit is a count of cores; derive from quota/period if set.
		// Round up so a fractional limit (quota < period) isn't silently dropped.
		if c.CPU.Quota != nil && c.CPU.Period != nil && *c.CPU.Period > 0 {
			limit := float64(*c.CPU.Quota) / float64(*c.CPU.Period)
			if limit > 0 {
				co.CPULimit = int(math.Ceil(limit))
			}
		}
	}

	// Memory.
	if c.Memory != nil {
		if c.Memory.Limit != "" {
			mb, err := parseSizeToMB(c.Memory.Limit)
			if err != nil {
				return CreateOptions{}, fmt.Errorf("invalid memory.limit: %w", err)
			}
			co.MemoryMB = mb
		}
		if c.Memory.Swap != "" {
			mb, err := parseSizeToMB(c.Memory.Swap)
			if err != nil {
				return CreateOptions{}, fmt.Errorf("invalid memory.swap: %w", err)
			}
			co.SwapMB = mb
		}
	}

	// Security -> privileged / features.
	if c.Security != nil {
		switch strings.ToLower(c.Security.Isolation) {
		case "privileged":
			co.Unprivileged = false
		case "strict", "default", "":
			co.Unprivileged = true
		}
		if c.Security.Privileged {
			co.Unprivileged = false
		}
	}

	// Network.
	if c.Network != nil {
		if len(c.Network.Interfaces) > 0 {
			for i, iface := range c.Network.Interfaces {
				co.Nets = append(co.Nets, translateNet(i, iface, opts.DefaultBridge))
			}
		} else if iface := firstInterface(c.Network); iface != nil {
			co.Nets = append(co.Nets, translateNet(0, *iface, opts.DefaultBridge))
		}

		// DNS: prefer explicit DNSServers, then primary interface DNS.
		dns := c.Network.DNS
		if len(dns) == 0 {
			if iface := firstInterface(c.Network); iface != nil {
				dns = iface.DNS
			}
		}
		co.Nameservers = dns
	}

	// Like docker-compose, every service gets connectivity by default. If the
	// compose file specifies no usable interface, attach a DHCP eth0 on the
	// default bridge so the container is reachable.
	if len(co.Nets) == 0 && opts.DefaultBridge != "" {
		co.Nets = append(co.Nets, translateNet(0, common.NetworkInterface{DHCP: true}, opts.DefaultBridge))
	}

	return co, nil
}
