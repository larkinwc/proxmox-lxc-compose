package main

import (
	"fmt"
	"os"

	"proxmox-lxc-compose/pkg/logging"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfgFile     string
	debugMode   bool
	development bool
)

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.lxc-compose.yaml)")
	rootCmd.PersistentFlags().BoolVar(&debugMode, "debug", false, "enable debug logging")
	rootCmd.PersistentFlags().BoolVar(&development, "dev", false, "enable development mode")
}

func initConfig() {
	// Initialize logging first
	logLevel := "info"
	if debugMode {
		logLevel = "debug"
	}

	if err := logging.Init(logging.Config{
		Level:       logLevel,
		Development: development,
	}); err != nil {
		fmt.Printf("Error initializing logger: %v\n", err)
		os.Exit(1)
	}

	// Load configuration file if specified
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		home, err := os.UserHomeDir()
		cobra.CheckErr(err)

		viper.AddConfigPath(home)
		viper.SetConfigType("yaml")
		viper.SetConfigName(".lxc-compose")
	}

	if err := viper.ReadInConfig(); err == nil {
		logging.Info("Using config file", "path", viper.ConfigFileUsed())
	}
}

var rootCmd = &cobra.Command{
	Use:   "lxc-compose",
	Short: "Manage LXC containers using docker-compose like syntax",
	Long: `lxc-compose is a CLI tool that allows you to manage LXC containers 
using a docker-compose like syntax. It supports creating, starting, stopping, 
and managing containers defined in a YAML configuration file.`,
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
