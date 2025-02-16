package oci

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/larkinwc/proxmox-lxc-compose/pkg/logging"
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

	tmpDir := t.TempDir()
	store, err := NewLocalImageStore(filepath.Join(tmpDir, "images"))
	if err != nil {
		t.Fatal(err)
	}

	testRef := ImageReference{
		Registry:   "docker.io",
		Repository: "library/ubuntu",
		Tag:        "latest",
	}

	t.Run("basic_operations", func(t *testing.T) {
		// Test storing an image
		testData := []byte("mock image data")
		if err := store.Store(testRef, testData); err != nil {
			t.Fatal(err)
		}

		// Test retrieving an image
		data, err := store.Get(testRef)
		if err != nil {
			t.Fatal(err)
		}
		if string(data) != string(testData) {
			t.Error("retrieved data does not match stored data")
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

		// Test removing an image
		if err := store.Remove(testRef); err != nil {
			t.Fatal(err)
		}

		// Verify removal
		if _, err := store.Get(testRef); err == nil {
			t.Error("expected error getting removed image")
		}
	})

	t.Run("cache_operations", func(t *testing.T) {
		store.SetCacheTTL(1) // 1 second TTL
		testData := []byte("cache test data")

		// Store image
		if err := store.Store(testRef, testData); err != nil {
			t.Fatal(err)
		}

		// Verify immediate retrieval
		if _, err := store.Get(testRef); err != nil {
			t.Error("failed to get cached image")
		}

		// Wait for TTL to expire
		time.Sleep(2 * time.Second)

		// Verify image is expired
		if _, err := store.Get(testRef); err == nil {
			t.Error("expected error getting expired image")
		}
	})

	t.Run("error_handling", func(t *testing.T) {
		// Test invalid reference
		invalidRef := ImageReference{}
		if err := store.Store(invalidRef, []byte{}); err == nil {
			t.Error("expected error storing with invalid reference")
		}

		// Test storing in non-existent directory
		badStore, _ := NewLocalImageStore("/nonexistent/path")
		if err := badStore.Store(testRef, []byte{}); err == nil {
			t.Error("expected error storing to invalid path")
		}

		// Test corrupted metadata
		metadataPath := filepath.Join(store.rootDir, "metadata.json")
		if err := os.WriteFile(metadataPath, []byte("invalid json"), 0644); err != nil {
			t.Fatal(err)
		}
		if _, err := store.List(); err == nil {
			t.Error("expected error with corrupted metadata")
		}
	})
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
