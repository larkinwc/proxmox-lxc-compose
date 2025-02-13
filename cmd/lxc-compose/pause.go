package main

import (
	"fmt"

	"proxmox-lxc-compose/pkg/container"

	"github.com/spf13/cobra"
)

func init() {
	var pauseCmd = &cobra.Command{
		Use:   "pause [container...]",
		Short: "Pause one or more containers",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			// Create container manager
			manager, err := container.NewLXCManager("/var/lib/lxc")
			if err != nil {
				return fmt.Errorf("failed to create container manager: %w", err)
			}

			// Pause each container
			for _, name := range args {
				fmt.Printf("Pausing container '%s'...\n", name)
				if err := manager.Pause(name); err != nil {
					return fmt.Errorf("failed to pause container '%s': %w", name, err)
				}
			}

			return nil
		},
	}

	rootCmd.AddCommand(pauseCmd)
}
