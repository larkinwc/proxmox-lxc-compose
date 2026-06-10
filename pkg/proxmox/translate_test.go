package proxmox

import (
	"strings"
	"testing"

	"github.com/larkinwc/proxmox-lxc-compose/pkg/common"
)

func intPtr(i int) *int     { return &i }
func i64Ptr(i int64) *int64 { return &i }

func TestParseSizeToMB(t *testing.T) {
	cases := map[string]int{
		"512M": 512,
		"2G":   2048,
		"1024": 1024, // bare number = MB
		"1G":   1024,
		"512":  512,
	}
	for in, want := range cases {
		got, err := parseSizeToMB(in)
		if err != nil {
			t.Fatalf("parseSizeToMB(%q) error: %v", in, err)
		}
		if got != want {
			t.Errorf("parseSizeToMB(%q) = %d, want %d", in, got, want)
		}
	}

	if _, err := parseSizeToMB("nonsense"); err == nil {
		t.Error("expected error for invalid size")
	}
}

func TestParseSizeToGB(t *testing.T) {
	cases := map[string]int{
		"2G":    2,
		"10G":   10,
		"1536M": 2, // rounds up
		"500M":  1, // rounds up to minimum 1
	}
	for in, want := range cases {
		got, err := parseSizeToGB(in)
		if err != nil {
			t.Fatalf("parseSizeToGB(%q) error: %v", in, err)
		}
		if got != want {
			t.Errorf("parseSizeToGB(%q) = %d, want %d", in, got, want)
		}
	}
}

func TestTranslateFull(t *testing.T) {
	c := &common.Container{
		Image: "alpine:3.19",
		CPU: &common.CPUConfig{
			Cores:  intPtr(2),
			Shares: i64Ptr(1024),
		},
		Memory: &common.MemoryConfig{
			Limit: "2G",
			Swap:  "1G",
		},
		Storage: &common.StorageConfig{
			Pool: "local-lvm",
			Root: "8G",
		},
		Security: &common.SecurityConfig{
			Isolation: "strict",
		},
		Network: &common.NetworkConfig{
			Interfaces: []common.NetworkInterface{
				{Type: "veth", Bridge: "vmbr0", IP: "192.168.1.10/24", Gateway: "192.168.1.1", DNS: []string{"1.1.1.1"}},
			},
		},
	}

	co, err := Translate(c, TranslateOptions{
		Hostname:        "web",
		DefaultStorage:  "local",
		DefaultBridge:   "vmbr0",
		DefaultRootFSGB: 8,
	})
	if err != nil {
		t.Fatalf("Translate error: %v", err)
	}

	if co.Hostname != "web" {
		t.Errorf("hostname = %q", co.Hostname)
	}
	if co.Storage != "local-lvm" {
		t.Errorf("storage = %q, want local-lvm", co.Storage)
	}
	if co.RootFSSize != 8 {
		t.Errorf("rootfs = %d, want 8", co.RootFSSize)
	}
	if co.Cores != 2 {
		t.Errorf("cores = %d, want 2", co.Cores)
	}
	if co.CPUUnits != 1024 {
		t.Errorf("cpuunits = %d, want 1024", co.CPUUnits)
	}
	if co.MemoryMB != 2048 {
		t.Errorf("memory = %d, want 2048", co.MemoryMB)
	}
	if co.SwapMB != 1024 {
		t.Errorf("swap = %d, want 1024", co.SwapMB)
	}
	if !co.Unprivileged {
		t.Error("expected unprivileged for strict isolation")
	}
	if len(co.Nets) != 1 {
		t.Fatalf("expected 1 net, got %d", len(co.Nets))
	}
	net := co.Nets[0]
	for _, want := range []string{"name=eth0", "bridge=vmbr0", "ip=192.168.1.10/24", "gw=192.168.1.1"} {
		if !strings.Contains(net, want) {
			t.Errorf("net %q missing %q", net, want)
		}
	}
	if len(co.Nameservers) != 1 || co.Nameservers[0] != "1.1.1.1" {
		t.Errorf("nameservers = %v", co.Nameservers)
	}
}

func TestTranslatePrivileged(t *testing.T) {
	c := &common.Container{
		Image:    "ubuntu:22.04",
		Security: &common.SecurityConfig{Isolation: "privileged", Privileged: true},
	}
	co, err := Translate(c, TranslateOptions{Hostname: "p", DefaultStorage: "local", DefaultRootFSGB: 4})
	if err != nil {
		t.Fatal(err)
	}
	if co.Unprivileged {
		t.Error("expected privileged container")
	}
}

func TestTranslateLegacyNetwork(t *testing.T) {
	c := &common.Container{
		Image: "alpine:3.19",
		Network: &common.NetworkConfig{
			Type:   "bridge",
			Bridge: "vmbr0",
			DHCP:   true,
		},
	}
	co, err := Translate(c, TranslateOptions{Hostname: "l", DefaultStorage: "local", DefaultRootFSGB: 4})
	if err != nil {
		t.Fatal(err)
	}
	if len(co.Nets) != 1 {
		t.Fatalf("expected 1 net from legacy config, got %d", len(co.Nets))
	}
	if !strings.Contains(co.Nets[0], "ip=dhcp") {
		t.Errorf("expected dhcp in %q", co.Nets[0])
	}
}

func TestTranslateDefaultNetwork(t *testing.T) {
	// No network section: like docker-compose, the container should still get
	// a DHCP eth0 on the default bridge.
	c := &common.Container{Image: "nginx:alpine"}
	co, err := Translate(c, TranslateOptions{
		Hostname:        "web",
		DefaultStorage:  "local-lvm",
		DefaultBridge:   "vmbr0",
		DefaultRootFSGB: 2,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(co.Nets) != 1 {
		t.Fatalf("expected 1 default net, got %d", len(co.Nets))
	}
	if !strings.Contains(co.Nets[0], "name=eth0") ||
		!strings.Contains(co.Nets[0], "bridge=vmbr0") ||
		!strings.Contains(co.Nets[0], "ip=dhcp") {
		t.Errorf("default net = %q, want eth0/vmbr0/dhcp", co.Nets[0])
	}
}

func TestTranslateNoDefaultNetworkWithoutBridge(t *testing.T) {
	// Without a default bridge there's nothing to attach to; leave Nets empty.
	c := &common.Container{Image: "nginx:alpine"}
	co, err := Translate(c, TranslateOptions{Hostname: "web", DefaultStorage: "local", DefaultRootFSGB: 2})
	if err != nil {
		t.Fatal(err)
	}
	if len(co.Nets) != 0 {
		t.Errorf("expected no nets without default bridge, got %d", len(co.Nets))
	}
}

func TestTranslateDefaultsTemplateFromImage(t *testing.T) {
	c := &common.Container{Image: "docker.io/library/alpine:3.19"}
	co, err := Translate(c, TranslateOptions{Hostname: "x", DefaultStorage: "local", DefaultRootFSGB: 4})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(co.OSTemplate, "local:vztmpl/alpine-3.19") {
		t.Errorf("derived template = %q", co.OSTemplate)
	}
}
