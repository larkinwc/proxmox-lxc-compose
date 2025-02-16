package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/larkinwc/proxmox-lxc-compose/pkg/errors"
	"github.com/larkinwc/proxmox-lxc-compose/pkg/logging"
	"github.com/larkinwc/proxmox-lxc-compose/pkg/oci"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(imagesCmd)
	imagesCmd.AddCommand(pullCmd)
	imagesCmd.AddCommand(pushCmd)
	imagesCmd.AddCommand(listCmd)
	imagesCmd.AddCommand(removeCmd)
}

var imagesCmd = &cobra.Command{
	Use:   "images",
	Short: "Manage OCI images",
	Long:  `Manage OCI images including pulling, pushing, listing and removing images`,
}

var pullCmd = &cobra.Command{
	Use:   "pull [registry/repository:tag]",
	Short: "Pull an image from a registry",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ref, err := oci.ParseImageReference(args[0])
		if err != nil {
			return errors.Wrap(err, errors.ErrValidation, "invalid image reference")
		}

		logging.Info("Starting image pull",
			"image", args[0],
			"ref", ref)

		manager, err := getRegistryManager()
		if err != nil {
			return errors.Wrap(err, errors.ErrSystem, "failed to initialize registry manager")
		}

		if err := manager.Pull(cmd.Context(), ref); err != nil {
			if errors.IsType(err, errors.ErrRegistry) {
				logging.Error("Failed to pull image",
					"image", args[0],
					"error", err)
			}
			return err
		}

		return nil
	},
}

var pushCmd = &cobra.Command{
	Use:   "push [registry/repository:tag]",
	Short: "Push an image to a registry",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ref, err := oci.ParseImageReference(args[0])
		if err != nil {
			return errors.Wrap(err, errors.ErrValidation, "invalid image reference")
		}

		logging.Info("Starting image push",
			"image", args[0],
			"ref", ref)

		manager, err := getRegistryManager()
		if err != nil {
			return errors.Wrap(err, errors.ErrSystem, "failed to initialize registry manager")
		}

		if err := manager.Push(cmd.Context(), ref); err != nil {
			if errors.IsType(err, errors.ErrRegistry) {
				logging.Error("Failed to push image",
					"image", args[0],
					"error", err)
			}
			return err
		}

		return nil
	},
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List locally stored images",
	RunE: func(cmd *cobra.Command, _ []string) error {
		manager, err := getRegistryManager()
		if err != nil {
			return errors.Wrap(err, errors.ErrSystem, "failed to initialize registry manager")
		}

		images, err := manager.List(cmd.Context())
		if err != nil {
			logging.Error("Failed to list images", "error", err)
			return err
		}

		if len(images) == 0 {
			fmt.Println("No images found")
			return nil
		}

		fmt.Println("REPOSITORY\t\tTAG\t\tDIGEST")
		for _, img := range images {
			digest := img.Digest
			if digest == "" {
				digest = "-"
			}
			fmt.Printf("%s/%s\t\t%s\t\t%s\n",
				img.Registry, img.Repository, img.Tag, digest)
		}
		return nil
	},
}

var removeCmd = &cobra.Command{
	Use:   "remove [registry/repository:tag]",
	Short: "Remove an image from local storage",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ref, err := oci.ParseImageReference(args[0])
		if err != nil {
			return errors.Wrap(err, errors.ErrValidation, "invalid image reference")
		}

		logging.Info("Removing image",
			"image", args[0],
			"ref", ref)

		manager, err := getRegistryManager()
		if err != nil {
			return errors.Wrap(err, errors.ErrSystem, "failed to initialize registry manager")
		}

		if err := manager.Delete(cmd.Context(), ref); err != nil {
			logging.Error("Failed to remove image",
				"image", args[0],
				"error", err)
			return err
		}

		return nil
	},
}

func getRegistryManager() (*oci.RegistryManager, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, errors.Wrap(err, errors.ErrSystem, "failed to get home directory")
	}

	storageDir := filepath.Join(homeDir, ".lxc-compose", "images")
	manager, err := oci.NewRegistryManager(storageDir)
	if err != nil {
		return nil, errors.Wrap(err, errors.ErrSystem, "failed to create registry manager")
	}

	return manager, nil
}
