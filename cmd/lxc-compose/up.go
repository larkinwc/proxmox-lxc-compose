package main

import (
	"fmt"
	"sort"

	"github.com/larkinwc/proxmox-lxc-compose/pkg/common"
	"github.com/larkinwc/proxmox-lxc-compose/pkg/proxmox"

	"github.com/spf13/cobra"
)

var configFile string

// resolveConfigFile returns the compose file path, defaulting to lxc-compose.yml
func resolveConfigFile() string {
	if configFile != "" {
		return configFile
	}
	return "lxc-compose.yml"
}

func init() {
	var upCmd = &cobra.Command{
		Use:   "up [service...]",
		Short: "Create and start containers",
		Long: `Create and start containers defined in the lxc-compose.yml file.
If service names are provided, only those services will be started.`,
		RunE: upCmdRunE,
	}

	upCmd.Flags().StringVarP(&configFile, "file", "f", "", "Specify an alternate compose file (default: lxc-compose.yml)")
	upCmd.Flags().Bool("force-convert", false, "Re-convert OCI images even if a cached template exists")
	upCmd.Flags().Bool("pull", false, "Pull and re-convert OCI images, refreshing the cached template")
	rootCmd.AddCommand(upCmd)
}

func upCmdRunE(cmd *cobra.Command, args []string) error {
	// Load configuration
	compose, err := common.Load(resolveConfigFile())
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}
	if compose == nil || len(compose.Services) == 0 {
		return fmt.Errorf("no services defined in config")
	}

	// --pull and --force-convert both force re-conversion of OCI images.
	forceConvert := false
	if cmd != nil {
		force, _ := cmd.Flags().GetBool("force-convert")
		pull, _ := cmd.Flags().GetBool("pull")
		forceConvert = force || pull
	}

	backend, err := newBackend()
	if err != nil {
		return err
	}
	store, err := newVMIDStore()
	if err != nil {
		return err
	}

	// Start all or specified services. Sort the auto-selected set so VMID
	// assignment is deterministic regardless of Go's map iteration order.
	services := args
	if len(services) == 0 {
		for name := range compose.Services {
			services = append(services, name)
		}
		sort.Strings(services)
	}

	inUse, err := backendVMIDs(backend)
	if err != nil {
		return err
	}

	for _, name := range services {
		svcCfg, ok := compose.Services[name]
		if !ok {
			return fmt.Errorf("service '%s' not found in config", name)
		}

		// Map the service name to a stable Proxmox VMID.
		vmid, err := store.Assign(name, inUse)
		if err != nil {
			return fmt.Errorf("failed to allocate VMID for '%s': %w", name, err)
		}
		inUse = append(inUse, vmid)

		// Resolve the image into a pct-usable template (optionally converting
		// an OCI image), capturing any init command to reproduce.
		tmpl, err := prepareTemplate(name, svcCfg.Image, forceConvert)
		if err != nil {
			return err
		}

		// Translate the compose config into Proxmox create options.
		topts := translateOptions(name)
		if tmpl.OSTemplate != "" {
			topts.OSTemplate = tmpl.OSTemplate
		}
		opts, err := proxmox.Translate(&svcCfg, topts)
		if err != nil {
			return fmt.Errorf("failed to translate config for '%s': %w", name, err)
		}

		fmt.Printf("Creating container '%s' (VMID %d)...\n", name, vmid)
		if err := backend.Create(vmid, opts); err != nil {
			return fmt.Errorf("failed to create container '%s': %w", name, err)
		}

		// Reproduce the OCI image's entrypoint/command under LXC, if any.
		if tmpl.InitCmd != "" {
			if err := backend.SetInitCommand(vmid, tmpl.InitCmd); err != nil {
				return fmt.Errorf("failed to set init command for '%s': %w", name, err)
			}
		}

		fmt.Printf("Starting container '%s' (VMID %d)...\n", name, vmid)
		if err := backend.Start(vmid); err != nil {
			return fmt.Errorf("failed to start container '%s': %w", name, err)
		}
	}

	return nil
}
