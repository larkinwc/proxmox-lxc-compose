package main

import (
	"fmt"
	"path/filepath"
	"strings"

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
			result, err := oci.ConvertOCIToLXC(imageName, outputPath)
			if err != nil {
				return fmt.Errorf("failed to convert image: %w", err)
			}
			fmt.Printf("Conversion complete: %s\n", result.OutputPath)
			fmt.Printf("  distro: %s, log symlinks fixed: %d\n",
				result.PostProcess.Distro, result.PostProcess.LogLinksFixed)
			if result.InitWrapperPath != "" {
				fmt.Printf("  init command: %s (runs: %s)\n",
					result.InitWrapperPath,
					strings.Join(append(append([]string{}, result.Entrypoint...), result.Command...), " "))
			}
			return nil
		},
	}

	convertCmd.Flags().StringVarP(&outputPath, "output", "o", "", "Output path for the LXC template")
	rootCmd.AddCommand(convertCmd)
}
