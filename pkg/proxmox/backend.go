// Package proxmox provides backends for managing Proxmox LXC containers.
//
// Unlike upstream LXC (lxc-start, lxc.* config keys), Proxmox manages
// containers via the `pct` CLI and the REST API, addressing them by numeric
// VMID and storing configuration in /etc/pve/lxc/<vmid>.conf using Proxmox's
// own key format (e.g. net0:, memory:, rootfs:). This package translates the
// compose-style configuration into that model.
package proxmox

// Status represents the runtime status of a Proxmox container.
type Status string

const (
	StatusRunning Status = "running"
	StatusStopped Status = "stopped"
	// StatusPaused maps to Proxmox's frozen/suspended state.
	StatusPaused  Status = "paused"
	StatusUnknown Status = "unknown"
)

// ContainerInfo describes a Proxmox LXC container.
type ContainerInfo struct {
	VMID   int    `json:"vmid"`
	Name   string `json:"name"`
	Status Status `json:"status"`
}

// Backend abstracts a Proxmox container management mechanism (pct CLI today,
// REST API in the future). All operations address containers by VMID.
type Backend interface {
	// Create provisions a new container from the given options.
	Create(vmid int, opts CreateOptions) error
	// Start boots a stopped container.
	Start(vmid int) error
	// Stop forcibly stops a container.
	Stop(vmid int) error
	// Shutdown gracefully shuts down a container.
	Shutdown(vmid int) error
	// Suspend pauses (freezes) a running container.
	Suspend(vmid int) error
	// Resume unfreezes a suspended container.
	Resume(vmid int) error
	// Destroy removes a container and its disks.
	Destroy(vmid int) error
	// Status returns the runtime status of a container.
	Status(vmid int) (Status, error)
	// List returns all containers known to the backend.
	List() ([]ContainerInfo, error)
	// SetInitCommand sets the container's init command (used to reproduce a
	// converted OCI image's entrypoint/command under LXC). Empty path is a
	// no-op.
	SetInitCommand(vmid int, initPath string) error
}

// CreateOptions holds the fully-translated parameters for provisioning a
// Proxmox LXC container. Fields map directly onto pct create options.
type CreateOptions struct {
	// Hostname is the container hostname (derived from the service name).
	Hostname string
	// OSTemplate is the Proxmox template volid, e.g.
	// "local:vztmpl/alpine-3.19-default_20240207_amd64.tar.xz".
	OSTemplate string
	// Storage is the target storage ID for the rootfs, e.g. "local-lvm".
	Storage string
	// RootFSSize is the rootfs size in gibibytes.
	RootFSSize int
	// Cores is the number of CPU cores. Zero means unset (Proxmox default).
	Cores int
	// CPULimit corresponds to pct --cpulimit (0 = unlimited).
	CPULimit int
	// CPUUnits corresponds to pct --cpuunits (cgroup CPU weight).
	CPUUnits int
	// MemoryMB is the memory limit in mebibytes.
	MemoryMB int
	// SwapMB is the swap limit in mebibytes.
	SwapMB int
	// Unprivileged marks the container as unprivileged.
	Unprivileged bool
	// Features is the pct --features value, e.g. "nesting=1".
	Features string
	// Nameservers are DNS servers (space-separated for pct --nameserver).
	Nameservers []string
	// SearchDomain is the DNS search domain.
	SearchDomain string
	// Nets holds translated network interface strings (pct --netN values).
	Nets []string
	// Mounts holds translated mountpoint strings (pct --mpN values).
	Mounts []string
	// Start indicates whether to start the container immediately after create.
	Start bool
	// Extra holds any additional raw key/value pct options.
	Extra map[string]string
}
