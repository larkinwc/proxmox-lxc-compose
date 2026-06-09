package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

func init() {
	var unpauseCmd = &cobra.Command{
		Use:   "unpause [container...]",
		Short: "Unpause one or more containers",
		Args:  cobra.MinimumNArgs(1),
		RunE:  unpauseCmdRunE,
	}

	rootCmd.AddCommand(unpauseCmd)
}

func unpauseCmdRunE(_ *cobra.Command, args []string) error {
	backend, err := newBackend()
	if err != nil {
		return err
	}
	store, err := newVMIDStore()
	if err != nil {
		return err
	}

	// Resume each container
	for _, name := range args {
		vmid, ok := store.Get(name)
		if !ok {
			return fmt.Errorf("no VMID mapping found for service '%s'", name)
		}
		fmt.Printf("Resuming container '%s' (VMID %d)...\n", name, vmid)
		if err := backend.Resume(vmid); err != nil {
			return fmt.Errorf("failed to resume container '%s': %w", name, err)
		}
	}

	return nil
}
