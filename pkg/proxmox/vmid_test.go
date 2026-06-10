package proxmox

import (
	"path/filepath"
	"testing"
)

func TestVMIDAutoAllocation(t *testing.T) {
	path := filepath.Join(t.TempDir(), "vmids.json")
	store, err := NewVMIDStore(path, 100)
	if err != nil {
		t.Fatal(err)
	}

	web, err := store.Assign("web", nil)
	if err != nil {
		t.Fatal(err)
	}
	if web != 100 {
		t.Errorf("web vmid = %d, want 100", web)
	}

	db, err := store.Assign("db", nil)
	if err != nil {
		t.Fatal(err)
	}
	if db != 101 {
		t.Errorf("db vmid = %d, want 101", db)
	}

	// Re-assigning an existing name returns the same id.
	again, _ := store.Assign("web", nil)
	if again != 100 {
		t.Errorf("re-assign web = %d, want 100", again)
	}
}

func TestVMIDAvoidsInUse(t *testing.T) {
	path := filepath.Join(t.TempDir(), "vmids.json")
	store, _ := NewVMIDStore(path, 100)

	// 100 and 101 already exist on the node.
	id, err := store.Assign("web", []int{100, 101})
	if err != nil {
		t.Fatal(err)
	}
	if id != 102 {
		t.Errorf("vmid = %d, want 102 (avoiding in-use)", id)
	}
}

func TestVMIDPersistence(t *testing.T) {
	path := filepath.Join(t.TempDir(), "vmids.json")
	store, _ := NewVMIDStore(path, 100)
	if _, err := store.Assign("web", nil); err != nil {
		t.Fatal(err)
	}

	// Reload from disk.
	store2, err := NewVMIDStore(path, 100)
	if err != nil {
		t.Fatal(err)
	}
	vmid, ok := store2.Get("web")
	if !ok || vmid != 100 {
		t.Errorf("after reload web = (%d, %v), want (100, true)", vmid, ok)
	}
}

func TestVMIDLookupAndRemove(t *testing.T) {
	path := filepath.Join(t.TempDir(), "vmids.json")
	store, _ := NewVMIDStore(path, 100)
	vmid, _ := store.Assign("web", nil)

	name, ok := store.Lookup(vmid)
	if !ok || name != "web" {
		t.Errorf("Lookup(%d) = (%q, %v)", vmid, name, ok)
	}

	if err := store.Remove("web"); err != nil {
		t.Fatal(err)
	}
	if _, ok := store.Get("web"); ok {
		t.Error("expected web to be removed")
	}
}
