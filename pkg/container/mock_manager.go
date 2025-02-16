package container

import (
	"fmt"
	"proxmox-lxc-compose/pkg/common"
	"proxmox-lxc-compose/pkg/logging"
)

type MockLXCManager struct {
	Containers       map[string]*common.Container
	Templates        map[string]*Template
	networkBandwidth map[string]map[string]*common.BandwidthLimit
}

func NewMockLXCManager() *MockLXCManager {
	return &MockLXCManager{
		Containers:       make(map[string]*common.Container),
		Templates:        make(map[string]*Template),
		networkBandwidth: make(map[string]map[string]*common.BandwidthLimit),
	}
}

func (m *MockLXCManager) Create(name string, cfg *common.Container) error {
	if _, exists := m.Containers[name]; exists {
		return fmt.Errorf("container %s already exists", name)
	}
	m.Containers[name] = cfg
	return nil
}

func (m *MockLXCManager) CreateTemplate(containerName, templateName, description string) error {
	container, exists := m.Containers[containerName]
	if !exists {
		return fmt.Errorf("container %s does not exist", containerName)
	}
	if _, exists := m.Templates[templateName]; exists {
		return fmt.Errorf("template %s already exists", templateName)
	}

	m.Templates[templateName] = &Template{
		Name:        templateName,
		Description: description,
		Config:      container,
	}
	return nil
}

func (m *MockLXCManager) ListTemplates() ([]Template, error) {
	var templates []Template
	for _, template := range m.Templates {
		templates = append(templates, *template)
	}
	return templates, nil
}

func (m *MockLXCManager) CreateContainerFromTemplate(templateName, containerName string) error {
	if _, exists := m.Templates[templateName]; !exists {
		return fmt.Errorf("template %s does not exist", templateName)
	}
	if _, exists := m.Containers[containerName]; exists {
		return fmt.Errorf("container %s already exists", containerName)
	}
	m.Containers[containerName] = &common.Container{}
	return nil
}

func (m *MockLXCManager) CreateFromTemplate(templateName, containerName string, overrides interface{}) error {
	template, exists := m.Templates[templateName]
	if !exists {
		return fmt.Errorf("template %s does not exist", templateName)
	}
	if _, exists := m.Containers[containerName]; exists {
		return fmt.Errorf("container %s already exists", containerName)
	}

	// Copy the template config
	newConfig := &common.Container{
		Image: template.Config.Image,
	}
	m.Containers[containerName] = newConfig
	return nil
}

func (m *MockLXCManager) Get(containerName string) (*common.Container, error) {
	if container, exists := m.Containers[containerName]; exists {
		return container, nil
	}
	return nil, fmt.Errorf("container %s does not exist", containerName)
}

func (m *MockLXCManager) DeleteTemplate(templateName string) error {
	if _, exists := m.Templates[templateName]; !exists {
		return fmt.Errorf("template %s does not exist", templateName)
	}
	delete(m.Templates, templateName)
	return nil
}

func (m *MockLXCManager) SetNetworkBandwidthLimit(containerName string, limit NetworkBandwidthLimit) error {
	if _, exists := m.Containers[containerName]; !exists {
		return fmt.Errorf("container %s does not exist", containerName)
	}
	return nil
}

func (m *MockLXCManager) VerifyNetworkConfig(containerName string) error {
	if _, exists := m.Containers[containerName]; !exists {
		return fmt.Errorf("container %s does not exist", containerName)
	}
	return nil
}

func (m *MockLXCManager) TestConnectivity(containerName string) error {
	if _, exists := m.Containers[containerName]; !exists {
		return fmt.Errorf("container %s does not exist", containerName)
	}
	return nil
}

func (m *MockLXCManager) GetNetworkBandwidthLimits(containerName, iface string) (*common.BandwidthLimit, error) {
	if _, exists := m.Containers[containerName]; !exists {
		return nil, fmt.Errorf("container %s does not exist", containerName)
	}

	if m.networkBandwidth[containerName] == nil {
		return nil, fmt.Errorf("no bandwidth limits found for container %s interface %s", containerName, iface)
	}

	if limit, exists := m.networkBandwidth[containerName][iface]; exists {
		return limit, nil
	}

	return nil, fmt.Errorf("no bandwidth limits found for container %s interface %s", containerName, iface)
}

func (m *MockLXCManager) UpdateNetworkBandwidthLimits(containerName, iface string, limits *common.BandwidthLimit) error {
	if _, exists := m.Containers[containerName]; !exists {
		return fmt.Errorf("container %s does not exist", containerName)
	}

	if m.networkBandwidth[containerName] == nil {
		m.networkBandwidth[containerName] = make(map[string]*common.BandwidthLimit)
	}

	m.networkBandwidth[containerName][iface] = limits

	logging.Debug("Updated network bandwidth limits",
		"container", containerName,
		"interface", iface,
		"ingress_rate", limits.IngressRate,
		"ingress_burst", limits.IngressBurst,
		"egress_rate", limits.EgressRate,
		"egress_burst", limits.EgressBurst,
	)

	return nil
}
