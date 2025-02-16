package oci

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type ImageMetadata struct {
	ImageReference
	StoredAt int64 `json:"stored_at"`
}

type cachedImage struct {
	Data      []byte
	Timestamp int64
}

// LocalImageStore represents a local OCI image store
type LocalImageStore struct {
	rootDir string
	mu      sync.RWMutex
	cache   map[string]*cachedImage
	ttl     int64
}

// NewLocalImageStore creates a new local image store
func NewLocalImageStore(rootDir string) (*LocalImageStore, error) {
	if err := os.MkdirAll(rootDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create image store directory: %w", err)
	}
	return &LocalImageStore{
		rootDir: rootDir,
		cache:   make(map[string]*cachedImage),
		ttl:     86400, // 24 hours default TTL
	}, nil
}

// Get retrieves an image from local storage
func (s *LocalImageStore) Get(ref ImageReference) ([]byte, error) {
	if s == nil {
		return nil, fmt.Errorf("store is nil")
	}
	if err := validateReference(ref); err != nil {
		return nil, err
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	// Check cache first
	if s.cache != nil && s.ttl > 0 {
		key := ref.String()
		if cached, ok := s.cache[key]; ok {
			if time.Now().Unix()-cached.Timestamp < s.ttl {
				return cached.Data, nil
			}
			delete(s.cache, key)
			return nil, fmt.Errorf("image expired: %s", ref.String())
		}
	}

	// Read from disk
	path := s.getImagePath(ref)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("image not found: %s", ref.String())
		}
		return nil, err
	}

	// Update cache
	if s.cache != nil && s.ttl > 0 {
		s.cache[ref.String()] = &cachedImage{
			Data:      data,
			Timestamp: time.Now().Unix(),
		}
	}

	return data, nil
}

// Store stores an image in local storage
func (s *LocalImageStore) Store(ref ImageReference, data []byte) error {
	if s == nil {
		return fmt.Errorf("store is nil")
	}
	if err := validateReference(ref); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	path := s.getImagePath(ref)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("failed to create image directory: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write image file: %w", err)
	}

	// Update metadata
	metadata := ImageMetadata{
		ImageReference: ref,
		StoredAt:       time.Now().Unix(),
	}
	if err := s.updateMetadata(metadata); err != nil {
		_ = os.Remove(path) // Clean up image file if metadata update fails
		return fmt.Errorf("failed to update metadata: %w", err)
	}

	// Update cache
	if s.cache != nil && s.ttl > 0 {
		s.cache[ref.String()] = &cachedImage{
			Data:      data,
			Timestamp: time.Now().Unix(),
		}
	}

	return nil
}

// List returns all stored images
func (s *LocalImageStore) List() ([]ImageReference, error) {
	if s == nil {
		return nil, fmt.Errorf("store is nil")
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	metadataList, err := s.readMetadata()
	if err != nil {
		return nil, fmt.Errorf("failed to read metadata: %w", err)
	}

	var refs []ImageReference
	for _, metadata := range metadataList {
		if s.ttl <= 0 || time.Now().Unix()-metadata.StoredAt < s.ttl {
			refs = append(refs, metadata.ImageReference)
		}
	}
	return refs, nil
}

func (s *LocalImageStore) readMetadata() ([]ImageMetadata, error) {
	metadataPath := filepath.Join(s.rootDir, "metadata.json")
	data, err := os.ReadFile(metadataPath)
	if err != nil {
		if os.IsNotExist(err) {
			return []ImageMetadata{}, nil
		}
		return nil, err
	}

	var metadataList []ImageMetadata
	if err := json.Unmarshal(data, &metadataList); err != nil {
		return nil, fmt.Errorf("invalid metadata format: %w", err)
	}
	return metadataList, nil
}

func (s *LocalImageStore) updateMetadata(newMetadata ImageMetadata) error {
	metadataList, err := s.readMetadata()
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	// Update or append metadata
	found := false
	for i, metadata := range metadataList {
		if metadata.ImageReference == newMetadata.ImageReference {
			metadataList[i] = newMetadata
			found = true
			break
		}
	}
	if !found {
		metadataList = append(metadataList, newMetadata)
	}

	// Write updated metadata
	data, err := json.Marshal(metadataList)
	if err != nil {
		return err
	}

	metadataPath := filepath.Join(s.rootDir, "metadata.json")
	return os.WriteFile(metadataPath, data, 0644)
}

func (s *LocalImageStore) removeFromMetadata(ref ImageReference) error {
	metadataList, err := s.readMetadata()
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	// Remove metadata entry
	newList := make([]ImageMetadata, 0, len(metadataList))
	for _, metadata := range metadataList {
		if metadata.ImageReference != ref {
			newList = append(newList, metadata)
		}
	}

	// Write updated metadata
	data, err := json.Marshal(newList)
	if err != nil {
		return err
	}

	metadataPath := filepath.Join(s.rootDir, "metadata.json")
	return os.WriteFile(metadataPath, data, 0644)
}

// Remove removes an image from local storage
func (s *LocalImageStore) Remove(ref ImageReference) error {
	if s == nil {
		return fmt.Errorf("store is nil")
	}
	if err := validateReference(ref); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	path := s.getImagePath(ref)
	if err := os.Remove(path); err != nil {
		if !os.IsNotExist(err) {
			return fmt.Errorf("failed to remove image file: %w", err)
		}
	}

	if err := s.removeFromMetadata(ref); err != nil {
		return fmt.Errorf("failed to update metadata: %w", err)
	}

	// Remove from cache
	if s.cache != nil {
		delete(s.cache, ref.String())
	}

	return nil
}

func (s *LocalImageStore) CleanExpiredImages() error {
	if s == nil {
		return fmt.Errorf("store is nil")
	}
	if s.ttl <= 0 {
		return nil // No cleanup needed if TTL is disabled
	}

	// First get the list of expired images under read lock
	s.mu.RLock()
	var toRemove []ImageReference
	metadataList, err := s.readMetadata()
	if err != nil {
		s.mu.RUnlock()
		return fmt.Errorf("failed to read metadata: %w", err)
	}

	now := time.Now().Unix()
	for _, metadata := range metadataList {
		if now-metadata.StoredAt >= s.ttl {
			toRemove = append(toRemove, metadata.ImageReference)
		}
	}
	s.mu.RUnlock()

	// Then remove them one by one
	var lastErr error
	for _, ref := range toRemove {
		if err := s.Remove(ref); err != nil {
			lastErr = fmt.Errorf("failed to remove expired image: %w", err)
		}
	}

	return lastErr
}

// getImagePath returns the full path for an image
func (s *LocalImageStore) getImagePath(ref ImageReference) string {
	filename := strings.ReplaceAll(ref.String(), "/", "_") + ".tar"
	return filepath.Join(s.rootDir, filename)
}

func validateReference(ref ImageReference) error {
	if ref.Registry == "" || ref.Repository == "" || ref.Tag == "" {
		return fmt.Errorf("invalid image reference: all fields must be non-empty")
	}
	return nil
}

// SetCacheTTL sets the cache TTL in seconds
func (s *LocalImageStore) SetCacheTTL(ttl int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ttl = int64(ttl)
}
