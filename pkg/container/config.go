package container

import (
	"fmt"

	"github.com/larkinwc/proxmox-lxc-compose/pkg/config"
	"github.com/larkinwc/proxmox-lxc-compose/pkg/proxmox"
)

func (m *LXCManager) applyCPUConfig(name string, cpu *config.CPUConfig) error {
	if cpu == nil {
		return nil
	}
	// Apply CPU configuration
	if err := m.client.SetContainerOptions(name, proxmox.ContainerOptions{
		Cores: cpu.Cores,
	}); err != nil {
		return fmt.Errorf("failed to set container options: %w", err)
	}
	return nil
}

func (m *LXCManager) applyMemoryConfig(name string, memory *config.MemoryConfig) error {
	if memory == nil {
		return nil
	}
	// Apply memory configuration
	if err := m.client.SetContainerOptions(name, proxmox.ContainerOptions{
		Memory: memory.Limit,
	}); err != nil {
		return fmt.Errorf("failed to set container options: %w", err)
	}
	return nil
}
