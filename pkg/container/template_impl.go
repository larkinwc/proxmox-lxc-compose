package container

import (
	"fmt"
	"os"
	"path/filepath"
)

// Template represents a container template
type Template struct {
	Name        string
	Description string
	Path        string
}

// CreateTemplate creates a new template from an existing container
func (m *LXCManager) CreateTemplate(containerName, templateName, description string) error {
	// Check if container exists
	if !m.ContainerExists(containerName) {
		return fmt.Errorf("container %s does not exist", containerName)
	}

	// Check if template already exists
	templatePath := filepath.Join(m.configPath, "templates", templateName)
	if _, err := os.Stat(templatePath); err == nil {
		return fmt.Errorf("template %s already exists", templateName)
	}

	// Create templates directory if it doesn't exist
	templatesDir := filepath.Join(m.configPath, "templates")
	if err := os.MkdirAll(templatesDir, 0755); err != nil {
		return fmt.Errorf("failed to create templates directory: %w", err)
	}

	// Copy container config to template directory
	containerPath := filepath.Join(m.configPath, containerName)
	if err := copyDir(containerPath, templatePath); err != nil {
		return fmt.Errorf("failed to copy container to template: %w", err)
	}

	// Create template metadata file
	metadata := fmt.Sprintf("description=%s\n", description)
	if err := os.WriteFile(filepath.Join(templatePath, "metadata"), []byte(metadata), 0644); err != nil {
		return fmt.Errorf("failed to write template metadata: %w", err)
	}

	return nil
}

// ListTemplates returns a list of available templates
func (m *LXCManager) ListTemplates() ([]Template, error) {
	templatesDir := filepath.Join(m.configPath, "templates")
	if _, err := os.Stat(templatesDir); os.IsNotExist(err) {
		return nil, nil
	}

	entries, err := os.ReadDir(templatesDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read templates directory: %w", err)
	}

	var templates []Template
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		templatePath := filepath.Join(templatesDir, entry.Name())
		metadata, err := os.ReadFile(filepath.Join(templatePath, "metadata"))
		if err != nil {
			continue
		}

		description := ""
		if len(metadata) > 0 {
			// Parse description=value format
			content := string(metadata)
			if prefix := "description="; len(content) > len(prefix) && content[:len(prefix)] == prefix {
				description = content[len(prefix):]
				if description[len(description)-1] == '\n' {
					description = description[:len(description)-1]
				}
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

// DeleteTemplate removes a template
func (m *LXCManager) DeleteTemplate(templateName string) error {
	templatePath := filepath.Join(m.configPath, "templates", templateName)
	if _, err := os.Stat(templatePath); os.IsNotExist(err) {
		return fmt.Errorf("template %s does not exist", templateName)
	}

	if err := os.RemoveAll(templatePath); err != nil {
		return fmt.Errorf("failed to delete template: %w", err)
	}

	return nil
}

// CreateContainerFromTemplate creates a new container from a template
func (m *LXCManager) CreateContainerFromTemplate(templateName, containerName string) error {
	templatePath := filepath.Join(m.configPath, "templates", templateName)
	if _, err := os.Stat(templatePath); os.IsNotExist(err) {
		return fmt.Errorf("template %s does not exist", templateName)
	}

	containerPath := filepath.Join(m.configPath, containerName)
	if _, err := os.Stat(containerPath); err == nil {
		return fmt.Errorf("container %s already exists", containerName)
	}

	// Copy template to new container directory
	if err := copyDir(templatePath, containerPath); err != nil {
		return fmt.Errorf("failed to create container from template: %w", err)
	}

	// Remove metadata file from container directory
	if err := os.Remove(filepath.Join(containerPath, "metadata")); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove template metadata: %w", err)
	}

	return nil
}

// copyDir recursively copies a directory tree
func copyDir(src, dst string) error {
	if err := os.MkdirAll(dst, 0755); err != nil {
		return err
	}

	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		if entry.IsDir() {
			if err := copyDir(srcPath, dstPath); err != nil {
				return err
			}
		} else {
			data, err := os.ReadFile(srcPath)
			if err != nil {
				return err
			}
			if err := os.WriteFile(dstPath, data, 0644); err != nil {
				return err
			}
		}
	}

	return nil
}
