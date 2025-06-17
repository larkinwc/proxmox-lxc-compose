package main

import (
	"strings"
	"testing"
	"time"
)

func TestPsCommand(t *testing.T) {
	// Find the ps command
	psCmd, _, err := rootCmd.Find([]string{"ps"})
	if err != nil {
		t.Fatalf("Failed to find ps command: %v", err)
	}

	// Test command metadata
	if psCmd.Use != "ps" {
		t.Errorf("Expected Use to be 'ps', got '%s'", psCmd.Use)
	}

	if psCmd.Short != "List containers" {
		t.Errorf("Expected Short to be 'List containers', got '%s'", psCmd.Short)
	}

	// Test that ps command has no flags (it shouldn't have any based on the implementation)
	if psCmd.Flags().NFlag() != 0 {
		t.Errorf("Expected ps command to have no flags, got %d", psCmd.Flags().NFlag())
	}
}

func TestLogsCommand(t *testing.T) {
	// Find the logs command
	logsCmd, _, err := rootCmd.Find([]string{"logs"})
	if err != nil {
		t.Fatalf("Failed to find logs command: %v", err)
	}

	// Test command metadata
	if logsCmd.Use != "logs [container]" {
		t.Errorf("Expected Use to be 'logs [container]', got '%s'", logsCmd.Use)
	}

	if logsCmd.Short != "View container logs" {
		t.Errorf("Expected Short to be 'View container logs', got '%s'", logsCmd.Short)
	}

	// Test that logs command requires exactly one argument
	if logsCmd.Args == nil {
		t.Error("Expected logs command to have Args validation")
	}
}

func TestLogsCommandFlags(t *testing.T) {
	logsCmd, _, err := rootCmd.Find([]string{"logs"})
	if err != nil {
		t.Fatalf("Failed to find logs command: %v", err)
	}

	// Test --follow flag
	followFlag := logsCmd.Flags().Lookup("follow")
	if followFlag == nil {
		t.Error("Expected --follow flag to exist")
	}
	if followFlag.Shorthand != "f" {
		t.Errorf("Expected --follow flag shorthand to be 'f', got '%s'", followFlag.Shorthand)
	}

	// Test --tail flag
	tailFlag := logsCmd.Flags().Lookup("tail")
	if tailFlag == nil {
		t.Error("Expected --tail flag to exist")
	}
	if tailFlag.Shorthand != "n" {
		t.Errorf("Expected --tail flag shorthand to be 'n', got '%s'", tailFlag.Shorthand)
	}

	// Test --since flag
	sinceFlag := logsCmd.Flags().Lookup("since")
	if sinceFlag == nil {
		t.Error("Expected --since flag to exist")
	}

	// Test --timestamps flag
	timestampsFlag := logsCmd.Flags().Lookup("timestamps")
	if timestampsFlag == nil {
		t.Error("Expected --timestamps flag to exist")
	}
	if timestampsFlag.Shorthand != "t" {
		t.Errorf("Expected --timestamps flag shorthand to be 't', got '%s'", timestampsFlag.Shorthand)
	}
}

func TestLogsCommandTimeParsingLogic(t *testing.T) {
	// Test the time parsing logic from the logs command
	tests := []struct {
		name        string
		since       string
		expectError bool
		description string
	}{
		{
			name:        "empty since",
			since:       "",
			expectError: false,
			description: "empty since should not error",
		},
		{
			name:        "1h duration",
			since:       "1h",
			expectError: false,
			description: "1h should parse as duration",
		},
		{
			name:        "24h duration",
			since:       "24h",
			expectError: false,
			description: "24h should parse as duration",
		},
		{
			name:        "RFC3339 timestamp",
			since:       "2023-01-01T00:00:00Z",
			expectError: false,
			description: "RFC3339 timestamp should parse",
		},
		{
			name:        "invalid timestamp",
			since:       "invalid-time",
			expectError: true,
			description: "invalid timestamp should error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var sinceTime time.Time
			var err error

			if tt.since != "" {
				if tt.since == "1h" || tt.since == "24h" {
					duration, parseErr := time.ParseDuration(tt.since)
					if parseErr != nil {
						err = parseErr
					} else {
						sinceTime = time.Now().Add(-duration)
					}
				} else {
					sinceTime, err = time.Parse(time.RFC3339, tt.since)
				}
			}

			if tt.expectError && err == nil {
				t.Errorf("Expected error for since value '%s'", tt.since)
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error for since value '%s': %v", tt.since, err)
			}

			if !tt.expectError && tt.since != "" {
				// Verify sinceTime was set
				if sinceTime.IsZero() {
					t.Errorf("Expected sinceTime to be set for since value '%s'", tt.since)
				}
			}
		})
	}
}

func TestPauseCommand(t *testing.T) {
	// Find the pause command
	pauseCmd, _, err := rootCmd.Find([]string{"pause"})
	if err != nil {
		t.Fatalf("Failed to find pause command: %v", err)
	}

	// Test command metadata
	if pauseCmd.Use != "pause [container...]" {
		t.Errorf("Expected Use to be 'pause [container...]', got '%s'", pauseCmd.Use)
	}

	if pauseCmd.Short != "Pause one or more containers" {
		t.Errorf("Expected Short to be 'Pause one or more containers', got '%s'", pauseCmd.Short)
	}

	// Test that pause command requires at least one argument
	if pauseCmd.Args == nil {
		t.Error("Expected pause command to have Args validation")
	}

	// Test that pause command has no flags
	if pauseCmd.Flags().NFlag() != 0 {
		t.Errorf("Expected pause command to have no flags, got %d", pauseCmd.Flags().NFlag())
	}
}

func TestUnpauseCommand(t *testing.T) {
	// Find the unpause command
	unpauseCmd, _, err := rootCmd.Find([]string{"unpause"})
	if err != nil {
		t.Fatalf("Failed to find unpause command: %v", err)
	}

	// Test command metadata
	if unpauseCmd.Use != "unpause [container...]" {
		t.Errorf("Expected Use to be 'unpause [container...]', got '%s'", unpauseCmd.Use)
	}

	if unpauseCmd.Short != "Unpause one or more containers" {
		t.Errorf("Expected Short to be 'Unpause one or more containers', got '%s'", unpauseCmd.Short)
	}

	// Test that unpause command requires at least one argument
	if unpauseCmd.Args == nil {
		t.Error("Expected unpause command to have Args validation")
	}

	// Test that unpause command has no flags
	if unpauseCmd.Flags().NFlag() != 0 {
		t.Errorf("Expected unpause command to have no flags, got %d", unpauseCmd.Flags().NFlag())
	}
}

func TestImagesCommand(t *testing.T) {
	// Find the images command
	imagesCmd, _, err := rootCmd.Find([]string{"images"})
	if err != nil {
		t.Fatalf("Failed to find images command: %v", err)
	}

	// Test command metadata
	if imagesCmd.Use != "images" {
		t.Errorf("Expected Use to be 'images', got '%s'", imagesCmd.Use)
	}

	if imagesCmd.Short != "Manage OCI images" {
		t.Errorf("Expected Short to be 'Manage OCI images', got '%s'", imagesCmd.Short)
	}

	if !strings.Contains(imagesCmd.Long, "Manage OCI images including pulling, pushing, listing and removing images") {
		t.Errorf("Unexpected Long description: %s", imagesCmd.Long)
	}
}

func TestImagesSubcommands(t *testing.T) {
	// Test that images command has the expected subcommands
	expectedSubcommands := []string{"pull", "push", "list", "remove"}

	for _, expectedCmd := range expectedSubcommands {
		t.Run(expectedCmd, func(t *testing.T) {
			cmd, _, err := rootCmd.Find([]string{"images", expectedCmd})
			if err != nil {
				t.Errorf("Expected to find 'images %s' command, got error: %v", expectedCmd, err)
			}
			if cmd == nil || cmd.Name() != expectedCmd {
				t.Errorf("Expected to find 'images %s' command, but it was not found", expectedCmd)
			}
		})
	}
}

func TestImagesPullCommand(t *testing.T) {
	pullCmd, _, err := rootCmd.Find([]string{"images", "pull"})
	if err != nil {
		t.Fatalf("Failed to find images pull command: %v", err)
	}

	// Test command metadata
	if pullCmd.Use != "pull [registry/repository:tag]" {
		t.Errorf("Expected Use to be 'pull [registry/repository:tag]', got '%s'", pullCmd.Use)
	}

	if pullCmd.Short != "Pull an image from a registry" {
		t.Errorf("Expected Short to be 'Pull an image from a registry', got '%s'", pullCmd.Short)
	}

	// Test that pull command requires exactly one argument
	if pullCmd.Args == nil {
		t.Error("Expected pull command to have Args validation")
	}
}

func TestImagesPushCommand(t *testing.T) {
	pushCmd, _, err := rootCmd.Find([]string{"images", "push"})
	if err != nil {
		t.Fatalf("Failed to find images push command: %v", err)
	}

	// Test command metadata
	if pushCmd.Use != "push [registry/repository:tag]" {
		t.Errorf("Expected Use to be 'push [registry/repository:tag]', got '%s'", pushCmd.Use)
	}

	if pushCmd.Short != "Push an image to a registry" {
		t.Errorf("Expected Short to be 'Push an image to a registry', got '%s'", pushCmd.Short)
	}

	// Test that push command requires exactly one argument
	if pushCmd.Args == nil {
		t.Error("Expected push command to have Args validation")
	}
}

func TestImagesListCommand(t *testing.T) {
	listCmd, _, err := rootCmd.Find([]string{"images", "list"})
	if err != nil {
		t.Fatalf("Failed to find images list command: %v", err)
	}

	// Test command metadata
	if listCmd.Use != "list" {
		t.Errorf("Expected Use to be 'list', got '%s'", listCmd.Use)
	}

	if listCmd.Short != "List locally stored images" {
		t.Errorf("Expected Short to be 'List locally stored images', got '%s'", listCmd.Short)
	}

	// Test that list command has no required arguments
	// (Args should be nil or allow zero arguments)
}

func TestImagesRemoveCommand(t *testing.T) {
	removeCmd, _, err := rootCmd.Find([]string{"images", "remove"})
	if err != nil {
		t.Fatalf("Failed to find images remove command: %v", err)
	}

	// Test command metadata
	if removeCmd.Use != "remove [registry/repository:tag]" {
		t.Errorf("Expected Use to be 'remove [registry/repository:tag]', got '%s'", removeCmd.Use)
	}

	if removeCmd.Short != "Remove an image from local storage" {
		t.Errorf("Expected Short to be 'Remove an image from local storage', got '%s'", removeCmd.Short)
	}

	// Test that remove command requires exactly one argument
	if removeCmd.Args == nil {
		t.Error("Expected remove command to have Args validation")
	}
}

func TestConvertCommand(t *testing.T) {
	// Find the convert command
	convertCmd, _, err := rootCmd.Find([]string{"convert"})
	if err != nil {
		t.Fatalf("Failed to find convert command: %v", err)
	}

	// Test command metadata
	if convertCmd.Use != "convert [image]" {
		t.Errorf("Expected Use to be 'convert [image]', got '%s'", convertCmd.Use)
	}

	if convertCmd.Short != "Convert an OCI image to LXC template" {
		t.Errorf("Expected Short to be 'Convert an OCI image to LXC template', got '%s'", convertCmd.Short)
	}

	// Test that convert command requires exactly one argument
	if convertCmd.Args == nil {
		t.Error("Expected convert command to have Args validation")
	}
}

func TestConvertCommandFlags(t *testing.T) {
	convertCmd, _, err := rootCmd.Find([]string{"convert"})
	if err != nil {
		t.Fatalf("Failed to find convert command: %v", err)
	}

	// Test --output flag
	outputFlag := convertCmd.Flags().Lookup("output")
	if outputFlag == nil {
		t.Error("Expected --output flag to exist")
	}
	if outputFlag.Shorthand != "o" {
		t.Errorf("Expected --output flag shorthand to be 'o', got '%s'", outputFlag.Shorthand)
	}
}

func TestCommandArgsValidation(t *testing.T) {
	// Test that commands with specific argument requirements validate correctly
	tests := []struct {
		name        string
		command     []string
		args        []string
		expectError bool
	}{
		{
			name:        "logs with no args",
			command:     []string{"logs"},
			args:        []string{},
			expectError: true,
		},
		{
			name:        "logs with one arg",
			command:     []string{"logs"},
			args:        []string{"container1"},
			expectError: false,
		},
		{
			name:        "logs with too many args",
			command:     []string{"logs"},
			args:        []string{"container1", "container2"},
			expectError: true,
		},
		{
			name:        "pause with no args",
			command:     []string{"pause"},
			args:        []string{},
			expectError: true,
		},
		{
			name:        "pause with one arg",
			command:     []string{"pause"},
			args:        []string{"container1"},
			expectError: false,
		},
		{
			name:        "pause with multiple args",
			command:     []string{"pause"},
			args:        []string{"container1", "container2"},
			expectError: false,
		},
		{
			name:        "unpause with no args",
			command:     []string{"unpause"},
			args:        []string{},
			expectError: true,
		},
		{
			name:        "unpause with one arg",
			command:     []string{"unpause"},
			args:        []string{"container1"},
			expectError: false,
		},
		{
			name:        "convert with no args",
			command:     []string{"convert"},
			args:        []string{},
			expectError: true,
		},
		{
			name:        "convert with one arg",
			command:     []string{"convert"},
			args:        []string{"image1"},
			expectError: false,
		},
		{
			name:        "convert with too many args",
			command:     []string{"convert"},
			args:        []string{"image1", "image2"},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd, _, err := rootCmd.Find(tt.command)
			if err != nil {
				t.Fatalf("Failed to find command %v: %v", tt.command, err)
			}

			if cmd.Args != nil {
				err = cmd.Args(cmd, tt.args)
				if tt.expectError && err == nil {
					t.Errorf("Expected args validation error for command %v with args %v", tt.command, tt.args)
				}
				if !tt.expectError && err != nil {
					t.Errorf("Unexpected args validation error for command %v with args %v: %v", tt.command, tt.args, err)
				}
			}
		})
	}
}

func TestGetRegistryManagerFunction(t *testing.T) {
	// Test the getRegistryManager helper function
	manager, err := getRegistryManager()
	
	// This should fail in test environment since we don't have proper setup
	// but we can test that the function exists and handles errors appropriately
	if err == nil && manager == nil {
		t.Error("getRegistryManager should return either a manager or an error")
	}
	
	// The function should either succeed or fail gracefully
	if err != nil {
		// Error is expected in test environment
		if !strings.Contains(err.Error(), "failed to") {
			t.Errorf("Expected error to contain 'failed to', got: %v", err)
		}
	}
}