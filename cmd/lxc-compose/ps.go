package main

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/larkinwc/proxmox-lxc-compose/pkg/config"
	"github.com/larkinwc/proxmox-lxc-compose/pkg/container"

	"github.com/spf13/cobra"
)

func init() {
	var psCmd = &cobra.Command{
		Use:   "ps",
		Short: "List containers",
		RunE: func(cmd *cobra.Command, _ []string) error {
			// Get config file from flag or use default
			configFile := cmd.Flag("config").Value.String()
			if configFile == "" {
				configFile = "lxc-compose.yml"
			}

			// Load and validate configuration if file exists
			if _, err := os.Stat(configFile); err == nil {
				cfg, err := config.Load(configFile)
				if err != nil {
					return fmt.Errorf("failed to load config: %w", err)
				}

				// Validate configuration
				if err := config.ValidateConfig(cfg); err != nil {
					return fmt.Errorf("configuration validation failed: %w", err)
				}
			}

			// Create container manager
			manager, err := container.NewLXCManager("/var/lib/lxc")
			if err != nil {
				return fmt.Errorf("failed to create container manager: %w", err)
			}

			// Get list of containers
			containers, err := manager.List()
			if err != nil {
				return fmt.Errorf("failed to list containers: %w", err)
			}

			// Create tabwriter for formatted output
			w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
			fmt.Fprintln(w, "NAME\tSTATE")
			for _, c := range containers {
				fmt.Fprintf(w, "%s\t%s\n", c.Name, c.State)
			}
			w.Flush()

			return nil
		},
	}

	rootCmd.AddCommand(psCmd)
}
