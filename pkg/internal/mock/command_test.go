package mock

import (
	"fmt"
	"os/exec"
	"strings"
	"testing"
)

func TestNewCommandState(t *testing.T) {
	state := NewCommandState()
	
	if state == nil {
		t.Fatal("NewCommandState() returned nil")
	}
	
	if state.ContainerStates == nil {
		t.Error("ContainerStates should be initialized")
	}
	
	if state.mockOutput == nil {
		t.Error("mockOutput should be initialized")
	}
	
	if state.CalledCommands == nil {
		t.Error("CalledCommands should be initialized")
	}
	
	if len(state.ContainerStates) != 0 {
		t.Errorf("Expected empty ContainerStates, got %d items", len(state.ContainerStates))
	}
	
	if len(state.mockOutput) != 0 {
		t.Errorf("Expected empty mockOutput, got %d items", len(state.mockOutput))
	}
	
	if len(state.CalledCommands) != 0 {
		t.Errorf("Expected empty CalledCommands, got %d items", len(state.CalledCommands))
	}
}

func TestCommandState_SetDebug(t *testing.T) {
	state := NewCommandState()
	
	// Test enabling debug
	state.SetDebug(true)
	if !state.debug {
		t.Error("Expected debug to be true after SetDebug(true)")
	}
	
	// Test disabling debug
	state.SetDebug(false)
	if state.debug {
		t.Error("Expected debug to be false after SetDebug(false)")
	}
}

func TestCommandState_SetContainerState(t *testing.T) {
	state := NewCommandState()
	
	tests := []struct {
		name          string
		containerName string
		containerState string
		expectedState string
	}{
		{"set running state", "test-container", "running", "RUNNING"},
		{"set stopped state", "test-container", "stopped", "STOPPED"},
		{"set lowercase state", "test-container", "paused", "PAUSED"},
		{"set mixed case state", "test-container", "StOpPeD", "STOPPED"},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := state.SetContainerState(tt.containerName, tt.containerState)
			if err != nil {
				t.Errorf("SetContainerState() returned error: %v", err)
			}
			
			actualState := state.GetContainerState(tt.containerName)
			if actualState != tt.expectedState {
				t.Errorf("Expected state '%s', got '%s'", tt.expectedState, actualState)
			}
		})
	}
}

func TestCommandState_GetContainerState(t *testing.T) {
	state := NewCommandState()
	
	// Test getting state for non-existent container
	containerState := state.GetContainerState("non-existent")
	if containerState != "" {
		t.Errorf("Expected empty state for non-existent container, got '%s'", containerState)
	}
	
	// Test getting state for existing container
	err := state.SetContainerState("test-container", "running")
	if err != nil {
		t.Fatalf("Failed to set container state: %v", err)
	}
	
	containerState = state.GetContainerState("test-container")
	if containerState != "RUNNING" {
		t.Errorf("Expected state 'RUNNING', got '%s'", containerState)
	}
}

func TestCommandState_getContainerState(t *testing.T) {
	state := NewCommandState()
	
	// Test getting state for non-existent container
	_, err := state.getContainerState("non-existent")
	if err == nil {
		t.Error("Expected error for non-existent container")
	}
	
	if !strings.Contains(err.Error(), "container state not found") {
		t.Errorf("Expected error to contain 'container state not found', got: %v", err)
	}
	
	// Test getting state for existing container
	state.SetContainerState("test-container", "running")
	
	containerState, err := state.getContainerState("test-container")
	if err != nil {
		t.Errorf("Unexpected error for existing container: %v", err)
	}
	
	if containerState != "RUNNING" {
		t.Errorf("Expected state 'RUNNING', got '%s'", containerState)
	}
}

func TestCommandState_ContainerExists(t *testing.T) {
	state := NewCommandState()
	
	// Test non-existent container
	if state.ContainerExists("non-existent") {
		t.Error("Expected ContainerExists to return false for non-existent container")
	}
	
	// Test existing container
	state.SetContainerState("test-container", "running")
	
	if !state.ContainerExists("test-container") {
		t.Error("Expected ContainerExists to return true for existing container")
	}
}

func TestCommandState_AddContainer(t *testing.T) {
	state := NewCommandState()
	
	tests := []struct {
		name          string
		containerName string
		containerState string
		expectedState string
	}{
		{"add running container", "container1", "running", "RUNNING"},
		{"add stopped container", "container2", "stopped", "STOPPED"},
		{"add paused container", "container3", "paused", "PAUSED"},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := state.AddContainer(tt.containerName, tt.containerState)
			if err != nil {
				t.Errorf("AddContainer() returned error: %v", err)
			}
			
			if !state.ContainerExists(tt.containerName) {
				t.Errorf("Container '%s' should exist after AddContainer", tt.containerName)
			}
			
			actualState := state.GetContainerState(tt.containerName)
			if actualState != tt.expectedState {
				t.Errorf("Expected state '%s', got '%s'", tt.expectedState, actualState)
			}
		})
	}
}

func TestCommandState_RemoveContainer(t *testing.T) {
	state := NewCommandState()
	
	// Add a container first
	state.AddContainer("test-container", "running")
	
	if !state.ContainerExists("test-container") {
		t.Fatal("Container should exist before removal")
	}
	
	// Remove the container
	err := state.RemoveContainer("test-container")
	if err != nil {
		t.Errorf("RemoveContainer() returned error: %v", err)
	}
	
	if state.ContainerExists("test-container") {
		t.Error("Container should not exist after removal")
	}
	
	// Test removing non-existent container (should not error)
	err = state.RemoveContainer("non-existent")
	if err != nil {
		t.Errorf("RemoveContainer() should not error for non-existent container: %v", err)
	}
}

func TestCommandState_AddMockOutput(t *testing.T) {
	state := NewCommandState()
	
	command := "test-command"
	output := []byte("test output")
	
	state.AddMockOutput(command, output)
	
	// Check that the command was added to CalledCommands
	if called, exists := state.CalledCommands[command]; !exists || called {
		t.Error("Command should be added to CalledCommands but not marked as called")
	}
	
	// Check that the output was stored
	if storedOutput, exists := state.mockOutput[command]; !exists {
		t.Error("Mock output should be stored")
	} else if string(storedOutput) != string(output) {
		t.Errorf("Expected output '%s', got '%s'", string(output), string(storedOutput))
	}
}

func TestCommandState_WasCalled(t *testing.T) {
	state := NewCommandState()
	
	command := "test-command"
	
	// Test command that was never added
	if state.WasCalled("non-existent") {
		t.Error("WasCalled should return false for non-existent command")
	}
	
	// Add command but don't call it
	state.AddMockOutput(command, []byte("output"))
	if state.WasCalled(command) {
		t.Error("WasCalled should return false for command that wasn't called")
	}
	
	// Call the command
	state.Command(command)
	if !state.WasCalled(command) {
		t.Error("WasCalled should return true for command that was called")
	}
}

func TestCommandState_Command(t *testing.T) {
	state := NewCommandState()
	
	// Test command without mock output
	output, err := state.Command("test-command")
	if err != nil {
		t.Errorf("Command() returned error: %v", err)
	}
	if len(output) != 0 {
		t.Errorf("Expected empty output for command without mock, got: %s", string(output))
	}
	
	// Test command with mock output
	expectedOutput := []byte("mock output")
	state.AddMockOutput("mock-command", expectedOutput)
	
	output, err = state.Command("mock-command")
	if err != nil {
		t.Errorf("Command() returned error: %v", err)
	}
	if string(output) != string(expectedOutput) {
		t.Errorf("Expected output '%s', got '%s'", string(expectedOutput), string(output))
	}
	
	// Test command with arguments
	commandWithArgs := "test-command arg1 arg2"
	state.AddMockOutput(commandWithArgs, []byte("args output"))
	
	output, err = state.Command("test-command", "arg1", "arg2")
	if err != nil {
		t.Errorf("Command() with args returned error: %v", err)
	}
	if string(output) != "args output" {
		t.Errorf("Expected output 'args output', got '%s'", string(output))
	}
}

func TestCommandState_Run(t *testing.T) {
	state := NewCommandState()
	
	tests := []struct {
		name        string
		command     string
		args        []string
		setup       func()
		expectError bool
		errorMsg    string
	}{
		{
			name:        "lxc-info for existing container",
			command:     "lxc-info",
			args:        []string{"-n", "test-container"},
			setup:       func() { state.AddContainer("test-container", "running") },
			expectError: false,
		},
		{
			name:        "lxc-info for non-existent container",
			command:     "lxc-info",
			args:        []string{"-n", "non-existent"},
			setup:       func() {},
			expectError: true,
			errorMsg:    "container does not exist",
		},
		{
			name:        "lxc-info with invalid args",
			command:     "lxc-info",
			args:        []string{"invalid"},
			setup:       func() {},
			expectError: true,
			errorMsg:    "invalid arguments",
		},
		{
			name:        "lxc-start for existing container",
			command:     "lxc-start",
			args:        []string{"-n", "test-container"},
			setup:       func() { state.AddContainer("test-container", "stopped") },
			expectError: false,
		},
		{
			name:        "lxc-start for non-existent container",
			command:     "lxc-start",
			args:        []string{"-n", "non-existent"},
			setup:       func() {},
			expectError: true,
			errorMsg:    "container does not exist",
		},
		{
			name:        "lxc-stop for existing container",
			command:     "lxc-stop",
			args:        []string{"-n", "test-container"},
			setup:       func() { state.AddContainer("test-container", "running") },
			expectError: false,
		},
		{
			name:        "lxc-stop for non-existent container",
			command:     "lxc-stop",
			args:        []string{"-n", "non-existent"},
			setup:       func() {},
			expectError: true,
			errorMsg:    "container does not exist",
		},
		{
			name:        "lxc-destroy for existing container",
			command:     "lxc-destroy",
			args:        []string{"-n", "test-container"},
			setup:       func() { state.AddContainer("test-container", "stopped") },
			expectError: false,
		},
		{
			name:        "unknown command",
			command:     "unknown-command",
			args:        []string{},
			setup:       func() {},
			expectError: false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset state for each test
			state = NewCommandState()
			tt.setup()
			
			err := state.Run(tt.command, tt.args...)
			
			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error for command '%s %v'", tt.command, tt.args)
				} else if tt.errorMsg != "" && !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error to contain '%s', got: %v", tt.errorMsg, err)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error for command '%s %v': %v", tt.command, tt.args, err)
				}
			}
		})
	}
}

func TestCommandState_Run_StateTransitions(t *testing.T) {
	state := NewCommandState()
	
	// Add a container
	state.AddContainer("test-container", "stopped")
	
	// Test starting container
	err := state.Run("lxc-start", "-n", "test-container")
	if err != nil {
		t.Errorf("Failed to start container: %v", err)
	}
	
	if state.GetContainerState("test-container") != "RUNNING" {
		t.Errorf("Expected container state to be RUNNING after start, got: %s", state.GetContainerState("test-container"))
	}
	
	// Test stopping container
	err = state.Run("lxc-stop", "-n", "test-container")
	if err != nil {
		t.Errorf("Failed to stop container: %v", err)
	}
	
	if state.GetContainerState("test-container") != "STOPPED" {
		t.Errorf("Expected container state to be STOPPED after stop, got: %s", state.GetContainerState("test-container"))
	}
	
	// Test destroying container
	err = state.Run("lxc-destroy", "-n", "test-container")
	if err != nil {
		t.Errorf("Failed to destroy container: %v", err)
	}
	
	if state.ContainerExists("test-container") {
		t.Error("Container should not exist after destroy")
	}
}

func TestCommandState_Output(t *testing.T) {
	state := NewCommandState()
	
	// Test lxc-info for existing container
	state.AddContainer("test-container", "running")
	
	output, err := state.Output("lxc-info", "-n", "test-container")
	if err != nil {
		t.Errorf("Output() returned error: %v", err)
	}
	
	expectedOutput := "Name: test-container\nState: RUNNING\n"
	if string(output) != expectedOutput {
		t.Errorf("Expected output '%s', got '%s'", expectedOutput, string(output))
	}
	
	// Test lxc-info for non-existent container
	output, err = state.Output("lxc-info", "-n", "non-existent")
	if err == nil {
		t.Error("Expected error for non-existent container")
	}
	
	expectedErrorOutput := "container does not exist\n"
	if string(output) != expectedErrorOutput {
		t.Errorf("Expected error output '%s', got '%s'", expectedErrorOutput, string(output))
	}
	
	// Test lxc-info with invalid arguments
	output, err = state.Output("lxc-info", "invalid")
	if err == nil {
		t.Error("Expected error for invalid arguments")
	}
	
	if !strings.Contains(err.Error(), "invalid arguments") {
		t.Errorf("Expected error to contain 'invalid arguments', got: %v", err)
	}
	
	// Test unknown command
	output, err = state.Output("unknown-command")
	if err != nil {
		t.Errorf("Unknown command should not return error: %v", err)
	}
	
	if len(output) != 0 {
		t.Errorf("Expected empty output for unknown command, got: %s", string(output))
	}
}

func TestCommandState_CombinedOutput(t *testing.T) {
	state := NewCommandState()
	
	// CombinedOutput should behave the same as Output
	state.AddContainer("test-container", "running")
	
	output, err := state.CombinedOutput("lxc-info", "-n", "test-container")
	if err != nil {
		t.Errorf("CombinedOutput() returned error: %v", err)
	}
	
	expectedOutput := "Name: test-container\nState: RUNNING\n"
	if string(output) != expectedOutput {
		t.Errorf("Expected output '%s', got '%s'", expectedOutput, string(output))
	}
}

func TestCommandState_execLXCCommand(t *testing.T) {
	state := NewCommandState()
	
	// Test lxc-ls command
	state.AddContainer("container1", "running")
	state.AddContainer("container2", "stopped")
	
	output, err := state.execLXCCommand("lxc-ls")
	if err != nil {
		t.Errorf("execLXCCommand() returned error: %v", err)
	}
	
	outputStr := string(output)
	if !strings.Contains(outputStr, "container1") || !strings.Contains(outputStr, "container2") {
		t.Errorf("Expected output to contain both containers, got: %s", outputStr)
	}
	
	// Test unknown command
	output, err = state.execLXCCommand("unknown-command")
	if err != nil {
		t.Errorf("Unknown command should not return error: %v", err)
	}
	
	if output != nil {
		t.Errorf("Expected nil output for unknown command, got: %s", string(output))
	}
}

func TestCommandState_DebugMode(t *testing.T) {
	state := NewCommandState()
	state.SetDebug(true)
	
	// Test that debug mode doesn't break functionality
	state.AddContainer("test-container", "running")
	
	if !state.ContainerExists("test-container") {
		t.Error("Container should exist in debug mode")
	}
	
	err := state.Run("lxc-info", "-n", "test-container")
	if err != nil {
		t.Errorf("Command should succeed in debug mode: %v", err)
	}
	
	output, err := state.Output("lxc-info", "-n", "test-container")
	if err != nil {
		t.Errorf("Output should succeed in debug mode: %v", err)
	}
	
	if !strings.Contains(string(output), "test-container") {
		t.Errorf("Output should contain container name, got: %s", string(output))
	}
}

func TestCommandState_ConcurrentAccess(t *testing.T) {
	state := NewCommandState()
	
	// Test concurrent access to ensure thread safety
	done := make(chan bool, 10)
	
	// Start multiple goroutines that modify state
	for i := 0; i < 10; i++ {
		go func(id int) {
			containerName := fmt.Sprintf("container-%d", id)
			state.AddContainer(containerName, "running")
			state.SetContainerState(containerName, "stopped")
			state.ContainerExists(containerName)
			state.GetContainerState(containerName)
			state.RemoveContainer(containerName)
			done <- true
		}(i)
	}
	
	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}
	
	// Test should complete without race conditions or panics
}

func TestSetupMockCommand(t *testing.T) {
	// Create a variable to hold the exec command function
	var execCommand func(string, ...string) *exec.Cmd = exec.Command
	
	// Setup mock command
	mockState, cleanup := SetupMockCommand(&execCommand)
	defer cleanup()
	
	// Verify mock state was returned
	if mockState == nil {
		t.Fatal("SetupMockCommand should return a mock state")
	}
	
	// Verify that execCommand was modified
	cmd := execCommand("test-command", "arg1", "arg2")
	if cmd == nil {
		t.Fatal("Mock execCommand should return a command")
	}
	
	// The command should be modified to use /bin/true or /bin/false
	if cmd.Path != "/bin/true" && cmd.Path != "/bin/false" {
		t.Errorf("Expected mock command to use /bin/true or /bin/false, got: %s", cmd.Path)
	}
	
	// Test cleanup function
	cleanup()
	
	// After cleanup, execCommand should be restored
	cmd2 := execCommand("echo", "test")
	if cmd2.Path == "/bin/true" || cmd2.Path == "/bin/false" {
		t.Error("execCommand should be restored after cleanup")
	}
}

func TestSetupMockCommand_WithContainerOperations(t *testing.T) {
	var execCommand func(string, ...string) *exec.Cmd = exec.Command
	
	mockState, cleanup := SetupMockCommand(&execCommand)
	defer cleanup()
	
	// Add a container to mock state
	mockState.AddContainer("test-container", "stopped")
	
	// Test that mock command handles container operations
	cmd := execCommand("lxc-info", "-n", "test-container")
	if cmd == nil {
		t.Fatal("Mock execCommand should return a command")
	}
	
	// The command should succeed for existing container
	if cmd.Path != "/bin/true" {
		t.Errorf("Expected successful command to use /bin/true, got: %s", cmd.Path)
	}
	
	// Test with non-existent container
	cmd = execCommand("lxc-info", "-n", "non-existent")
	if cmd == nil {
		t.Fatal("Mock execCommand should return a command")
	}
	
	// The command should fail for non-existent container
	if cmd.Path != "/bin/false" {
		t.Errorf("Expected failing command to use /bin/false, got: %s", cmd.Path)
	}
}

func TestCommandInterface(t *testing.T) {
	// Test that CommandState implements the Command interface
	var _ Command = (*CommandState)(nil)
	
	state := NewCommandState()
	
	// Test all interface methods
	state.SetDebug(true)
	state.AddContainer("test", "running")
	
	if !state.ContainerExists("test") {
		t.Error("ContainerExists should work through interface")
	}
	
	err := state.SetContainerState("test", "stopped")
	if err != nil {
		t.Errorf("SetContainerState should work through interface: %v", err)
	}
	
	err = state.RemoveContainer("test")
	if err != nil {
		t.Errorf("RemoveContainer should work through interface: %v", err)
	}
	
	state.AddMockOutput("test-cmd", []byte("output"))
	
	if !state.WasCalled("test-cmd") {
		// WasCalled should return false until command is actually called
	}
	
	output, err := state.Command("test-cmd")
	if err != nil {
		t.Errorf("Command should work through interface: %v", err)
	}
	
	if string(output) != "output" {
		t.Errorf("Expected output 'output', got '%s'", string(output))
	}
	
	err = state.Run("lxc-ls")
	if err != nil {
		t.Errorf("Run should work through interface: %v", err)
	}
	
	output, err = state.Output("lxc-ls")
	if err != nil {
		t.Errorf("Output should work through interface: %v", err)
	}
	
	output, err = state.CombinedOutput("lxc-ls")
	if err != nil {
		t.Errorf("CombinedOutput should work through interface: %v", err)
	}
}