package container_test

import (
	"testing"

	"github.com/larkinwc/proxmox-lxc-compose/pkg/common"
	"github.com/larkinwc/proxmox-lxc-compose/pkg/container"
	"github.com/larkinwc/proxmox-lxc-compose/pkg/internal/mock"
	testing_internal "github.com/larkinwc/proxmox-lxc-compose/pkg/internal/testing"
)

func TestTemplateOperations(t *testing.T) {
	// Create mock manager
	manager := container.NewMockLXCManager()

	mock, cleanup := mock.SetupMockCommand(&execCommand)
	defer cleanup()

	// Setup test container
	containerName := "test-container"
	templateName := "test-template"
	containerConfig := &common.Container{
		Image: "ubuntu:20.04",
	}

	mock.AddContainer(containerName, "STOPPED")
	err := manager.Create(containerName, containerConfig)
	testing_internal.AssertNoError(t, err)

	t.Run("create_template", func(t *testing.T) {
		err := manager.CreateTemplate(containerName, templateName, "Test template description")
		testing_internal.AssertNoError(t, err)

		// Verify template exists
		templates, err := manager.ListTemplates()
		testing_internal.AssertNoError(t, err)
		found := false
		for _, tmpl := range templates {
			if tmpl.Name == templateName {
				found = true
				testing_internal.AssertEqual(t, "Test template description", tmpl.Description)
				break
			}
		}
		testing_internal.AssertEqual(t, true, found)
	})

	t.Run("create_from_template", func(t *testing.T) {
		newContainerName := "from-template"
		err := manager.CreateFromTemplate(templateName, newContainerName, nil)
		testing_internal.AssertNoError(t, err)

		container, err := manager.Get(newContainerName)
		testing_internal.AssertNoError(t, err)
		testing_internal.AssertEqual(t, containerConfig.Image, container.Image)
	})

	t.Run("template_validation", func(t *testing.T) {
		// Test creating template from non-existent container
		err := manager.CreateTemplate("nonexistent", "invalid-template", "")
		testing_internal.AssertError(t, err)

		// Test creating container from non-existent template
		err = manager.CreateFromTemplate("nonexistent-template", "invalid-container", nil)
		testing_internal.AssertError(t, err)

		// Test creating template with existing name
		err = manager.CreateTemplate(containerName, templateName, "")
		testing_internal.AssertError(t, err)
	})
}
