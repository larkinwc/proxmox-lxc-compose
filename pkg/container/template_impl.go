package container

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// Template represents an LXC container template
type Template struct {
	Name        string
	Description string
	Path        string
}

// CreateTemplate creates a template from an existing container
func (m *LXCManager) CreateTemplate(containerName, templateName, description string) error {
	if !m.ContainerExists(containerName) {
		return fmt.Errorf("container %s does not exist", containerName)
	}

	templatesPath := filepath.Join(m.configPath, "templates")
	if err := os.MkdirAll(templatesPath, 0755); err != nil {
		return fmt.Errorf("failed to create templates directory: %w", err)
	}

	templatePath := filepath.Join(templatesPath, templateName)
	if _, err := os.Stat(templatePath); err == nil {
		return fmt.Errorf("template %s already exists", templateName)
	}

	// Create template directory
	if err := os.MkdirAll(templatePath, 0755); err != nil {
		return fmt.Errorf("failed to create template directory: %w", err)
	}

	// Copy container files to template
	containerPath := filepath.Join(m.configPath, containerName)
	if err := copyDir(containerPath, templatePath); err != nil {
		os.RemoveAll(templatePath)
		return fmt.Errorf("failed to copy container files: %w", err)
	}

	// Write metadata file
	metadata := fmt.Sprintf("description=%s\n", description)
	if err := os.WriteFile(filepath.Join(templatePath, "metadata"), []byte(metadata), 0644); err != nil {
		os.RemoveAll(templatePath)
		return fmt.Errorf("failed to write metadata: %w", err)
	}

	return nil
}

// ListTemplates returns a list of available templates
func (m *LXCManager) ListTemplates() ([]Template, error) {
	templatesPath := filepath.Join(m.configPath, "templates")
	entries, err := os.ReadDir(templatesPath)
	if err != nil {
		if os.IsNotExist(err) {
			return []Template{}, nil
		}
		return nil, fmt.Errorf("failed to read templates directory: %w", err)
	}

	var templates []Template
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		templatePath := filepath.Join(templatesPath, entry.Name())
		metadata, err := os.ReadFile(filepath.Join(templatePath, "metadata"))
		if err != nil {
			continue
		}

		description := ""
		for _, line := range strings.Split(string(metadata), "\n") {
			if strings.HasPrefix(line, "description=") {
				description = strings.TrimPrefix(line, "description=")
				break
			}
		}

		templates = append(templates, Template{
			Name:        entry.Name(),
			Description: description,
			Path:        templatePath,
		})
	}

	return templates, nil
}

// CreateContainerFromTemplate creates a new container from a template
func (m *LXCManager) CreateContainerFromTemplate(templateName, containerName string) error {
	templatesPath := filepath.Join(m.configPath, "templates")
	templatePath := filepath.Join(templatesPath, templateName)
	if _, err := os.Stat(templatePath); err != nil {
		return fmt.Errorf("template %s does not exist", templateName)
	}

	containerPath := filepath.Join(m.configPath, containerName)
	if _, err := os.Stat(containerPath); err == nil {
		return fmt.Errorf("container %s already exists", containerName)
	}

	// Create container directory
	if err := os.MkdirAll(containerPath, 0755); err != nil {
		return fmt.Errorf("failed to create container directory: %w", err)
	}

	// Copy template files to container directory, excluding metadata
	err := filepath.Walk(templatePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip metadata file
		if info.Name() == "metadata" {
			return nil
		}

		relPath, err := filepath.Rel(templatePath, path)
		if err != nil {
			return err
		}

		destPath := filepath.Join(containerPath, relPath)

		if info.IsDir() {
			return os.MkdirAll(destPath, info.Mode())
		}

		return copyFile(path, destPath)
	})

	if err != nil {
		os.RemoveAll(containerPath)
		return fmt.Errorf("failed to copy template files: %w", err)
	}

	return nil
}

// DeleteTemplate deletes a template
func (m *LXCManager) DeleteTemplate(templateName string) error {
	templatesPath := filepath.Join(m.configPath, "templates")
	templatePath := filepath.Join(templatesPath, templateName)
	if _, err := os.Stat(templatePath); err != nil {
		return fmt.Errorf("template %s does not exist", templateName)
	}

	if err := os.RemoveAll(templatePath); err != nil {
		return fmt.Errorf("failed to delete template: %w", err)
	}

	return nil
}

// Helper functions for copying files and directories
func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	if _, err := io.Copy(destFile, sourceFile); err != nil {
		return err
	}

	sourceInfo, err := os.Stat(src)
	if err != nil {
		return err
	}

	return os.Chmod(dst, sourceInfo.Mode())
}

func copyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}

		destPath := filepath.Join(dst, relPath)

		if info.IsDir() {
			return os.MkdirAll(destPath, info.Mode())
		}

		return copyFile(path, destPath)
	})
}
