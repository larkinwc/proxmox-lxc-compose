package main

import (
	"fmt"

	"proxmox-lxc-compose/pkg/common"
	"proxmox-lxc-compose/pkg/container"

	"github.com/spf13/cobra"
)

var configFile string

func init() {
	var upCmd = &cobra.Command{
		Use:   "up [service...]",
		Short: "Create and start containers",
		Long: `Create and start containers defined in the lxc-compose.yml file.
If service names are provided, only those services will be started.`,
		RunE: upCmdRunE,
	}

	upCmd.Flags().StringVarP(&configFile, "file", "f", "", "Specify an alternate compose file (default: lxc-compose.yml)")
	rootCmd.AddCommand(upCmd)
}

func upCmdRunE(_ *cobra.Command, args []string) error {
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

	// Start all or specified services
	services := args
	if len(services) == 0 {
		for name := range compose.Services {
			services = append(services, name)
		}
	}

	for _, name := range services {
		svcCfg, ok := compose.Services[name]
		if !ok {
			return fmt.Errorf("service '%s' not found in config", name)
		}

		fmt.Printf("Creating container '%s'...\n", name)
		if err := manager.Create(name, &svcCfg); err != nil {
			return fmt.Errorf("failed to create container '%s': %w", name, err)
		}

		fmt.Printf("Starting container '%s'...\n", name)
		if err := manager.Start(name); err != nil {
			return fmt.Errorf("failed to start container '%s': %w", name, err)
		}
	}

	return nil
}
