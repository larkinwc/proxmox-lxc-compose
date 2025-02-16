package validation

import (
	"fmt"
	"path/filepath"
	"proxmox-lxc-compose/pkg/common"
	"regexp"
	"strings"
)

// Supported device types
var supportedDeviceTypes = map[string]bool{
	"unix-char":  true, // Character device
	"unix-block": true, // Block device
	"nic":        true, // Network interface
	"disk":       true, // Disk device
	"gpu":        true, // GPU device
	"usb":        true, // USB device
	"pci":        true, // PCI device
}

// ValidateDeviceType validates the device type
func ValidateDeviceType(deviceType string) error {
	if deviceType == "" {
		return fmt.Errorf("device type is required")
	}

	if !supportedDeviceTypes[strings.ToLower(deviceType)] {
		return fmt.Errorf("unsupported device type: %s (supported types: unix-char, unix-block, nic, disk, gpu, usb, pci)", deviceType)
	}

	return nil
}

// ValidateDeviceName validates the device name
func ValidateDeviceName(name string) error {
	if name == "" {
		return fmt.Errorf("device name is required")
	}

	// Device names should be alphanumeric with hyphens and underscores
	validName := regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_-]*$`)
	if !validName.MatchString(name) {
		return fmt.Errorf("invalid device name (must start with letter/number and contain only letters, numbers, hyphens, and underscores): %s", name)
	}

	if len(name) > 64 {
		return fmt.Errorf("device name too long (max 64 characters): %s", name)
	}

	return nil
}

// ValidateDevicePath validates a device path (source or destination)
func ValidateDevicePath(path string, isSource bool) error {
	if path == "" {
		if isSource {
			return fmt.Errorf("device source path is required")
		}
		return nil // destination is optional for some device types
	}

	// Path should be absolute
	if !filepath.IsAbs(path) {
		return fmt.Errorf("device path must be absolute: %s", path)
	}

	// Check for parent directory references
	if strings.Contains(path, "..") || strings.Contains(filepath.Clean(path), "..") {
		return fmt.Errorf("device path must not contain '..': %s", path)
	}

	// Ensure path is clean/normalized
	cleanPath := filepath.Clean(path)
	if cleanPath != path {
		return fmt.Errorf("device path must be normalized: %s", path)
	}

	return nil
}

// ValidateDeviceOptions validates device options based on device type
func ValidateDeviceOptions(deviceType string, options []string) error {
	if len(options) == 0 {
		return nil
	}

	validOptions := map[string]bool{
		"ro":         true, // Read-only
		"rw":         true, // Read-write
		"required":   true, // Device is required
		"optional":   true, // Device is optional
		"recursive":  true, // Recursive bind mount
		"bind":       true, // Bind mount
		"create":     true, // Create destination path
		"persistent": true, // Persistent device
		"dynamic":    true, // Dynamic device
	}

	deviceType = strings.ToLower(deviceType)
	for _, opt := range options {
		opt = strings.ToLower(opt)
		if !validOptions[opt] {
			return fmt.Errorf("invalid device option for type %s: %s", deviceType, opt)
		}

		// Check for mutually exclusive options
		if opt == "ro" && containsOption(options, "rw") {
			return fmt.Errorf("conflicting device options: ro and rw")
		}
		if opt == "required" && containsOption(options, "optional") {
			return fmt.Errorf("conflicting device options: required and optional")
		}
	}

	return nil
}

// ValidateDevice validates a complete device configuration
func ValidateDevice(name, deviceType, source, destination string, options []string) error {
	// Validate device name
	if err := ValidateDeviceName(name); err != nil {
		return err
	}

	// Validate device type
	if err := ValidateDeviceType(deviceType); err != nil {
		return err
	}

	// Validate source path
	if err := ValidateDevicePath(source, true); err != nil {
		return err
	}

	// Validate destination path
	if err := ValidateDevicePath(destination, false); err != nil {
		return err
	}

	// Validate device options
	if err := ValidateDeviceOptions(deviceType, options); err != nil {
		return err
	}

	return nil
}

// Helper function to check if an option exists in a list (case-insensitive)
func containsOption(options []string, target string) bool {
	target = strings.ToLower(target)
	for _, opt := range options {
		if strings.ToLower(opt) == target {
			return true
		}
	}
	return false
}

func validateDeviceConfig(device *common.DeviceConfig) error {
	// ...existing code...

	return nil
}
