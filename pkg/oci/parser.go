package oci

import (
	"fmt"
	"regexp"
	"strings"
)

var (
	// Based on OCI distribution spec
	namePattern = `^[a-z0-9]+(?:(?:[._]|__|[-]*)[a-z0-9]+)*$`
	tagPattern  = `^[\w][\w.-]{0,127}$`
	// sha256:<hex string>
	digestPattern = `^[A-Za-z][A-Za-z0-9]*(?:[-_+.][A-Za-z][A-Za-z0-9]*)*[:][[:xdigit:]]{32,}$`
)

var (
	nameRegex   = regexp.MustCompile(namePattern)
	tagRegex    = regexp.MustCompile(tagPattern)
	digestRegex = regexp.MustCompile(digestPattern)
)

// ParseImageReference parses an image reference string into an ImageReference
// Supports formats:
// - repository:tag
// - registry/repository:tag
// - registry/namespace/repository:tag
// - repository@digest
// - registry/repository@digest
func ParseImageReference(ref string) (ImageReference, error) {
	if ref == "" {
		return ImageReference{}, fmt.Errorf("empty image reference")
	}

	var registry, repository, tag, digest string

	// Split digest if present
	parts := strings.SplitN(ref, "@", 2)
	if len(parts) == 2 {
		digest = parts[1]
		if !digestRegex.MatchString(digest) {
			return ImageReference{}, fmt.Errorf("invalid digest format: %s", digest)
		}
	}

	// Handle the main part (everything before @digest if present)
	mainPart := parts[0]

	// Split tag if present
	parts = strings.SplitN(mainPart, ":", 2)
	if len(parts) == 2 {
		tag = parts[1]
		if !tagRegex.MatchString(tag) {
			return ImageReference{}, fmt.Errorf("invalid tag format: %s", tag)
		}
	} else {
		tag = "latest"
	}

	// Handle registry and repository
	parts = strings.Split(parts[0], "/")
	switch len(parts) {
	case 1:
		registry = "registry.hub.docker.com"
		repository = "library/" + parts[0]
	case 2:
		if strings.Contains(parts[0], ".") || strings.Contains(parts[0], ":") {
			registry = parts[0]
			repository = parts[1]
		} else {
			registry = "registry.hub.docker.com"
			repository = strings.Join(parts, "/")
		}
	default:
		registry = parts[0]
		repository = strings.Join(parts[1:], "/")
	}

	// Validate repository name
	for _, part := range strings.Split(repository, "/") {
		if !nameRegex.MatchString(part) {
			return ImageReference{}, fmt.Errorf("invalid repository name part: %s", part)
		}
	}

	return ImageReference{
		Registry:   registry,
		Repository: repository,
		Tag:        tag,
		Digest:     digest,
	}, nil
}