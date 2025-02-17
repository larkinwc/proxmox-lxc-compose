package validation

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/larkinwc/proxmox-lxc-compose/pkg/common"
)

var (
	// sizeRegex matches patterns like: 10B, 10KB, 10MB, 10GB, 10TB, 10PB (case insensitive)
	sizeRegex = regexp.MustCompile(`^(\d+(?:\.\d+)?)\s*([KMGTP])?B?$`)

	// multipliers for different size units
	sizeMultipliers = map[string]int64{
		"":  1,                                // Bytes
		"K": 1024,                             // Kilobytes
		"M": 1024 * 1024,                      // Megabytes
		"G": 1024 * 1024 * 1024,               // Gigabytes
		"T": 1024 * 1024 * 1024 * 1024,        // Terabytes
		"P": 1024 * 1024 * 1024 * 1024 * 1024, // Petabytes
	}
)

// ValidateStorageSize validates a storage size string and returns the size in bytes
func ValidateStorageSize(size string) (int64, error) {
	// Normalize input
	size = strings.TrimSpace(strings.ToUpper(size))

	// Match against regex
	matches := sizeRegex.FindStringSubmatch(size)
	if matches == nil {
		return 0, fmt.Errorf("invalid size format: %s (should be a number followed by optional unit B/KB/MB/GB/TB/PB)", size)
	}

	// Parse numeric value
	value, err := strconv.ParseFloat(matches[1], 64)
	if err != nil {
		return 0, fmt.Errorf("invalid numeric value: %s", matches[1])
	}

	// Get unit multiplier (default to bytes if no unit specified)
	unit := matches[2]
	multiplier, ok := sizeMultipliers[unit]
	if !ok {
		return 0, fmt.Errorf("invalid size unit: %s", unit)
	}

	// Calculate total bytes
	bytes := int64(value * float64(multiplier))

	// Validate reasonable limits
	if bytes <= 0 {
		return 0, fmt.Errorf("size must be positive")
	}
	if bytes > sizeMultipliers["P"] {
		return 0, fmt.Errorf("size too large (maximum is 1PB)")
	}

	return bytes, nil
}

func FormatBytes(bytes int64) string {
	units := []string{"", "K", "M", "G", "T", "P", "E"}
	value := float64(bytes)
	unit := 0

	for value >= 1024 && unit < len(units)-1 {
		value /= 1024
		unit++
	}

	// If the value is a whole number, format without decimals
	if value == float64(int64(value)) {
		return fmt.Sprintf("%d%s", int64(value), units[unit])
	}

	return fmt.Sprintf("%.1f%s", value, units[unit])
}

func ValidateStorageConfig(config *common.StorageConfig) error {
	if config == nil {
		return fmt.Errorf("storage configuration is required")
	}

	if config.Root == "" {
		return fmt.Errorf("root storage size is required")
	}

	if _, err := ValidateStorageSize(config.Root); err != nil {
		return fmt.Errorf("invalid root storage size: %w", err)
	}

	switch config.Backend {
	case "dir", "zfs", "btrfs", "lvm":
		// Valid backends
	default:
		return fmt.Errorf("invalid storage backend: %s", config.Backend)
	}

	if config.Backend == "zfs" && config.Pool == "" {
		return fmt.Errorf("storage pool is required for zfs backend")
	}

	for _, mount := range config.Mounts {
		if mount.Source == "" {
			return fmt.Errorf("mount source is required")
		}
		if mount.Target == "" {
			return fmt.Errorf("mount target is required")
		}
		if mount.Type == "" {
			return fmt.Errorf("mount type is required")
		}
		switch mount.Type {
		case "bind", "volume":
			// Valid mount types
		default:
			return fmt.Errorf("invalid mount type: %s", mount.Type)
		}
	}

	return nil
}
