package main

import (
	"fmt"

	"github.com/larkinwc/proxmox-lxc-compose/pkg/common"

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
	compose, err := common.Load(resolveConfigFile())
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}
	if compose == nil || len(compose.Services) == 0 {
		return fmt.Errorf("no services defined in config")
	}

	backend, err := newBackend()
	if err != nil {
		return err
	}
	store, err := newVMIDStore()
	if err != nil {
		return err
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

		vmid, ok := store.Get(name)
		if !ok {
			return fmt.Errorf("no VMID mapping found for service '%s' (was it started?)", name)
		}

		fmt.Printf("Shutting down container '%s' (VMID %d)...\n", name, vmid)
		if err := backend.Shutdown(vmid); err != nil {
			return fmt.Errorf("failed to stop container '%s': %w", name, err)
		}

		if removeContainers {
			fmt.Printf("Removing container '%s' (VMID %d)...\n", name, vmid)
			if err := backend.Destroy(vmid); err != nil {
				return fmt.Errorf("failed to remove container '%s': %w", name, err)
			}
			if err := store.Remove(name); err != nil {
				return fmt.Errorf("failed to remove VMID mapping for '%s': %w", name, err)
			}
		}
	}

	return nil
}
