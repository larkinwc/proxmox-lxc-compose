package main

import (
	"fmt"

	"github.com/larkinwc/proxmox-lxc-compose/pkg/common"
	"github.com/larkinwc/proxmox-lxc-compose/pkg/container"

	"github.com/spf13/cobra"
)

var removeContainers bool

func init() {
	var downCmd = &cobra.Command{
		Use:   "down [service...]",
		Short: "Stop and optionally remove containers",
		Long: `Stop containers defined in the lxc-compose.yml file.
If service names are provided, only those services will be stopped.
Use --rm to also remove the containers.`,
		RunE: downCmdRunE,
	}

	downCmd.Flags().StringVarP(&configFile, "file", "f", "", "Specify an alternate compose file (default: lxc-compose.yml)")
	downCmd.Flags().BoolVar(&removeContainers, "rm", false, "Remove containers after stopping")
	rootCmd.AddCommand(downCmd)
}

func downCmdRunE(_ *cobra.Command, args []string) error {
	// Load configuration
	cfg, err := common.Load(configFile)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Convert to compose config type
	var compose common.ComposeConfig
	compose.Services = make(map[string]common.Container)
	if cfg != nil {
		compose.Services["default"] = cfg.Services["default"]
	}

	// Create container manager
	manager, err := container.NewLXCManager("/var/lib/lxc")
	if err != nil {
		return fmt.Errorf("failed to create container manager: %w", err)
	}

	// Stop all or specified services
	services := args
	if len(services) == 0 {
		for name := range compose.Services {
			services = append(services, name)
		}
	}

	for _, name := range services {
		if _, ok := compose.Services[name]; !ok {
			return fmt.Errorf("service '%s' not found in config", name)
		}

		fmt.Printf("Stopping container '%s'...\n", name)
		if err := manager.Stop(name); err != nil {
			return fmt.Errorf("failed to stop container '%s': %w", name, err)
		}

		if removeContainers {
			fmt.Printf("Removing container '%s'...\n", name)
			if err := manager.Remove(name); err != nil {
				return fmt.Errorf("failed to remove container '%s': %w", name, err)
			}
		}
	}

	return nil
}
