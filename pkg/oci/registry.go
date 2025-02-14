package oci

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"time"

	"proxmox-lxc-compose/pkg/errors"
	"proxmox-lxc-compose/pkg/internal/recovery"
	"proxmox-lxc-compose/pkg/logging"
)

// execCommand is a variable that allows us to mock exec.Command during tests
var execCommand = exec.Command

type RegistryManager struct {
	store *LocalImageStore
	// Add cleanup interval
	cleanupInterval time.Duration
	stopCleanup     chan struct{}
}

func NewRegistryManager(storageDir string) (*RegistryManager, error) {
	store, err := NewLocalImageStore(storageDir)
	if err != nil {
		return nil, err
	}

	manager := &RegistryManager{
		store:           store,
		cleanupInterval: 1 * time.Hour,
		stopCleanup:     make(chan struct{}),
	}

	// Start cleanup goroutine
	go manager.cleanupLoop()

	return manager, nil
}

func (m *RegistryManager) Pull(ctx context.Context, ref ImageReference) error {
	return recovery.RetryWithBackoff(ctx, recovery.DefaultRetryConfig, func() error {
		logging.Info("Pulling image",
			"registry", ref.Registry,
			"repository", ref.Repository,
			"tag", ref.Tag)

		// Use docker to pull the image
		pullCmd := execCommand("docker", "pull", formatDockerRef(ref))
		if out, err := pullCmd.CombinedOutput(); err != nil {
			return errors.Wrap(err, errors.ErrRegistry, "failed to pull image").
				WithDetails(map[string]interface{}{
					"output": string(out),
					"image":  formatDockerRef(ref),
				})
		}

		// Save the image to a tar
		logging.Debug("Saving image to tar")
		saveCmd := execCommand("docker", "save", formatDockerRef(ref))
		data, err := saveCmd.Output()
		if err != nil {
			return errors.Wrap(err, errors.ErrRegistry, "failed to save image")
		}

		// Store in local cache
		logging.Debug("Storing image in local cache")
		if err := m.store.Store(ref, data); err != nil {
			return errors.Wrap(err, errors.ErrStorage, "failed to store image in cache")
		}

		logging.Info("Successfully pulled and cached image",
			"image", formatDockerRef(ref))
		return nil
	})
}

func (m *RegistryManager) Push(ctx context.Context, ref ImageReference) error {
	return recovery.RetryWithBackoff(ctx, recovery.DefaultRetryConfig, func() error {
		logging.Info("Pushing image",
			"registry", ref.Registry,
			"repository", ref.Repository,
			"tag", ref.Tag)

		// Load from local cache
		logging.Debug("Loading image from cache")
		data, err := m.store.Retrieve(ref)
		if err != nil {
			return errors.Wrap(err, errors.ErrStorage, "failed to retrieve image from cache")
		}

		// Load into docker
		logging.Debug("Loading image into docker")
		loadCmd := execCommand("docker", "load")
		loadCmd.Stdin = bytes.NewReader(data)
		if out, err := loadCmd.CombinedOutput(); err != nil {
			return errors.Wrap(err, errors.ErrRegistry, "failed to load image").
				WithDetails(map[string]interface{}{
					"output": string(out),
					"image":  formatDockerRef(ref),
				})
		}

		// Push to registry
		logging.Debug("Pushing image to registry")
		pushCmd := execCommand("docker", "push", formatDockerRef(ref))
		if out, err := pushCmd.CombinedOutput(); err != nil {
			return errors.Wrap(err, errors.ErrRegistry, "failed to push image").
				WithDetails(map[string]interface{}{
					"output": string(out),
					"image":  formatDockerRef(ref),
				})
		}

		logging.Info("Successfully pushed image",
			"image", formatDockerRef(ref))
		return nil
	})
}

func (m *RegistryManager) Save(ctx context.Context, ref ImageReference) error {
	logging.Info("Saving image",
		"registry", ref.Registry,
		"repository", ref.Repository,
		"tag", ref.Tag)

	// Check if image exists in docker
	inspectCmd := execCommand("docker", "inspect", formatDockerRef(ref))
	if err := inspectCmd.Run(); err != nil {
		return errors.Wrap(err, errors.ErrRegistry, "image not found in docker")
	}

	// Save the image to a tar
	logging.Debug("Saving image to tar")
	saveCmd := execCommand("docker", "save", formatDockerRef(ref))
	data, err := saveCmd.Output()
	if err != nil {
		return errors.Wrap(err, errors.ErrRegistry, "failed to save image")
	}

	// Store in local cache
	logging.Debug("Storing image in local cache")
	if err := m.store.Store(ref, data); err != nil {
		return errors.Wrap(err, errors.ErrStorage, "failed to store image in cache")
	}

	logging.Info("Successfully saved and cached image",
		"image", formatDockerRef(ref))
	return nil
}

func (m *RegistryManager) Load(ctx context.Context, ref ImageReference) error {
	logging.Info("Loading image",
		"registry", ref.Registry,
		"repository", ref.Repository,
		"tag", ref.Tag)

	// Load from local cache
	logging.Debug("Loading image from cache")
	data, err := m.store.Retrieve(ref)
	if err != nil {
		return errors.Wrap(err, errors.ErrStorage, "failed to retrieve image from cache")
	}

	// Load into docker
	logging.Debug("Loading image into docker")
	loadCmd := execCommand("docker", "load")
	loadCmd.Stdin = bytes.NewReader(data)
	if out, err := loadCmd.CombinedOutput(); err != nil {
		return errors.Wrap(err, errors.ErrRegistry, "failed to load image").
			WithDetails(map[string]interface{}{
				"output": string(out),
				"image":  formatDockerRef(ref),
			})
	}

	logging.Info("Successfully loaded image",
		"image", formatDockerRef(ref))
	return nil
}

func (m *RegistryManager) List(ctx context.Context) ([]ImageReference, error) {
	logging.Info("Listing images")

	images, err := m.store.List()
	if err != nil {
		return nil, errors.Wrap(err, errors.ErrStorage, "failed to list images")
	}

	logging.Info("Successfully listed images")
	return images, nil
}

func (m *RegistryManager) Delete(ctx context.Context, ref ImageReference) error {
	logging.Info("Deleting image",
		"registry", ref.Registry,
		"repository", ref.Repository,
		"tag", ref.Tag)

	// Remove from docker
	rmiCmd := execCommand("docker", "rmi", formatDockerRef(ref))
	if out, err := rmiCmd.CombinedOutput(); err != nil {
		return errors.Wrap(err, errors.ErrRegistry, "failed to remove docker image").
			WithDetails(map[string]interface{}{
				"output": string(out),
				"image":  formatDockerRef(ref),
			})
	}

	// Remove from local storage
	logging.Debug("Removing image from local storage")
	if err := m.store.Remove(ref); err != nil {
		return errors.Wrap(err, errors.ErrStorage, "failed to remove image from cache")
	}

	logging.Info("Successfully deleted image",
		"image", formatDockerRef(ref))
	return nil
}

func (m *RegistryManager) cleanupLoop() {
	ticker := time.NewTicker(m.cleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := m.store.CleanExpiredImages(); err != nil {
				logging.Error("Failed to clean expired images", "error", err)
			}
		case <-m.stopCleanup:
			return
		}
	}
}

// Stop cleans up resources and stops the cleanup goroutine
func (m *RegistryManager) Stop() {
	close(m.stopCleanup)
}

func formatDockerRef(ref ImageReference) string {
	if ref.Digest != "" {
		return fmt.Sprintf("%s/%s@%s", ref.Registry, ref.Repository, ref.Digest)
	}
	return fmt.Sprintf("%s/%s:%s", ref.Registry, ref.Repository, ref.Tag)
}
