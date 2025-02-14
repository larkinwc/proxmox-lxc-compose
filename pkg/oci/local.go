package oci

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"proxmox-lxc-compose/pkg/errors"
	"proxmox-lxc-compose/pkg/logging"
)

type LocalImageStore struct {
	storageDir string
}

// NewLocalImageStore creates a new local image store
func NewLocalImageStore(storageDir string) (*LocalImageStore, error) {
	if err := os.MkdirAll(storageDir, 0755); err != nil {
		return nil, errors.Wrap(err, errors.ErrStorage, "failed to create storage directory").
			WithDetails(map[string]interface{}{
				"path": storageDir,
			})
	}
	logging.Debug("Created image storage directory", "path", storageDir)
	return &LocalImageStore{storageDir: storageDir}, nil
}

func (s *LocalImageStore) imagePath(ref ImageReference) string {
	// Create a unique filename based on the image reference
	// Replace slashes with underscores to avoid invalid paths
	repo := strings.ReplaceAll(ref.Repository, "/", "_")
	name := fmt.Sprintf("%s_%s_%s", ref.Registry, repo, ref.Tag)
	if ref.Digest != "" {
		name = fmt.Sprintf("%s@%s", name, ref.Digest)
	}
	return filepath.Join(s.storageDir, name)
}

func (s *LocalImageStore) Store(ref ImageReference, data []byte) error {
	path := s.imagePath(ref)
	logging.Debug("Storing image",
		"path", path,
		"size", len(data))

	// Create parent directories if they don't exist
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return errors.Wrap(err, errors.ErrStorage, "failed to create image directory").
			WithDetails(map[string]interface{}{
				"path": dir,
			})
	}

	// Store image data
	if err := os.WriteFile(path+".tar", data, 0644); err != nil {
		return errors.Wrap(err, errors.ErrStorage, "failed to write image data").
			WithDetails(map[string]interface{}{
				"path": path + ".tar",
				"size": len(data),
			})
	}

	// Store metadata
	metadata, err := json.Marshal(ref)
	if err != nil {
		return errors.Wrap(err, errors.ErrInternal, "failed to marshal metadata")
	}

	if err := os.WriteFile(path+".json", metadata, 0644); err != nil {
		return errors.Wrap(err, errors.ErrStorage, "failed to write metadata").
			WithDetails(map[string]interface{}{
				"path": path + ".json",
			})
	}

	logging.Info("Successfully stored image",
		"registry", ref.Registry,
		"repository", ref.Repository,
		"tag", ref.Tag)
	return nil
}

func (s *LocalImageStore) Retrieve(ref ImageReference) ([]byte, error) {
	path := s.imagePath(ref)
	logging.Debug("Retrieving image", "path", path)

	data, err := os.ReadFile(path + ".tar")
	if err != nil {
		if os.IsNotExist(err) {
			return nil, errors.New(errors.ErrStorage, "image not found in local storage").
				WithDetails(map[string]interface{}{
					"path": path + ".tar",
				})
		}
		return nil, errors.Wrap(err, errors.ErrStorage, "failed to read image data").
			WithDetails(map[string]interface{}{
				"path": path + ".tar",
			})
	}

	logging.Debug("Successfully retrieved image",
		"path", path,
		"size", len(data))
	return data, nil
}

func (s *LocalImageStore) Remove(ref ImageReference) error {
	path := s.imagePath(ref)
	logging.Debug("Removing image", "path", path)

	if err := os.Remove(path + ".tar"); err != nil && !os.IsNotExist(err) {
		return errors.Wrap(err, errors.ErrStorage, "failed to remove image data").
			WithDetails(map[string]interface{}{
				"path": path + ".tar",
			})
	}

	if err := os.Remove(path + ".json"); err != nil && !os.IsNotExist(err) {
		return errors.Wrap(err, errors.ErrStorage, "failed to remove metadata").
			WithDetails(map[string]interface{}{
				"path": path + ".json",
			})
	}

	logging.Info("Successfully removed image",
		"registry", ref.Registry,
		"repository", ref.Repository,
		"tag", ref.Tag)
	return nil
}

func (s *LocalImageStore) List() ([]ImageReference, error) {
	logging.Debug("Listing images", "path", s.storageDir)

	entries, err := os.ReadDir(s.storageDir)
	if err != nil {
		return nil, errors.Wrap(err, errors.ErrStorage, "failed to read storage directory").
			WithDetails(map[string]interface{}{
				"path": s.storageDir,
			})
	}

	var refs []ImageReference
	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".json" {
			data, err := os.ReadFile(filepath.Join(s.storageDir, entry.Name()))
			if err != nil {
				logging.Warn("Failed to read metadata file",
					"file", entry.Name(),
					"error", err)
				continue
			}

			var ref ImageReference
			if err := json.Unmarshal(data, &ref); err != nil {
				logging.Warn("Failed to unmarshal metadata",
					"file", entry.Name(),
					"error", err)
				continue
			}
			refs = append(refs, ref)
		}
	}

	logging.Debug("Found images", "count", len(refs))
	return refs, nil
}
