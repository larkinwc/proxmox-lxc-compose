package main

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/larkinwc/proxmox-lxc-compose/pkg/common"
	"github.com/larkinwc/proxmox-lxc-compose/pkg/proxmox"
)

// deploymentState records only successfully provisioned services. VMIDs remain
// in the existing mapping file so removing a missing CT never changes its ID.
type deploymentRecord struct {
	VMID     int    `json:"vmid"`
	Digest   string `json:"digest"`
	Hostname string `json:"hostname"`
}

type deploymentState struct {
	path     string
	services map[string]deploymentRecord
}

func loadDeploymentState() (*deploymentState, error) {
	s := &deploymentState{path: vmidStorePath + ".state.json", services: make(map[string]deploymentRecord)}
	data, err := os.ReadFile(s.path)
	if os.IsNotExist(err) {
		return s, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to read deployment state: %w", err)
	}
	if err := json.Unmarshal(data, &s.services); err != nil {
		return nil, fmt.Errorf("failed to parse deployment state: %w", err)
	}
	if s.services == nil {
		s.services = make(map[string]deploymentRecord)
	}
	return s, nil
}

func (s *deploymentState) save() error {
	data, err := json.MarshalIndent(s.services, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0700); err != nil {
		return err
	}
	f, err := os.CreateTemp(filepath.Dir(s.path), ".deployment-*")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	if _, err := f.Write(data); err != nil {
		f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	return os.Rename(f.Name(), s.path)
}

func (s *deploymentState) remove(name string) error {
	delete(s.services, name)
	return s.save()
}

// JSON sorts map keys; retaining the entire accepted service model ensures even
// fields not currently translated by pct cannot silently change on repeat up.
// The translated options capture only node defaults effective for this service.
func desiredConfigDigest(svc common.Container, opts proxmox.CreateOptions) (string, error) {
	data, err := json.Marshal(struct {
		Service       common.Container
		Options       proxmox.CreateOptions
		CommandSet    bool
		EntrypointSet bool
	}{svc, opts, svc.Command != nil, svc.Entrypoint != nil})
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", sha256.Sum256(data)), nil
}

func selectedServices(compose *common.ComposeConfig, args []string) ([]string, error) {
	services := append([]string(nil), args...)
	if len(services) == 0 {
		for name := range compose.Services {
			services = append(services, name)
		}
		sort.Strings(services)
	}
	seen := make(map[string]bool)
	selected := make([]string, 0, len(services))
	for _, name := range services {
		if _, ok := compose.Services[name]; !ok {
			return nil, fmt.Errorf("service '%s' not found in config", name)
		}
		if !seen[name] {
			selected = append(selected, name)
			seen[name] = true
		}
	}
	return selected, nil
}

func validateMappedContainer(name string, vmid int, info proxmox.ContainerInfo, record deploymentRecord, recorded bool, mappings map[string]int) error {
	for other, id := range mappings {
		if id == vmid && other != name {
			return fmt.Errorf("VMID %d is mapped to both '%s' and '%s'; refusing to modify container", vmid, name, other)
		}
	}
	hostname := name
	if recorded {
		if record.VMID != vmid {
			return fmt.Errorf("service '%s' VMID mapping %d differs from recorded VMID %d; refusing to modify container", name, vmid, record.VMID)
		}
		hostname = record.Hostname
	}
	if hostname == "" || info.Name != hostname {
		return fmt.Errorf("service '%s' maps to VMID %d with hostname %q, expected %q; refusing to modify unrelated container", name, vmid, info.Name, hostname)
	}
	return nil
}
