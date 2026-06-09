package proxmox

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// DefaultVMIDBase is the lowest VMID auto-allocation will assign. Proxmox
// reserves IDs below 100 for internal use.
const DefaultVMIDBase = 100

// VMIDStore persists a mapping between compose service names and the numeric
// Proxmox VMIDs they were allocated. It is safe for concurrent use.
type VMIDStore struct {
	path string
	mu   sync.Mutex
	base int
	// mapping is service name -> vmid.
	mapping map[string]int
}

// NewVMIDStore loads (or initializes) a VMID mapping stored at path. The base
// controls the lowest VMID that auto-allocation will assign.
func NewVMIDStore(path string, base int) (*VMIDStore, error) {
	if base <= 0 {
		base = DefaultVMIDBase
	}
	s := &VMIDStore{
		path:    path,
		base:    base,
		mapping: make(map[string]int),
	}
	if err := s.load(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *VMIDStore) load() error {
	data, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("failed to read vmid store: %w", err)
	}
	if len(data) == 0 {
		return nil
	}
	if err := json.Unmarshal(data, &s.mapping); err != nil {
		return fmt.Errorf("invalid vmid store format: %w", err)
	}
	return nil
}

func (s *VMIDStore) save() error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0755); err != nil {
		return fmt.Errorf("failed to create vmid store directory: %w", err)
	}
	data, err := json.MarshalIndent(s.mapping, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal vmid store: %w", err)
	}
	if err := os.WriteFile(s.path, data, 0644); err != nil {
		return fmt.Errorf("failed to write vmid store: %w", err)
	}
	return nil
}

// Get returns the VMID mapped to a service name, if one exists.
func (s *VMIDStore) Get(name string) (int, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	vmid, ok := s.mapping[name]
	return vmid, ok
}

// Lookup returns the service name mapped to a VMID, if one exists.
func (s *VMIDStore) Lookup(vmid int) (string, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for name, id := range s.mapping {
		if id == vmid {
			return name, true
		}
	}
	return "", false
}

// All returns a copy of the full name->vmid mapping.
func (s *VMIDStore) All() map[string]int {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make(map[string]int, len(s.mapping))
	for k, v := range s.mapping {
		out[k] = v
	}
	return out
}

// Remove deletes a service's mapping and persists the change.
func (s *VMIDStore) Remove(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.mapping, name)
	return s.save()
}

// Assign returns the VMID for a service, allocating a new one if necessary.
// inUse lists VMIDs already present on the node (e.g. from `pct list`) so that
// allocation never collides with containers the store doesn't know about.
func (s *VMIDStore) Assign(name string, inUse []int) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if vmid, ok := s.mapping[name]; ok {
		return vmid, nil
	}

	used := make(map[int]bool)
	for _, id := range inUse {
		used[id] = true
	}
	for _, id := range s.mapping {
		used[id] = true
	}

	// Find the next free VMID at or above the base. Allocation is
	// deterministic: always the lowest free ID.
	candidate := s.base
	for used[candidate] {
		candidate++
	}

	s.mapping[name] = candidate
	if err := s.save(); err != nil {
		delete(s.mapping, name)
		return 0, err
	}
	return candidate, nil
}
