package main

import (
	"fmt"
	"path/filepath"

	"github.com/larkinwc/proxmox-lxc-compose/pkg/oci"

	"github.com/spf13/cobra"
)

func init() {
	var outputPath string
	var convertCmd = &cobra.Command{
		Use:   "convert [image]",
		Short: "Convert an OCI image to LXC template",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			imageName := args[0]
			if outputPath == "" {
				outputPath = filepath.Join("templates", fmt.Sprintf("%s.tar.gz", imageName))
			}

			fmt.Printf("Converting image '%s' to LXC template at '%s'...\n", imageName, outputPath)
			if err := oci.ConvertOCIToLXC(imageName, outputPath); err != nil {
				return fmt.Errorf("failed to convert image: %w", err)
			}
			fmt.Println("Conversion complete!")
			return nil
		},
	}

	convertCmd.Flags().StringVarP(&outputPath, "output", "o", "", "Output path for the LXC template")
	rootCmd.AddCommand(convertCmd)
}
