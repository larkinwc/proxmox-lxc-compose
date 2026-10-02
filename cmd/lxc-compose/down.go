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
	services, err := selectedServices(compose, args)
	if err != nil {
		return err
	}

	backend, err := newBackend()
	if err != nil {
		return err
	}
	store, err := newVMIDStore()
	if err != nil {
		return err
	}

	state, err := loadDeploymentState()
	if err != nil {
		return err
	}
	infos, err := backend.List()
	if err != nil {
		return fmt.Errorf("failed to list containers: %w", err)
	}
	existing := make(map[int]proxmox.ContainerInfo, len(infos))
	for _, info := range infos {
		existing[info.VMID] = info
	}
	mappings := store.All()
	// Reject unknown names, absent mappings and ownership mismatches for the
	// entire request before stopping or removing its first service.
	for _, name := range services {
		vmid, ok := store.Get(name)
		if !ok {
			return fmt.Errorf("no VMID mapping found for service '%s' (was it started?)", name)
		}
		if info, exists := existing[vmid]; exists {
			record, recorded := state.services[name]
			if err := validateMappedContainer(name, vmid, info, record, recorded, mappings); err != nil {
				return err
			}
		} else if !removeContainers {
			return fmt.Errorf("container '%s' (VMID %d) is missing", name, vmid)
		}
	}

	for _, name := range services {

		vmid, ok := store.Get(name)
		if !ok {
			return fmt.Errorf("no VMID mapping found for service '%s' (was it started?)", name)
		}

		if _, exists := existing[vmid]; exists {
			fmt.Printf("Shutting down container '%s' (VMID %d)...\n", name, vmid)
			if err := backend.Shutdown(vmid); err != nil {
				return fmt.Errorf("failed to stop container '%s': %w", name, err)
			}
		}

		if removeContainers {
			if _, exists := existing[vmid]; exists {
				// Recheck the backend identity at the destructive boundary.
				current, err := backend.List()
				if err != nil {
					return fmt.Errorf("failed to verify container '%s': %w", name, err)
				}
				for _, info := range current {
					if info.VMID != vmid {
						continue
					}
					record, recorded := state.services[name]
					if err := validateMappedContainer(name, vmid, info, record, recorded, mappings); err != nil {
						return err
					}
					fmt.Printf("Removing container '%s' (VMID %d)...\n", name, vmid)
					if err := backend.Destroy(vmid); err != nil {
						return fmt.Errorf("failed to remove container '%s': %w", name, err)
					}
					break
				}
			}
			if err := state.remove(name); err != nil {
				return fmt.Errorf("failed to remove deployment state for '%s': %w", name, err)
			}
			if err := store.Remove(name); err != nil {
				return fmt.Errorf("failed to remove VMID mapping for '%s': %w", name, err)
			}
		}
	}

	return nil
}
