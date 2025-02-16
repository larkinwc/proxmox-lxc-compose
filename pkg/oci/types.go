package oci

import "context"

// ImageReference represents a reference to an OCI image
type ImageReference struct {
	Registry   string
	Repository string
	Tag        string
	Digest     string
}

// String returns a string representation of the image reference
func (r ImageReference) String() string {
	ref := r.Registry + "/" + r.Repository
	if r.Tag != "" {
		ref += ":" + r.Tag
	}
	return ref
}

// ImageManager handles OCI image operations
type ImageManager interface {
	// Pull pulls an image from a registry
	Pull(ctx context.Context, ref ImageReference) error

	// Push pushes an image to a registry
	Push(ctx context.Context, ref ImageReference) error

	// Save stores an image in the local cache
	Save(ctx context.Context, ref ImageReference) error

	// Load loads an image from the local cache
	Load(ctx context.Context, ref ImageReference) error

	// List returns a list of locally stored images
	List(ctx context.Context) ([]ImageReference, error)

	// Delete removes an image from local storage
	Delete(ctx context.Context, ref ImageReference) error
}

// ImageStore represents the local image storage
type ImageStore interface {
	// Store stores image data
	Store(ref ImageReference, data []byte) error

	// Retrieve retrieves image data
	Retrieve(ref ImageReference) ([]byte, error)

	// Remove removes an image from storage
	Remove(ref ImageReference) error

	// List returns all stored images
	List() ([]ImageReference, error)
}
