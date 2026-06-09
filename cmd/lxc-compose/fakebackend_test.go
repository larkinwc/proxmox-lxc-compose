package main

import (
	"fmt"
	"sort"

	"github.com/larkinwc/proxmox-lxc-compose/pkg/proxmox"
)

// fakeBackend is an in-memory proxmox.Backend used by CLI tests so they never
// shell out to the real pct binary.
type fakeBackend struct {
	created  map[int]proxmox.CreateOptions
	status   map[int]proxmox.Status
	names    map[int]string
	initCmds map[int]string
}

func newFakeBackend() *fakeBackend {
	return &fakeBackend{
		created:  make(map[int]proxmox.CreateOptions),
		status:   make(map[int]proxmox.Status),
		names:    make(map[int]string),
		initCmds: make(map[int]string),
	}
}

func (f *fakeBackend) SetInitCommand(vmid int, initPath string) error {
	if err := f.requireExists(vmid); err != nil {
		return err
	}
	f.initCmds[vmid] = initPath
	return nil
}

func (f *fakeBackend) Create(vmid int, opts proxmox.CreateOptions) error {
	if _, exists := f.created[vmid]; exists {
		return fmt.Errorf("vmid %d already exists", vmid)
	}
	f.created[vmid] = opts
	f.status[vmid] = proxmox.StatusStopped
	f.names[vmid] = opts.Hostname
	return nil
}

func (f *fakeBackend) requireExists(vmid int) error {
	if _, ok := f.status[vmid]; !ok {
		return fmt.Errorf("vmid %d does not exist", vmid)
	}
	return nil
}

func (f *fakeBackend) Start(vmid int) error {
	if err := f.requireExists(vmid); err != nil {
		return err
	}
	f.status[vmid] = proxmox.StatusRunning
	return nil
}

func (f *fakeBackend) Stop(vmid int) error {
	if err := f.requireExists(vmid); err != nil {
		return err
	}
	f.status[vmid] = proxmox.StatusStopped
	return nil
}

func (f *fakeBackend) Shutdown(vmid int) error { return f.Stop(vmid) }

func (f *fakeBackend) Suspend(vmid int) error {
	if err := f.requireExists(vmid); err != nil {
		return err
	}
	f.status[vmid] = proxmox.StatusPaused
	return nil
}

func (f *fakeBackend) Resume(vmid int) error {
	if err := f.requireExists(vmid); err != nil {
		return err
	}
	f.status[vmid] = proxmox.StatusRunning
	return nil
}

func (f *fakeBackend) Destroy(vmid int) error {
	if err := f.requireExists(vmid); err != nil {
		return err
	}
	delete(f.created, vmid)
	delete(f.status, vmid)
	delete(f.names, vmid)
	delete(f.initCmds, vmid)
	return nil
}

func (f *fakeBackend) Status(vmid int) (proxmox.Status, error) {
	if err := f.requireExists(vmid); err != nil {
		return proxmox.StatusUnknown, err
	}
	return f.status[vmid], nil
}

func (f *fakeBackend) List() ([]proxmox.ContainerInfo, error) {
	var ids []int
	for id := range f.status {
		ids = append(ids, id)
	}
	sort.Ints(ids)
	infos := make([]proxmox.ContainerInfo, 0, len(ids))
	for _, id := range ids {
		infos = append(infos, proxmox.ContainerInfo{
			VMID:   id,
			Name:   f.names[id],
			Status: f.status[id],
		})
	}
	return infos, nil
}
