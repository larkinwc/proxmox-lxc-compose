package oci

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"proxmox-lxc-compose/pkg/logging"
	"proxmox-lxc-compose/pkg/testutil"
)

func TestRegistryManager(t *testing.T) {
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

	// Create mock command executor
	mockCmd := testutil.NewMockCommandExecutor()
	oldExecCommand := execCommand // Use the one from the main package
	defer func() { execCommand = oldExecCommand }()
	execCommand = mockCmd.Command

	// Mock docker commands
	mockImageData := []byte("mock image data")
	mockCmd.AddMockCommand("docker pull docker.io/library/alpine:latest", nil)
	mockCmd.AddMockCommand("docker save docker.io/library/alpine:latest", mockImageData)
	mockCmd.AddMockCommand("docker load", nil)
	mockCmd.AddMockCommand("docker push docker.io/library/alpine:latest", nil)
	mockCmd.AddMockCommand("docker inspect docker.io/library/alpine:latest", nil)
	mockCmd.AddMockCommand("docker rmi docker.io/library/alpine:latest", nil)

	// Create manager with short cleanup interval for testing
	manager, err := NewRegistryManager(filepath.Join(tmpDir, "images"))
	if err != nil {
		t.Fatal(err)
	}
	// Ensure cleanup goroutine is stopped
	defer manager.Stop()

	// Set short TTL for testing
	manager.store.SetCacheTTL(1) // 1 second TTL

	ctx := context.Background()
	testRef := ImageReference{
		Registry:   "docker.io",
		Repository: "library/alpine",
		Tag:        "latest",
	}

	// Test Pull
	if err := manager.Pull(ctx, testRef); err != nil {
		t.Fatal(err)
	}

	// Test List
	images, err := manager.List(ctx)
	if err != nil {
		t.Fatal(err)
	}

	if len(images) != 1 {
		t.Errorf("expected 1 image, got %d", len(images))
	}

	// Test Save (of already pulled image)
	if err := manager.Save(ctx, testRef); err != nil {
		t.Fatal(err)
	}

	// Test Load
	if err := manager.Load(ctx, testRef); err != nil {
		t.Fatal(err)
	}

	// Test Push
	if err := manager.Push(ctx, testRef); err != nil {
		t.Fatal(err)
	}

	// Test Delete
	if err := manager.Delete(ctx, testRef); err != nil {
		t.Fatal(err)
	}

	// Verify deletion
	images, err = manager.List(ctx)
	if err != nil {
		t.Fatal(err)
	}

	if len(images) != 0 {
		t.Errorf("expected 0 images after deletion, got %d", len(images))
	}

	// Test automatic cleanup
	// Wait for TTL and cleanup interval
	time.Sleep(2 * time.Second)

	// Verify cleanup happened
	images, err = manager.List(ctx)
	if err != nil {
		t.Fatal(err)
	}

	if len(images) != 0 {
		t.Errorf("expected 0 images after automatic cleanup, got %d", len(images))
	}
}
