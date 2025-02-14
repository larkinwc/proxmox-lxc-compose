package oci

import "testing"

func TestParseImageReference(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    ImageReference
		wantErr bool
	}{
		{
			name:  "simple image name",
			input: "ubuntu",
			want: ImageReference{
				Registry:   "registry.hub.docker.com",
				Repository: "library/ubuntu",
				Tag:        "latest",
			},
		},
		{
			name:  "image with tag",
			input: "ubuntu:20.04",
			want: ImageReference{
				Registry:   "registry.hub.docker.com",
				Repository: "library/ubuntu",
				Tag:        "20.04",
			},
		},
		{
			name:  "full reference with registry",
			input: "registry.example.com/org/app:1.0",
			want: ImageReference{
				Registry:   "registry.example.com",
				Repository: "org/app",
				Tag:        "1.0",
			},
		},
		{
			name:  "reference with digest",
			input: "ubuntu@sha256:1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef",
			want: ImageReference{
				Registry:   "registry.hub.docker.com",
				Repository: "library/ubuntu",
				Tag:        "latest",
				Digest:     "sha256:1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef",
			},
		},
		{
			name:    "invalid tag",
			input:   "ubuntu:invalid@tag",
			wantErr: true,
		},
		{
			name:    "invalid digest",
			input:   "ubuntu@invalid-digest",
			wantErr: true,
		},
		{
			name:    "invalid repository name",
			input:   "invalid..name",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseImageReference(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseImageReference() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err == nil {
				if got.Registry != tt.want.Registry {
					t.Errorf("Registry = %v, want %v", got.Registry, tt.want.Registry)
				}
				if got.Repository != tt.want.Repository {
					t.Errorf("Repository = %v, want %v", got.Repository, tt.want.Repository)
				}
				if got.Tag != tt.want.Tag {
					t.Errorf("Tag = %v, want %v", got.Tag, tt.want.Tag)
				}
				if got.Digest != tt.want.Digest {
					t.Errorf("Digest = %v, want %v", got.Digest, tt.want.Digest)
				}
			}
		})
	}
}