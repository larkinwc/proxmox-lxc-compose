package oci

import (
	"os"
	"path/filepath"
	"testing"

	"proxmox-lxc-compose/pkg/logging"
)

func TestLocalImageStore(t *testing.T) {
	// Initialize logger for tests
	err := logging.Init(logging.Config{
		Level:       "debug",
		Development: true,
	})
	if err != nil {
		t.Fatal(err)
	}

	// Create temp directory for tests
	tmpDir, err := os.MkdirTemp("", "lxc-compose-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create the image storage directory
	err = os.MkdirAll(filepath.Join(tmpDir, "images"), 0755)
	if err != nil {
		t.Fatal(err)
	}

	store, err := NewLocalImageStore(filepath.Join(tmpDir, "images"))
	if err != nil {
		t.Fatal(err)
	}

	testRef := ImageReference{
		Registry:   "docker.io",
		Repository: "library/ubuntu",
		Tag:        "latest",
	}

	// Test storing an image
	testData := []byte("mock image data")
	if err := store.Store(testRef, testData); err != nil {
		t.Errorf("Store() error = %v", err)
	}

	// Test retrieving an image
	data, err := store.Retrieve(testRef)
	if err != nil {
		t.Errorf("Retrieve() error = %v", err)
	}
	if string(data) != string(testData) {
		t.Errorf("Retrieved data does not match stored data")
	}

	// Test listing images
	refs, err := store.List()
	if err != nil {
		t.Errorf("List() error = %v", err)
	}
	if len(refs) != 1 {
		t.Errorf("Expected 1 image, got %d", len(refs))
		t.FailNow() // Stop here to prevent index out of range panic
	}
	if refs[0].Repository != testRef.Repository {
		t.Errorf("Listed image does not match stored image")
	}

	// Test removing an image
	if err := store.Remove(testRef); err != nil {
		t.Errorf("Remove() error = %v", err)
	}

	// Verify image was removed
	refs, err = store.List()
	if err != nil {
		t.Errorf("List() error = %v", err)
	}
	if len(refs) != 0 {
		t.Errorf("Expected 0 images after removal, got %d", len(refs))
	}
}
