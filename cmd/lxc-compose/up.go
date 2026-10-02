package main

import (
	"fmt"
	"strings"

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
	upCmd.Flags().Bool("recreate", false, "Replace mapped containers at their existing VMIDs")
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

	services, err := selectedServices(compose, args)
	if err != nil {
		return err
	}
	forceConvert, recreate := false, false
	if cmd != nil {
		force, _ := cmd.Flags().GetBool("force-convert")
		pull, _ := cmd.Flags().GetBool("pull")
		forceConvert = force || pull
		recreate, _ = cmd.Flags().GetBool("recreate")
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
	inUse := make([]int, 0, len(infos))
	for _, info := range infos {
		existing[info.VMID] = info
		inUse = append(inUse, info.VMID)
	}

	type servicePlan struct {
		name     string
		vmid     int
		digest   string
		fields   map[string]string
		opts     proxmox.CreateOptions
		template preparedTemplate
		exists   bool
		status   proxmox.Status
	}
	plans := make([]servicePlan, 0, len(services))
	mappings := store.All()
	// Validate every selected service before changing any container or mapping.
	for _, name := range services {
		svc := compose.Services[name]
		opts, err := proxmox.Translate(&svc, translateOptions(name))
		if err != nil {
			return fmt.Errorf("failed to translate config for '%s': %w", name, err)
		}
		digest, fields, err := desiredConfigDigest(svc, opts)
		if err != nil {
			return fmt.Errorf("failed to fingerprint config for '%s': %w", name, err)
		}
		vmid, mapped := store.Get(name)
		info, exists := existing[vmid]
		exists = mapped && exists
		record, recorded := state.services[name]
		if mapped && exists {
			if err := validateMappedContainer(name, vmid, info, record, recorded, mappings); err != nil {
				return err
			}
			if forceConvert && !recreate {
				return fmt.Errorf("service '%s' already exists; --pull/--force-convert requires --recreate", name)
			}
		}
		if mapped && !recreate {
			if !recorded || record.Digest == "" {
				return fmt.Errorf("service '%s' has a legacy VMID mapping without desired state; use --recreate", name)
			}
			if record.VMID != vmid {
				return fmt.Errorf("service '%s' VMID mapping differs from recorded deployment; refusing to modify container", name)
			}
			if record.Digest != digest {
				difference := "field-level differences unavailable in legacy deployment state"
				if len(record.Fields) > 0 {
					difference = "changed fields: " + strings.Join(changedDesiredFields(record.Fields, fields), ", ")
				}
				return fmt.Errorf("desired configuration for service '%s' changed (%s); use --recreate", name, difference)
			}
		}
		if exists && info.Status != proxmox.StatusRunning && info.Status != proxmox.StatusStopped && info.Status != proxmox.StatusPaused {
			return fmt.Errorf("container '%s' (VMID %d) has unknown status %q; refusing to modify it", name, vmid, info.Status)
		}
		plans = append(plans, servicePlan{name: name, vmid: vmid, digest: digest, fields: fields, opts: opts, exists: exists, status: info.Status})
	}
	// Template resolution and translation must succeed before recreation can
	// stop or destroy an existing service, including multi-service requests.
	for i := range plans {
		plan := &plans[i]
		if plan.exists && !recreate {
			continue
		}
		svc := compose.Services[plan.name]
		tmpl, err := prepareTemplate(plan.name, svc.Image, forceConvert)
		if err != nil {
			return err
		}
		topts := translateOptions(plan.name)
		topts.OSTemplate = tmpl.OSTemplate
		opts, err := proxmox.Translate(&svc, topts)
		if err != nil {
			return fmt.Errorf("failed to translate config for '%s': %w", plan.name, err)
		}
		if opts.OSTemplate == "" {
			return fmt.Errorf("service '%s' has no usable template; refusing to provision or recreate", plan.name)
		}
		if err := templateReadinessFn(opts.OSTemplate); err != nil {
			return fmt.Errorf("template for service '%s' is not ready: %w", plan.name, err)
		}
		plan.opts, plan.template = opts, tmpl
	}

	for _, plan := range plans {
		name, vmid := plan.name, plan.vmid
		if plan.exists && !recreate {
			switch plan.status {
			case proxmox.StatusRunning:
				fmt.Printf("Container '%s' (VMID %d) is already running; unchanged.\n", name, vmid)
			case proxmox.StatusPaused:
				fmt.Printf("Container '%s' (VMID %d) is paused; leaving paused. Use unpause to resume.\n", name, vmid)
			case proxmox.StatusStopped:
				fmt.Printf("Starting container '%s' (VMID %d)...\n", name, vmid)
				if err := backend.Start(vmid); err != nil {
					return fmt.Errorf("failed to start container '%s': %w", name, err)
				}
			}
			continue
		}
		vmid, err = store.Assign(name, inUse)
		if err != nil {
			return fmt.Errorf("failed to allocate VMID for '%s': %w", name, err)
		}
		inUse = append(inUse, vmid)
		if plan.exists {
			// Recheck ownership immediately before the destructive boundary.
			current, err := backend.List()
			if err != nil {
				return fmt.Errorf("failed to verify container '%s': %w", name, err)
			}
			found := false
			for _, info := range current {
				if info.VMID == vmid {
					record, recorded := state.services[name]
					if err := validateMappedContainer(name, vmid, info, record, recorded, mappings); err != nil {
						return err
					}
					found = true
					if info.Status != proxmox.StatusStopped {
						if err := backend.Stop(vmid); err != nil {
							return fmt.Errorf("failed to stop container '%s' for recreation: %w", name, err)
						}
					}
					if err := backend.Destroy(vmid); err != nil {
						return fmt.Errorf("failed to destroy container '%s' for recreation: %w", name, err)
					}
					break
				}
			}
			if !found {
				fmt.Printf("Container '%s' (VMID %d) disappeared; provisioning its mapped VMID.\n", name, vmid)
			}
		}
		// An interrupted provisioning attempt must not inherit a successful
		// digest from the container it replaced.
		if _, recorded := state.services[name]; recorded {
			if err := state.remove(name); err != nil {
				return fmt.Errorf("failed to clear deployment state for '%s': %w", name, err)
			}
		}
		fmt.Printf("Creating container '%s' (VMID %d)...\n", name, vmid)
		if err := backend.Create(vmid, plan.opts); err != nil {
			return fmt.Errorf("failed to create container '%s': %w", name, err)
		}
		if plan.template.InitCmd != "" {
			if err := backend.SetInitCommand(vmid, plan.template.InitCmd); err != nil {
				return fmt.Errorf("failed to set init command for '%s': %w", name, err)
			}
		}
		fmt.Printf("Starting container '%s' (VMID %d)...\n", name, vmid)
		if err := backend.Start(vmid); err != nil {
			return fmt.Errorf("failed to start container '%s': %w", name, err)
		}
		state.services[name] = deploymentRecord{VMID: vmid, Digest: plan.digest, Hostname: plan.opts.Hostname, Fields: plan.fields}
		if err := state.save(); err != nil {
			return fmt.Errorf("failed to save deployment state for '%s': %w", name, err)
		}
	}
	return nil
}
