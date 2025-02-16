package main

import (
	"fmt"
	"os"
	"text/tabwriter"

	"proxmox-lxc-compose/pkg/container"

	"github.com/spf13/cobra"
)

func init() {
	var psCmd = &cobra.Command{
		Use:   "ps",
		Short: "List containers",
		RunE: func(_ *cobra.Command, _ []string) error {
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
