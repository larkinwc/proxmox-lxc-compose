package main

import (
	"fmt"

	"github.com/larkinwc/proxmox-lxc-compose/pkg/config"
	"github.com/larkinwc/proxmox-lxc-compose/pkg/container"

	"github.com/spf13/cobra"
)

var removeContainers bool

func init() {
	var configFile string

	var downCmd = &cobra.Command{
		Use:   "down [service...]",
		Short: "Stop and optionally remove containers",
		Long: `Stop containers defined in the lxc-compose.yml file.
If service names are provided, only those services will be stopped.
Use --rm to also remove the containers.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return downCmdRunE(cmd, args, configFile)
		},
	}

	downCmd.Flags().StringVarP(&configFile, "file", "f", "", "Specify an alternate compose file (default: lxc-compose.yml)")
	downCmd.Flags().BoolVar(&removeContainers, "rm", false, "Remove containers after stopping")
	rootCmd.AddCommand(downCmd)
}

func downCmdRunE(_ *cobra.Command, args []string, configFile string) error {
	// Use default config file if not specified
	if configFile == "" {
		configFile = "lxc-compose.yml"
	}

	// Load configuration
	cfg, err := config.Load(configFile)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Use the loaded config directly
	compose := cfg

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
			// If container is already stopped, that's fine, continue to removal if requested
			if !removeContainers {
				return fmt.Errorf("failed to stop container '%s': %w", name, err)
			}
			// For removal operations, log the stop error but continue
			fmt.Printf("Warning: %v\n", err)
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
