package oci

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/larkinwc/proxmox-lxc-compose/pkg/logging"
	"github.com/larkinwc/proxmox-lxc-compose/pkg/testutil"
)

func setupRegistryTest(t *testing.T) (*RegistryManager, *testutil.MockCommandExecutor, string, func()) {
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

	// Create mock command executor
	mockCmd := testutil.NewMockCommandExecutor()
	oldExecCommand := execCommand
	execCommand = mockCmd.Command

	// Set up basic mock commands
	mockImageData := []byte("mock image data")
	mockCmd.AddMockCommand("docker pull docker.io/library/alpine:latest", nil)
	mockCmd.AddMockCommand("docker save docker.io/library/alpine:latest", mockImageData)
	mockCmd.AddMockCommand("docker load", nil)
	mockCmd.AddMockCommand("docker push docker.io/library/alpine:latest", nil)
	mockCmd.AddMockCommand("docker inspect docker.io/library/alpine:latest", nil)
	mockCmd.AddMockCommand("docker rmi docker.io/library/alpine:latest", nil)

	// Create manager with short cleanup interval
	manager, err := NewRegistryManager(filepath.Join(tmpDir, "images"))
	if err != nil {
		t.Fatal(err)
	}

	cleanup := func() {
		manager.Stop()
		execCommand = oldExecCommand
		os.RemoveAll(tmpDir)
	}

	return manager, mockCmd, tmpDir, cleanup
}

func TestRegistryManager(t *testing.T) {
	manager, _, _, cleanup := setupRegistryTest(t)
	defer cleanup()

	// Set short TTL for testing
	manager.store.SetCacheTTL(1) // 1 second TTL
	ctx := context.Background()

	testRef := ImageReference{
		Registry:   "docker.io",
		Repository: "library/alpine",
		Tag:        "latest",
	}

	t.Run("basic_operations", func(t *testing.T) {
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
	})

	t.Run("save_and_load", func(t *testing.T) {
		// Test Save
		if err := manager.Save(ctx, testRef); err != nil {
			t.Fatal(err)
		}

		// Test Load
		if err := manager.Load(ctx, testRef); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("push_operation", func(t *testing.T) {
		if err := manager.Push(ctx, testRef); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("auto_cleanup", func(t *testing.T) {
		// Pull image
		if err := manager.Pull(ctx, testRef); err != nil {
			t.Fatal(err)
		}

		// Wait for TTL and cleanup interval
		time.Sleep(2 * time.Second)

		// Verify cleanup happened
		images, err := manager.List(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if len(images) != 0 {
			t.Errorf("expected 0 images after automatic cleanup, got %d", len(images))
		}
	})
}

func TestRegistryErrors(t *testing.T) {
	manager, mockCmd, _, cleanup := setupRegistryTest(t)
	defer cleanup()

	ctx := context.Background()
	testRef := ImageReference{
		Registry:   "docker.io",
		Repository: "library/alpine",
		Tag:        "latest",
	}

	t.Run("pull_errors", func(t *testing.T) {
		mockCmd.AddErrorCommand("docker pull docker.io/library/alpine:latest", "failed to pull")
		if err := manager.Pull(ctx, testRef); err == nil {
			t.Error("expected error on pull failure")
		}
	})

	t.Run("push_errors", func(t *testing.T) {
		mockCmd.AddErrorCommand("docker push docker.io/library/alpine:latest", "failed to push")
		if err := manager.Push(ctx, testRef); err == nil {
			t.Error("expected error on push failure")
		}
	})

	t.Run("invalid_reference", func(t *testing.T) {
		invalidRef := ImageReference{
			Registry: "",
			Tag:      "",
		}
		if err := manager.Pull(ctx, invalidRef); err == nil {
			t.Error("expected error with invalid reference")
		}
	})
}
