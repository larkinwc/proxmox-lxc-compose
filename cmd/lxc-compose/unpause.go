package main

import (
	"fmt"

	"proxmox-lxc-compose/pkg/container"

	"github.com/spf13/cobra"
)

func init() {
	var unpauseCmd = &cobra.Command{
		Use:   "unpause [container...]",
		Short: "Unpause one or more containers",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			// Create container manager
			manager, err := container.NewLXCManager("/var/lib/lxc")
			if err != nil {
				return fmt.Errorf("failed to create container manager: %w", err)
			}

			// Resume each container
			for _, name := range args {
				fmt.Printf("Resuming container '%s'...\n", name)
				if err := manager.Resume(name); err != nil {
					return fmt.Errorf("failed to resume container '%s': %w", name, err)
				}
			}

			return nil
		},
	}

	rootCmd.AddCommand(unpauseCmd)
}
