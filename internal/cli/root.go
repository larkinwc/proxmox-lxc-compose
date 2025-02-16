package cli

import (
	"fmt"
	"os"

	"github.com/larkinwc/proxmox-lxc-compose/pkg/logging"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfgFile     string
	debugMode   bool
	development bool
	rootCmd     *cobra.Command
)

func init() {
	cobra.OnInitialize(initConfig)
	rootCmd = &cobra.Command{
		Use:   "lxc-compose",
		Short: "A docker-compose like tool for managing LXC containers in Proxmox",
	}
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.lxc-compose.yaml)")
	rootCmd.PersistentFlags().BoolVar(&debugMode, "debug", false, "enable debug mode")
	rootCmd.PersistentFlags().BoolVar(&development, "dev", false, "enable development mode")
}

func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		home, err := os.UserHomeDir()
		cobra.CheckErr(err)
		viper.AddConfigPath(home)
		viper.SetConfigType("yaml")
		viper.SetConfigName(".lxc-compose")
	}

	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err == nil {
		fmt.Fprintln(os.Stderr, "Using config file:", viper.ConfigFileUsed())
	}

	if debugMode {
		if err := logging.Init(logging.Config{
			Level:       "debug",
			Development: true,
		}); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to initialize logger: %v\n", err)
		}
	}

	if development {
		if err := logging.Init(logging.Config{
			Level:       "debug",
			Development: true,
		}); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to initialize logger: %v\n", err)
		}
	}
}

// Execute executes the root command
func Execute() error {
	return rootCmd.Execute()
}

// GetRootCmd returns the root command for adding subcommands
func GetRootCmd() *cobra.Command {
	return rootCmd
}
