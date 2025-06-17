package oci

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// ConvertOCIToLXC converts an OCI image to an LXC template
func ConvertOCIToLXC(imageName, outputPath string) error {
	// Ensure docker is installed
	if _, err := exec.LookPath("docker"); err != nil {
		return fmt.Errorf("docker is not installed: %w", err)
	}

	// Create output directory if it doesn't exist
	outputDir := filepath.Dir(outputPath)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Pull the image
	pullCmd := exec.Command("docker", "pull", imageName)
	pullCmd.Stdout = os.Stdout
	pullCmd.Stderr = os.Stderr
	if err := pullCmd.Run(); err != nil {
		return fmt.Errorf("failed to pull image '%s': %w", imageName, err)
	}

	// Run container in background
	runCmd := exec.Command("docker", "run", "--rm", "--entrypoint", "sh", "-id", imageName)
	containerIDBytes, err := runCmd.Output()
	if err != nil {
		return fmt.Errorf("failed to start container: %w", err)
	}

	// Properly trim whitespace and get full container ID
	containerIDStr := string(containerIDBytes)
	if len(containerIDStr) > 0 && containerIDStr[len(containerIDStr)-1] == '\n' {
		containerIDStr = containerIDStr[:len(containerIDStr)-1]
	}

	// Export container filesystem and compress it using a simpler approach
	// Use shell to pipe docker export directly to gzip
	exportCmd := exec.Command("sh", "-c", fmt.Sprintf("docker export %s | gzip > %s", containerIDStr, outputPath))
	exportCmd.Stdout = os.Stdout
	exportCmd.Stderr = os.Stderr

	if err := exportCmd.Run(); err != nil {
		// Cleanup container on error
		exec.Command("docker", "kill", containerIDStr).Run()
		return fmt.Errorf("failed to export and compress container: %w", err)
	}

	// Cleanup container after successful export
	if err := exec.Command("docker", "kill", containerIDStr).Run(); err != nil {
		fmt.Printf("Warning: failed to cleanup container %s: %v\n", containerIDStr, err)
	}

	return nil
}
