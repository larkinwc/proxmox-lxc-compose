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
	containerID, err := runCmd.Output()
	if err != nil {
		return fmt.Errorf("failed to start container: %w", err)
	}
	containerIDStr := string(containerID)[:12]

	// Cleanup container on exit
	defer func() {
		if err := exec.Command("docker", "kill", containerIDStr).Run(); err != nil {
			// We're in a defer, so just log the error
			fmt.Printf("Warning: failed to kill container %s: %v\n", containerIDStr, err)
		}
	}()

	// Export container filesystem
	exportCmd := exec.Command("docker", "export", containerIDStr)
	exportFile, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer exportFile.Close()

	exportCmd.Stdout = exportFile
	exportCmd.Stderr = os.Stderr
	if err := exportCmd.Run(); err != nil {
		return fmt.Errorf("failed to export container: %w", err)
	}

	return nil
}
