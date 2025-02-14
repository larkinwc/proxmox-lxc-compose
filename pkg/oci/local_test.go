package oci

import (
	"os"
	"path/filepath"
	"testing"
	"time"

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

	// Test storing an image
	testData = []byte("mock image data")
	if err := store.Store(testRef, testData); err != nil {
		t.Fatal(err)
	}

	// Test listing images
	images, err := store.List()
	if err != nil {
		t.Fatal(err)
	}

	if len(images) != 1 {
		t.Errorf("expected 1 image, got %d", len(images))
	}

	if images[0].Registry != testRef.Registry {
		t.Errorf("expected registry %s, got %s", testRef.Registry, images[0].Registry)
	}

	if images[0].Repository != testRef.Repository {
		t.Errorf("expected repository %s, got %s", testRef.Repository, images[0].Repository)
	}

	if images[0].Tag != testRef.Tag {
		t.Errorf("expected tag %s, got %s", testRef.Tag, images[0].Tag)
	}

	// Test multiple images
	testRef2 := ImageReference{
		Registry:   "docker.io",
		Repository: "library/nginx",
		Tag:        "latest",
	}

	if err := store.Store(testRef2, testData); err != nil {
		t.Fatal(err)
	}

	images, err = store.List()
	if err != nil {
		t.Fatal(err)
	}

	if len(images) != 2 {
		t.Errorf("expected 2 images, got %d", len(images))
	}

	// Test corrupted metadata handling
	corruptedPath := filepath.Join(tmpDir, "images", "corrupted.json")
	if err := os.WriteFile(corruptedPath, []byte("invalid json"), 0644); err != nil {
		t.Fatal(err)
	}

	images, err = store.List()
	if err != nil {
		t.Fatal(err)
	}

	// Should still get valid images even with corrupted file
	if len(images) != 2 {
		t.Errorf("expected 2 images despite corrupted file, got %d", len(images))
	}
}

func TestImageCaching(t *testing.T) {
	err := logging.Init(logging.Config{
		Level:       "debug",
		Development: true,
	})
	if err != nil {
		t.Fatal(err)
	}

	tmpDir, err := os.MkdirTemp("", "lxc-compose-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	store, err := NewLocalImageStore(filepath.Join(tmpDir, "images"))
	if err != nil {
		t.Fatal(err)
	}

	// Set short TTL for testing
	store.SetCacheTTL(1) // 1 second TTL

	testRef := ImageReference{
		Registry:   "docker.io",
		Repository: "library/ubuntu",
		Tag:        "latest",
	}

	// Store test image
	cacheTestData := []byte("mock image data for cache test")
	if err := store.Store(testRef, cacheTestData); err != nil {
		t.Fatal(err)
	}

	// Verify image is listed
	images, err := store.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(images) != 1 {
		t.Errorf("expected 1 image, got %d", len(images))
	}

	// Wait for TTL to expire
	time.Sleep(2 * time.Second)

	// Clean expired images
	if err := store.CleanExpiredImages(); err != nil {
		t.Fatal(err)
	}

	// Verify image was removed
	images, err = store.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(images) != 0 {
		t.Errorf("expected 0 images after TTL expiry, got %d", len(images))
	}
}
