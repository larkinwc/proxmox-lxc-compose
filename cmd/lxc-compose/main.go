package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "lxc-compose",
	Short: "A docker-compose like tool for managing LXC containers",
	Long: `lxc-compose is a CLI tool that allows you to manage LXC containers 
using a docker-compose like syntax. It supports creating, starting, stopping,
and updating LXC containers based on a lxc-compose.yml configuration file.`,
}

func init() {
	// Commands will be added here
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
