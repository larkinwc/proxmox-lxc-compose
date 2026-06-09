package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

func init() {
	var pauseCmd = &cobra.Command{
		Use:   "pause [container...]",
		Short: "Pause one or more containers",
		Args:  cobra.MinimumNArgs(1),
		RunE:  pauseCmdRunE,
	}

	rootCmd.AddCommand(pauseCmd)
}

func pauseCmdRunE(_ *cobra.Command, args []string) error {
	backend, err := newBackend()
	if err != nil {
		return err
	}
	store, err := newVMIDStore()
	if err != nil {
		return err
	}

	// Pause each container
	for _, name := range args {
		vmid, ok := store.Get(name)
		if !ok {
			return fmt.Errorf("no VMID mapping found for service '%s'", name)
		}
		fmt.Printf("Pausing container '%s' (VMID %d)...\n", name, vmid)
		if err := backend.Suspend(vmid); err != nil {
			return fmt.Errorf("failed to pause container '%s': %w", name, err)
		}
	}

	return nil
}
