package main

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"
)

func init() {
	var templateCmd = &cobra.Command{
		Use:   "template",
		Short: "Manage container templates",
		Long:  `Create, list, delete, and instantiate container templates.`,
	}

	templateCmd.AddCommand(templateCreateCmd())
	templateCmd.AddCommand(templateListCmd())
	templateCmd.AddCommand(templateDeleteCmd())
	templateCmd.AddCommand(templateApplyCmd())

	rootCmd.AddCommand(templateCmd)
}

func templateCreateCmd() *cobra.Command {
	var description string
	cmd := &cobra.Command{
		Use:   "create [container] [template]",
		Short: "Create a template from an existing container",
		Args:  cobra.ExactArgs(2),
		RunE: func(_ *cobra.Command, args []string) error {
			manager, err := newManager()
			if err != nil {
				return err
			}

			containerName, templateName := args[0], args[1]
			if err := manager.CreateTemplate(containerName, templateName, description); err != nil {
				return fmt.Errorf("failed to create template '%s': %w", templateName, err)
			}
			fmt.Printf("Created template '%s' from container '%s'\n", templateName, containerName)
			return nil
		},
	}
	cmd.Flags().StringVarP(&description, "description", "d", "", "Template description")
	return cmd
}

func templateListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "ls",
		Short: "List available templates",
		RunE: func(_ *cobra.Command, _ []string) error {
			manager, err := newManager()
			if err != nil {
				return err
			}

			templates, err := manager.ListTemplates()
			if err != nil {
				return fmt.Errorf("failed to list templates: %w", err)
			}

			w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
			fmt.Fprintln(w, "NAME\tDESCRIPTION\tCREATED")
			for _, tmpl := range templates {
				fmt.Fprintf(w, "%s\t%s\t%s\n", tmpl.Name, tmpl.Description, tmpl.CreatedAt.Format("2006-01-02 15:04:05"))
			}
			return w.Flush()
		},
	}
}

func templateDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "rm [template]",
		Short: "Delete a template",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			manager, err := newManager()
			if err != nil {
				return err
			}

			templateName := args[0]
			if err := manager.DeleteTemplate(templateName); err != nil {
				return fmt.Errorf("failed to delete template '%s': %w", templateName, err)
			}
			fmt.Printf("Deleted template '%s'\n", templateName)
			return nil
		},
	}
}

func templateApplyCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "apply [template] [container]",
		Short: "Create a new container from a template",
		Args:  cobra.ExactArgs(2),
		RunE: func(_ *cobra.Command, args []string) error {
			manager, err := newManager()
			if err != nil {
				return err
			}

			templateName, containerName := args[0], args[1]
			if err := manager.CreateFromTemplate(templateName, containerName, nil); err != nil {
				return fmt.Errorf("failed to create container '%s' from template '%s': %w", containerName, templateName, err)
			}
			fmt.Printf("Created container '%s' from template '%s'\n", containerName, templateName)
			return nil
		},
	}
}
