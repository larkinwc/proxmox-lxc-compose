package main

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"
)

func init() {
	var psCmd = &cobra.Command{
		Use:   "ps",
		Short: "List containers",
		RunE: func(_ *cobra.Command, _ []string) error {
			backend, err := newBackend()
			if err != nil {
				return err
			}
			// The VMID store only supplies friendly names; if it's
			// unreadable, still list containers from Proxmox.
			store, err := newVMIDStore()
			if err != nil {
				fmt.Fprintf(os.Stderr, "warning: failed to load VMID store: %v\n", err)
				store = nil
			}

			containers, err := backend.List()
			if err != nil {
				return fmt.Errorf("failed to list containers: %w", err)
			}

			// Create tabwriter for formatted output. Prefer the compose
			// service name from the VMID store, falling back to the Proxmox
			// container name.
			w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
			fmt.Fprintln(w, "NAME\tVMID\tSTATE")
			for _, c := range containers {
				name := c.Name
				if store != nil {
					if mapped, ok := store.Lookup(c.VMID); ok {
						name = mapped
					}
				}
				fmt.Fprintf(w, "%s\t%d\t%s\n", name, c.VMID, c.Status)
			}
			return w.Flush()
		},
	}

	rootCmd.AddCommand(psCmd)
}
