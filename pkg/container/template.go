package container

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/larkinwc/proxmox-lxc-compose/pkg/common"
)

// validateTemplateName rejects names that could escape the templates directory
// when joined into filesystem paths (separators, "..", or empty).
func validateTemplateName(name string) error {
	if name == "" || name != filepath.Base(name) ||
		strings.ContainsAny(name, `/\`) || strings.Contains(name, "..") {
		return fmt.Errorf("invalid template name: %q", name)
	}
	return nil
}

// Template represents a container template
type Template struct {
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Config      *common.Container `json:"config"`
	CreatedAt   time.Time         `json:"created_at"`
}

// CreateTemplate creates a new template from an existing container
func (m *LXCManager) CreateTemplate(containerName string, templateName string, description string) error {
	if err := validateTemplateName(templateName); err != nil {
		return err
	}
	// Get the container configuration
	container, err := m.Get(containerName)
	if err != nil {
		return fmt.Errorf("failed to get container: %w", err)
	}

	// Check if template already exists
	if _, err := m.GetTemplate(templateName); err == nil {
		return fmt.Errorf("template %s already exists", templateName)
	}

	// Create template directory
	templatesDir := filepath.Join(m.configPath, "templates")
	if err := os.MkdirAll(templatesDir, 0755); err != nil {
		return fmt.Errorf("failed to create templates directory: %w", err)
	}

	// Copy container files to template directory
	containerPath := filepath.Join(m.configPath, containerName)
	templatePath := filepath.Join(templatesDir, templateName)
	if err := copyDir(containerPath, templatePath); err != nil {
		os.RemoveAll(templatePath)
		return fmt.Errorf("failed to copy container files: %w", err)
	}

	// Convert config.Container to common.Container
	commonConfig := &common.Container{
		Image: container.Config.Image,
	}

	// Create template
	template := &Template{
		Name:        templateName,
		Description: description,
		Config:      commonConfig,
		CreatedAt:   time.Now(),
	}

	// Save template metadata
	return m.saveTemplate(template)
}

// GetTemplate retrieves a template by name
func (m *LXCManager) GetTemplate(name string) (*Template, error) {
	if err := validateTemplateName(name); err != nil {
		return nil, err
	}
	templatePath := filepath.Join(m.configPath, "templates", name+".json")
	data, err := os.ReadFile(templatePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read template: %w", err)
	}

	var template Template
	if err := json.Unmarshal(data, &template); err != nil {
		return nil, fmt.Errorf("failed to unmarshal template: %w", err)
	}

	return &template, nil
}

// ListTemplates returns a list of all available templates
func (m *LXCManager) ListTemplates() ([]*Template, error) {
	templatesDir := filepath.Join(m.configPath, "templates")
	if err := os.MkdirAll(templatesDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create templates directory: %w", err)
	}

	entries, err := os.ReadDir(templatesDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read templates directory: %w", err)
	}

	var templates []*Template
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}

		templateName := strings.TrimSuffix(entry.Name(), ".json")
		template, err := m.GetTemplate(templateName)
		if err != nil {
			continue
		}

		templates = append(templates, template)
	}

	return templates, nil
}

// DeleteTemplate removes a template and its associated files
func (m *LXCManager) DeleteTemplate(name string) error {
	if err := validateTemplateName(name); err != nil {
		return err
	}
	templatesDir := filepath.Join(m.configPath, "templates")
	metadataPath := filepath.Join(templatesDir, name+".json")

	if _, err := os.Stat(metadataPath); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("template %s does not exist", name)
		}
		return fmt.Errorf("failed to stat template: %w", err)
	}

	// Remove the copied container files (best effort if present)
	templatePath := filepath.Join(templatesDir, name)
	if err := os.RemoveAll(templatePath); err != nil {
		return fmt.Errorf("failed to remove template files: %w", err)
	}

	// Remove the metadata file
	if err := os.Remove(metadataPath); err != nil {
		return fmt.Errorf("failed to remove template metadata: %w", err)
	}

	return nil
}

// saveTemplate saves a template to disk
func (m *LXCManager) saveTemplate(template *Template) error {
	templatesDir := filepath.Join(m.configPath, "templates")
	if err := os.MkdirAll(templatesDir, 0755); err != nil {
		return fmt.Errorf("failed to create templates directory: %w", err)
	}

	data, err := json.MarshalIndent(template, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal template: %w", err)
	}

	templatePath := filepath.Join(templatesDir, template.Name+".json")
	if err := os.WriteFile(templatePath, data, 0600); err != nil {
		return fmt.Errorf("failed to write template file: %w", err)
	}

	return nil
}

func (m *LXCManager) CreateFromTemplate(templateName string, containerName string, overrides *common.Container) error {
	// Get the template configuration
	template, err := m.GetTemplate(templateName)
	if err != nil {
		return fmt.Errorf("failed to get template: %w", err)
	}

	// Create a new container config from the template
	config := template.Config

	// Apply any overrides
	if overrides != nil {
		if overrides.Image != "" {
			config.Image = overrides.Image
		}
		if overrides.CPU != nil {
			config.CPU = overrides.CPU
		}
		if overrides.Memory != nil {
			config.Memory = overrides.Memory
		}
		if overrides.Storage != nil {
			config.Storage = overrides.Storage
		}
		if overrides.Network != nil {
			config.Network = overrides.Network
		}
		if overrides.Environment != nil {
			config.Environment = overrides.Environment
		}
		if overrides.Command != nil {
			config.Command = overrides.Command
		}
		if overrides.Entrypoint != nil {
			config.Entrypoint = overrides.Entrypoint
		}
		if overrides.Devices != nil {
			config.Devices = overrides.Devices
		}
		if overrides.Security != nil {
			config.Security = overrides.Security
		}
	}

	// Create the container with the template config
	return m.Create(containerName, config)
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

// ... existing code ...
