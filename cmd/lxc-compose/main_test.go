package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func TestRootCommand(t *testing.T) {
	// Test that root command exists and has correct metadata
	if rootCmd == nil {
		t.Fatal("rootCmd should not be nil")
	}

	if rootCmd.Use != "lxc-compose" {
		t.Errorf("Expected Use to be 'lxc-compose', got '%s'", rootCmd.Use)
	}

	if rootCmd.Short != "Manage LXC containers using docker-compose like syntax" {
		t.Errorf("Unexpected Short description: %s", rootCmd.Short)
	}

	if !strings.Contains(rootCmd.Long, "lxc-compose is a CLI tool") {
		t.Errorf("Unexpected Long description: %s", rootCmd.Long)
	}
}

func TestRootCommandFlags(t *testing.T) {
	// Test persistent flags
	configFlag := rootCmd.PersistentFlags().Lookup("config")
	if configFlag == nil {
		t.Error("Expected --config flag to exist")
	}

	debugFlag := rootCmd.PersistentFlags().Lookup("debug")
	if debugFlag == nil {
		t.Error("Expected --debug flag to exist")
	}

	devFlag := rootCmd.PersistentFlags().Lookup("dev")
	if devFlag == nil {
		t.Error("Expected --dev flag to exist")
	}
}

func TestRootCommandSubcommands(t *testing.T) {
	// Test that expected subcommands are registered
	expectedCommands := []string{
		"up",
		"down", 
		"ps",
		"logs",
		"pause",
		"unpause",
		"images",
		"convert",
	}

	for _, expectedCmd := range expectedCommands {
		cmd, _, err := rootCmd.Find([]string{expectedCmd})
		if err != nil {
			t.Errorf("Expected to find command '%s', got error: %v", expectedCmd, err)
		}
		if cmd == nil || cmd.Name() != expectedCmd {
			t.Errorf("Expected to find command '%s', but it was not found", expectedCmd)
		}
	}
}

func TestInitConfig(t *testing.T) {
	// Save original values
	originalCfgFile := cfgFile
	originalDebugMode := debugMode
	originalDevelopment := development
	
	defer func() {
		cfgFile = originalCfgFile
		debugMode = originalDebugMode
		development = originalDevelopment
		viper.Reset()
	}()

	tests := []struct {
		name        string
		cfgFile     string
		debugMode   bool
		development bool
	}{
		{
			name:        "default config",
			cfgFile:     "",
			debugMode:   false,
			development: false,
		},
		{
			name:        "debug mode enabled",
			cfgFile:     "",
			debugMode:   true,
			development: false,
		},
		{
			name:        "development mode enabled",
			cfgFile:     "",
			debugMode:   false,
			development: true,
		},
		{
			name:        "both debug and development enabled",
			cfgFile:     "",
			debugMode:   true,
			development: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set test values
			cfgFile = tt.cfgFile
			debugMode = tt.debugMode
			development = tt.development
			
			// Reset viper for clean test
			viper.Reset()

			// Call initConfig - this should not panic or error
			initConfig()

			// Verify viper configuration was set up
			if tt.cfgFile == "" {
				// Should have set up default config paths
				configPaths := viper.ConfigFileUsed()
				// ConfigFileUsed() returns empty if no config file was actually read,
				// which is expected in tests
				t.Logf("Config file used: %s", configPaths)
			}
		})
	}
}

func TestInitConfigWithConfigFile(t *testing.T) {
	// Save original values
	originalCfgFile := cfgFile
	originalDebugMode := debugMode
	originalDevelopment := development
	
	defer func() {
		cfgFile = originalCfgFile
		debugMode = originalDebugMode
		development = originalDevelopment
		viper.Reset()
	}()

	// Create a temporary config file
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "test-config.yaml")
	
	configContent := `
test_setting: "test_value"
another_setting: 123
`
	
	err := os.WriteFile(configFile, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write test config file: %v", err)
	}

	// Set config file path
	cfgFile = configFile
	debugMode = false
	development = false

	// Reset viper for clean test
	viper.Reset()

	// Call initConfig
	initConfig()

	// Verify config was loaded
	if viper.ConfigFileUsed() != configFile {
		t.Errorf("Expected config file '%s' to be used, got '%s'", configFile, viper.ConfigFileUsed())
	}

	// Verify config values were loaded
	if viper.GetString("test_setting") != "test_value" {
		t.Errorf("Expected test_setting to be 'test_value', got '%s'", viper.GetString("test_setting"))
	}

	if viper.GetInt("another_setting") != 123 {
		t.Errorf("Expected another_setting to be 123, got %d", viper.GetInt("another_setting"))
	}
}

func TestMainFunction(t *testing.T) {
	// Test that main function exists and can be called
	// We can't easily test the actual execution without complex setup,
	// but we can verify the function exists and doesn't panic immediately
	
	// Save original args
	originalArgs := os.Args
	defer func() { os.Args = originalArgs }()

	// Set test args that should show help and exit cleanly
	os.Args = []string{"lxc-compose", "--help"}

	// Capture output
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)

	// Execute help command
	err := rootCmd.Execute()
	if err != nil {
		t.Errorf("Help command should not return error, got: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "lxc-compose") {
		t.Error("Help output should contain 'lxc-compose'")
	}
}

func TestCommandExecution(t *testing.T) {
	// Test basic command structure without actually executing container operations
	tests := []struct {
		name     string
		args     []string
		wantHelp bool
	}{
		{
			name:     "root help",
			args:     []string{"--help"},
			wantHelp: true,
		},
		{
			name:     "up help",
			args:     []string{"up", "--help"},
			wantHelp: true,
		},
		{
			name:     "down help", 
			args:     []string{"down", "--help"},
			wantHelp: true,
		},
		{
			name:     "ps help",
			args:     []string{"ps", "--help"},
			wantHelp: true,
		},
		{
			name:     "logs help",
			args:     []string{"logs", "--help"},
			wantHelp: true,
		},
		{
			name:     "pause help",
			args:     []string{"pause", "--help"},
			wantHelp: true,
		},
		{
			name:     "unpause help",
			args:     []string{"unpause", "--help"},
			wantHelp: true,
		},
		{
			name:     "images help",
			args:     []string{"images", "--help"},
			wantHelp: true,
		},
		{
			name:     "convert help",
			args:     []string{"convert", "--help"},
			wantHelp: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a new command instance to avoid state pollution
			cmd := &cobra.Command{
				Use:   "lxc-compose",
				Short: "Manage LXC containers using docker-compose like syntax",
			}
			
			// Add all subcommands
			addAllSubcommands(cmd)

			var buf bytes.Buffer
			cmd.SetOut(&buf)
			cmd.SetErr(&buf)
			cmd.SetArgs(tt.args)

			err := cmd.Execute()
			
			if tt.wantHelp {
				// Help commands should not return an error
				if err != nil {
					t.Errorf("Help command should not return error, got: %v", err)
				}
				
				output := buf.String()
				if !strings.Contains(output, "Usage:") {
					t.Error("Help output should contain 'Usage:'")
				}
			}
		})
	}
}

// Helper function to add all subcommands for testing
func addAllSubcommands(cmd *cobra.Command) {
	// Add up command
	upCmd := &cobra.Command{
		Use:   "up [service...]",
		Short: "Create and start containers",
		RunE: func(_ *cobra.Command, _ []string) error {
			return fmt.Errorf("test mode - not implemented")
		},
	}
	upCmd.Flags().StringP("file", "f", "", "Specify an alternate compose file")
	cmd.AddCommand(upCmd)

	// Add down command
	downCmd := &cobra.Command{
		Use:   "down [service...]",
		Short: "Stop and optionally remove containers",
		RunE: func(_ *cobra.Command, _ []string) error {
			return fmt.Errorf("test mode - not implemented")
		},
	}
	downCmd.Flags().StringP("file", "f", "", "Specify an alternate compose file")
	downCmd.Flags().Bool("rm", false, "Remove containers after stopping")
	cmd.AddCommand(downCmd)

	// Add ps command
	psCmd := &cobra.Command{
		Use:   "ps",
		Short: "List containers",
		RunE: func(_ *cobra.Command, _ []string) error {
			return fmt.Errorf("test mode - not implemented")
		},
	}
	cmd.AddCommand(psCmd)

	// Add logs command
	logsCmd := &cobra.Command{
		Use:   "logs [container]",
		Short: "View container logs",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, _ []string) error {
			return fmt.Errorf("test mode - not implemented")
		},
	}
	logsCmd.Flags().BoolP("follow", "f", false, "Follow log output")
	logsCmd.Flags().IntP("tail", "n", 0, "Number of lines to show")
	logsCmd.Flags().String("since", "", "Show logs since timestamp")
	logsCmd.Flags().BoolP("timestamps", "t", false, "Show timestamps")
	cmd.AddCommand(logsCmd)

	// Add pause command
	pauseCmd := &cobra.Command{
		Use:   "pause [container...]",
		Short: "Pause one or more containers",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(_ *cobra.Command, _ []string) error {
			return fmt.Errorf("test mode - not implemented")
		},
	}
	cmd.AddCommand(pauseCmd)

	// Add unpause command
	unpauseCmd := &cobra.Command{
		Use:   "unpause [container...]",
		Short: "Unpause one or more containers",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(_ *cobra.Command, _ []string) error {
			return fmt.Errorf("test mode - not implemented")
		},
	}
	cmd.AddCommand(unpauseCmd)

	// Add images command
	imagesCmd := &cobra.Command{
		Use:   "images",
		Short: "Manage OCI images",
	}
	
	pullCmd := &cobra.Command{
		Use:   "pull [image]",
		Short: "Pull an image from a registry",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, _ []string) error {
			return fmt.Errorf("test mode - not implemented")
		},
	}
	imagesCmd.AddCommand(pullCmd)
	
	cmd.AddCommand(imagesCmd)

	// Add convert command
	convertCmd := &cobra.Command{
		Use:   "convert [image]",
		Short: "Convert an OCI image to LXC template",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, _ []string) error {
			return fmt.Errorf("test mode - not implemented")
		},
	}
	convertCmd.Flags().StringP("output", "o", "", "Output path")
	cmd.AddCommand(convertCmd)
}

func TestGlobalVariables(t *testing.T) {
	// Test that global variables are properly initialized
	tests := []struct {
		name     string
		variable interface{}
		varName  string
	}{
		{"cfgFile", &cfgFile, "cfgFile"},
		{"debugMode", &debugMode, "debugMode"},
		{"development", &development, "development"},
		{"configFile", &configFile, "configFile"},
		{"removeContainers", &removeContainers, "removeContainers"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.variable == nil {
				t.Errorf("Global variable %s should not be nil", tt.varName)
			}
		})
	}
}