package validation

import (
	"strings"
	"testing"
)

func TestValidateStorageSize(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		wantBytes   int64
		wantErr     bool
		errContains string
	}{
		{
			name:      "bytes without unit",
			input:     "1024",
			wantBytes: 1024,
		},
		{
			name:      "bytes with B",
			input:     "1024B",
			wantBytes: 1024,
		},
		{
			name:      "kilobytes",
			input:     "1KB",
			wantBytes: 1024,
		},
		{
			name:      "megabytes",
			input:     "1MB",
			wantBytes: 1024 * 1024,
		},
		{
			name:      "gigabytes",
			input:     "1GB",
			wantBytes: 1024 * 1024 * 1024,
		},
		{
			name:      "terabytes",
			input:     "1TB",
			wantBytes: 1024 * 1024 * 1024 * 1024,
		},
		{
			name:      "decimal value",
			input:     "1.5GB",
			wantBytes: int64(1.5 * float64(1024*1024*1024)),
		},
		{
			name:      "lowercase units",
			input:     "1gb",
			wantBytes: 1024 * 1024 * 1024,
		},
		{
			name:      "space before unit",
			input:     "1 GB",
			wantBytes: 1024 * 1024 * 1024,
		},
		{
			name:        "negative value",
			input:       "-1GB",
			wantErr:     true,
			errContains: "invalid size format",
		},
		{
			name:        "invalid unit",
			input:       "1XB",
			wantErr:     true,
			errContains: "invalid size format",
		},
		{
			name:        "no numeric value",
			input:       "GB",
			wantErr:     true,
			errContains: "invalid size format",
		},
		{
			name:        "too large",
			input:       "2PB",
			wantErr:     true,
			errContains: "size too large",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ValidateStorageSize(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				} else if tt.errContains != "" && !contains(err.Error(), tt.errContains) {
					t.Errorf("error %q does not contain %q", err.Error(), tt.errContains)
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			if got != tt.wantBytes {
				t.Errorf("ValidateStorageSize(%q) = %d, want %d", tt.input, got, tt.wantBytes)
			}
		})
	}
}

func TestFormatBytes(t *testing.T) {
	tests := []struct {
		name     string
		bytes    int64
		expected string
	}{
		{
			name:     "bytes",
			bytes:    900,
			expected: "900B",
		},
		{
			name:     "kilobytes",
			bytes:    1024 * 2,
			expected: "2K",
		},
		{
			name:     "megabytes",
			bytes:    1024 * 1024 * 3,
			expected: "3M",
		},
		{
			name:     "gigabytes",
			bytes:    1024 * 1024 * 1024 * 4,
			expected: "4G",
		},
		{
			name:     "decimal gigabytes",
			bytes:    int64(1.5 * float64(1024*1024*1024)),
			expected: "1.50G",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatBytes(tt.bytes)
			if got != tt.expected {
				t.Errorf("FormatBytes(%d) = %q, want %q", tt.bytes, got, tt.expected)
			}
		})
	}
}

func contains(s, substr string) bool {
	return s != "" && substr != "" && strings.Contains(s, substr)
}
