package container

import (
	"os"
	"path/filepath"
	"proxmox-lxc-compose/pkg/testutil"
	"testing"
)

// Save the original exec.Command
var origExecCommand = execCommand

func TestTemplateManagement(t *testing.T) {
	// Create temporary directory for tests
	tempDir, cleanup := testutil.TempDir(t)
	defer cleanup()

	// Reset execCommand after the test
	defer func() { execCommand = origExecCommand }()

	// Create state manager
	stateManager, err := NewStateManager(filepath.Join(tempDir, "state"))
	testutil.AssertNoError(t, err)

	// Create manager
	manager := &LXCManager{
		configPath: tempDir,
		state:      stateManager,
	}

	// Create a test container
	containerName := "test-container"
	containerPath := filepath.Join(tempDir, containerName)
	if err := os.MkdirAll(containerPath, 0755); err != nil {
		t.Fatalf("failed to create test container: %v", err)
	}

	t.Run("CreateTemplate", func(t *testing.T) {
		// Test creating a template from non-existent container
		if err := manager.CreateTemplate("nonexistent", "test-template", "test description"); err == nil {
			t.Error("expected error when creating template from non-existent container")
		}

		// Test creating a template from existing container
		if err := manager.CreateTemplate(containerName, "test-template", "test description"); err != nil {
			t.Errorf("failed to create template: %v", err)
		}

		// Verify template was created
		templatePath := filepath.Join(tempDir, "templates", "test-template")
		if _, err := os.Stat(templatePath); err != nil {
			t.Errorf("template directory not created: %v", err)
		}

		// Verify metadata file was created
		metadata, err := os.ReadFile(filepath.Join(templatePath, "metadata"))
		if err != nil {
			t.Errorf("failed to read metadata: %v", err)
		}
		expectedMetadata := "description=test description\n"
		if string(metadata) != expectedMetadata {
			t.Errorf("unexpected metadata content: got %q, want %q", string(metadata), expectedMetadata)
		}

		// Test creating a template with same name
		if err := manager.CreateTemplate(containerName, "test-template", "another description"); err == nil {
			t.Error("expected error when creating template with existing name")
		}
	})

	t.Run("ListTemplates", func(t *testing.T) {
		// Test listing templates
		templates, err := manager.ListTemplates()
		if err != nil {
			t.Errorf("failed to list templates: %v", err)
		}
		if len(templates) != 1 {
			t.Errorf("unexpected number of templates: got %d, want 1", len(templates))
		}
		if templates[0].Name != "test-template" {
			t.Errorf("unexpected template name: got %q, want %q", templates[0].Name, "test-template")
		}
		if templates[0].Description != "test description" {
			t.Errorf("unexpected template description: got %q, want %q", templates[0].Description, "test description")
		}
	})

	t.Run("CreateContainerFromTemplate", func(t *testing.T) {
		// Test creating container from non-existent template
		if err := manager.CreateContainerFromTemplate("nonexistent", "new-container"); err == nil {
			t.Error("expected error when creating container from non-existent template")
		}

		// Test creating container from template
		if err := manager.CreateContainerFromTemplate("test-template", "new-container"); err != nil {
			t.Errorf("failed to create container from template: %v", err)
		}

		// Verify container was created
		containerPath := filepath.Join(tempDir, "new-container")
		if _, err := os.Stat(containerPath); err != nil {
			t.Errorf("container directory not created: %v", err)
		}

		// Verify metadata file was not copied
		if _, err := os.Stat(filepath.Join(containerPath, "metadata")); err == nil {
			t.Error("metadata file should not be copied to container")
		}

		// Test creating container with existing name
		if err := manager.CreateContainerFromTemplate("test-template", "new-container"); err == nil {
			t.Error("expected error when creating container with existing name")
		}
	})

	t.Run("DeleteTemplate", func(t *testing.T) {
		// Test deleting non-existent template
		if err := manager.DeleteTemplate("nonexistent"); err == nil {
			t.Error("expected error when deleting non-existent template")
		}

		// Test deleting template
		if err := manager.DeleteTemplate("test-template"); err != nil {
			t.Errorf("failed to delete template: %v", err)
		}

		// Verify template was deleted
		templatePath := filepath.Join(tempDir, "templates", "test-template")
		if _, err := os.Stat(templatePath); err == nil {
			t.Error("template directory should be deleted")
		}

		// Verify templates list is empty
		templates, err := manager.ListTemplates()
		if err != nil {
			t.Errorf("failed to list templates: %v", err)
		}
		if len(templates) != 0 {
			t.Errorf("unexpected number of templates after deletion: got %d, want 0", len(templates))
		}
	})
}
