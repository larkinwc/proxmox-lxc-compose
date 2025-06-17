package testutil

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestMockCommandState_CommandWasCalled(t *testing.T) {
	mock := &MockCommandState{
		commandHistory: []struct {
			name string
			args []string
		}{
			{"lxc-start", []string{"-n", "container1"}},
			{"lxc-stop", []string{"-n", "container2"}},
			{"lxc-info", []string{"-n", "container1"}},
		},
	}
	
	tests := []struct {
		name     string
		command  string
		args     []string
		expected bool
	}{
		{"exact match", "lxc-start", []string{"-n", "container1"}, true},
		{"different container", "lxc-start", []string{"-n", "container2"}, false},
		{"different command", "lxc-destroy", []string{"-n", "container1"}, false},
		{"different args", "lxc-start", []string{"-f", "container1"}, false},
		{"partial args", "lxc-start", []string{"-n"}, false},
		{"extra args", "lxc-start", []string{"-n", "container1", "-d"}, false},
		{"no args", "lxc-ls", []string{}, false},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := mock.CommandWasCalled(tt.command, tt.args...)
			if result != tt.expected {
				t.Errorf("CommandWasCalled(%s, %v) = %v, want %v", tt.command, tt.args, result, tt.expected)
			}
		})
	}
}

func TestMockCommandState_SetContainerState(t *testing.T) {
	// Set up temporary config path
	tempDir := t.TempDir()
	os.Setenv("CONTAINER_CONFIG_PATH", tempDir)
	defer os.Unsetenv("CONTAINER_CONFIG_PATH")
	
	mock := &MockCommandState{
		ContainerStates: make(map[string]string),
		debug:           false, // Disable debug to reduce test output
	}
	
	tests := []struct {
		name          string
		containerName string
		state         string
		expectError   bool
	}{
		{"valid container and state", "test-container", "running", false},
		{"uppercase state", "test-container", "STOPPED", false},
		{"lowercase state", "test-container", "frozen", false},
		{"nonexistent container", "nonexistent", "running", true},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := mock.SetContainerState(tt.containerName, tt.state)
			
			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error for container %s", tt.containerName)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				
				// Verify state was set in memory
				actualState, exists := mock.ContainerStates[tt.containerName]
				if !exists {
					t.Errorf("Container state not found in memory")
				}
				
				expectedState := strings.ToUpper(tt.state)
				if actualState != expectedState {
					t.Errorf("Expected state %s, got %s", expectedState, actualState)
				}
				
				// Verify state file was created
				stateFile := filepath.Join(tempDir, "state", tt.containerName+".json")
				if _, err := os.Stat(stateFile); os.IsNotExist(err) {
					t.Errorf("State file should exist at %s", stateFile)
				}
			}
		})
	}
}

func TestMockCommandState_AddContainer(t *testing.T) {
	// Set up temporary config path
	tempDir := t.TempDir()
	os.Setenv("CONTAINER_CONFIG_PATH", tempDir)
	defer os.Unsetenv("CONTAINER_CONFIG_PATH")
	
	mock := &MockCommandState{
		ContainerStates: make(map[string]string),
		debug:           false,
	}
	
	tests := []struct {
		name          string
		containerName string
		state         string
	}{
		{"running container", "container1", "running"},
		{"stopped container", "container2", "stopped"},
		{"frozen container", "container3", "frozen"},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock.AddContainer(tt.containerName, tt.state)
			
			// Verify state was set in memory
			actualState, exists := mock.ContainerStates[tt.containerName]
			if !exists {
				t.Errorf("Container state not found in memory")
			}
			
			expectedState := strings.ToUpper(tt.state)
			if actualState != expectedState {
				t.Errorf("Expected state %s, got %s", expectedState, actualState)
			}
			
			// Verify container directory was created
			containerDir := filepath.Join(tempDir, tt.containerName)
			if _, err := os.Stat(containerDir); os.IsNotExist(err) {
				t.Errorf("Container directory should exist at %s", containerDir)
			}
			
			// Verify state file was created
			stateFile := filepath.Join(tempDir, "state", tt.containerName+".json")
			if _, err := os.Stat(stateFile); os.IsNotExist(err) {
				t.Errorf("State file should exist at %s", stateFile)
			}
		})
	}
}

func TestMockCommandState_SetDebug(t *testing.T) {
	mock := &MockCommandState{}
	
	// Test enabling debug
	mock.SetDebug(true)
	if !mock.debug {
		t.Error("Expected debug to be true after SetDebug(true)")
	}
	
	// Test disabling debug
	mock.SetDebug(false)
	if mock.debug {
		t.Error("Expected debug to be false after SetDebug(false)")
	}
}

func TestMockCommandState_GetContainerState(t *testing.T) {
	mock := &MockCommandState{
		ContainerStates: map[string]string{
			"container1": "RUNNING",
			"container2": "STOPPED",
		},
	}
	
	tests := []struct {
		name          string
		containerName string
		expectedState string
		expectedExists bool
	}{
		{"existing running container", "container1", "RUNNING", true},
		{"existing stopped container", "container2", "STOPPED", true},
		{"non-existent container", "container3", "", false},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			state, exists := mock.GetContainerState(tt.containerName)
			
			if exists != tt.expectedExists {
				t.Errorf("Expected exists=%v, got %v", tt.expectedExists, exists)
			}
			
			if state != tt.expectedState {
				t.Errorf("Expected state=%s, got %s", tt.expectedState, state)
			}
		})
	}
}

func TestMockCommandState_RemoveContainer(t *testing.T) {
	mock := &MockCommandState{
		ContainerStates: map[string]string{
			"container1": "RUNNING",
			"container2": "STOPPED",
		},
	}
	
	// Remove existing container
	mock.RemoveContainer("container1")
	
	// Verify container was removed
	_, exists := mock.ContainerStates["container1"]
	if exists {
		t.Error("Container should be removed from state")
	}
	
	// Verify other container still exists
	_, exists = mock.ContainerStates["container2"]
	if !exists {
		t.Error("Other container should still exist")
	}
	
	// Remove non-existent container (should not panic)
	mock.RemoveContainer("nonexistent")
}

func TestMockCommandState_ContainerExists(t *testing.T) {
	mock := &MockCommandState{
		ContainerStates: map[string]string{
			"container1": "RUNNING",
		},
	}
	
	tests := []struct {
		name          string
		containerName string
		expected      bool
	}{
		{"existing container", "container1", true},
		{"non-existent container", "container2", false},
		{"nonexistent special case", "nonexistent", false},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := mock.ContainerExists(tt.containerName)
			if result != tt.expected {
				t.Errorf("ContainerExists(%s) = %v, want %v", tt.containerName, result, tt.expected)
			}
		})
	}
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
	
	// Use mockState to avoid unused variable error
	_ = mockState
	
	// Verify that execCommand was modified
	cmd := execCommand("lxc-info", "-n", "test-container")
	if cmd == nil {
		t.Fatal("Mock execCommand should return a command")
	}
	
	// Test cleanup function
	cleanup()
	
	// After cleanup, execCommand should be restored
	cmd2 := execCommand("echo", "test")
	if cmd2 == nil {
		t.Fatal("execCommand should be restored after cleanup")
	}
}

func TestSetupMockCommand_WithContainerOperations(t *testing.T) {
	var execCommand func(string, ...string) *exec.Cmd = exec.Command
	
	mockState, cleanup := SetupMockCommand(&execCommand)
	defer cleanup()
	
	// Add a container to mock state
	mockState.AddContainer("test-container", "stopped")
	
	// Test that mock command handles container operations
	cmd := execCommand("lxc-start", "-n", "test-container")
	if cmd == nil {
		t.Fatal("Mock execCommand should return a command")
	}
	
	// Execute the command to test state transitions
	err := cmd.Run()
	if err != nil {
		t.Errorf("Command should succeed for valid state transition: %v", err)
	}
	
	// Verify state was updated
	state, exists := mockState.GetContainerState("test-container")
	if !exists {
		t.Error("Container should still exist after start")
	}
	if state != "RUNNING" {
		t.Errorf("Expected container state to be RUNNING, got %s", state)
	}
}

func TestSetupMockCommand_InvalidArguments(t *testing.T) {
	var execCommand func(string, ...string) *exec.Cmd = exec.Command
	
	mockState, cleanup := SetupMockCommand(&execCommand)
	defer cleanup()
	
	// Use mockState to avoid unused variable error
	_ = mockState
	
	// Test with invalid arguments
	cmd := execCommand("lxc-start", "invalid")
	if cmd == nil {
		t.Fatal("Mock execCommand should return a command even for invalid args")
	}
	
	// Command should fail
	err := cmd.Run()
	if err == nil {
		t.Error("Command should fail for invalid arguments")
	}
}

func TestSetupMockCommand_NonexistentContainer(t *testing.T) {
	var execCommand func(string, ...string) *exec.Cmd = exec.Command
	
	mockState, cleanup := SetupMockCommand(&execCommand)
	defer cleanup()
	
	// Use mockState to avoid unused variable error
	_ = mockState
	
	// Test with nonexistent container
	cmd := execCommand("lxc-start", "-n", "nonexistent")
	if cmd == nil {
		t.Fatal("Mock execCommand should return a command")
	}
	
	// Command should fail
	err := cmd.Run()
	if err == nil {
		t.Error("Command should fail for nonexistent container")
	}
}

func TestNewMockCommandExecutor(t *testing.T) {
	executor := NewMockCommandExecutor()
	
	if executor == nil {
		t.Fatal("NewMockCommandExecutor should not return nil")
	}
	
	if executor.commands == nil {
		t.Error("commands map should be initialized")
	}
	
	if executor.errorCmds == nil {
		t.Error("errorCmds map should be initialized")
	}
	
	if executor.actualExec {
		t.Error("actualExec should be false by default")
	}
}

func TestMockCommandExecutor_AddMockCommand(t *testing.T) {
	executor := NewMockCommandExecutor()
	
	cmd := "echo hello"
	output := []byte("hello\n")
	
	executor.AddMockCommand(cmd, output)
	
	// Verify command was added
	if storedOutput, exists := executor.commands[cmd]; !exists {
		t.Error("Command should be stored")
	} else if string(storedOutput) != string(output) {
		t.Errorf("Expected output %s, got %s", string(output), string(storedOutput))
	}
}

func TestMockCommandExecutor_AddMockError(t *testing.T) {
	executor := NewMockCommandExecutor()
	
	cmd := "failing-command"
	expectedErr := os.ErrNotExist
	
	executor.AddMockError(cmd, expectedErr)
	
	// Verify error was added
	if storedErr, exists := executor.errorCmds[cmd]; !exists {
		t.Error("Error should be stored")
	} else if storedErr != expectedErr {
		t.Errorf("Expected error %v, got %v", expectedErr, storedErr)
	}
}

func TestMockCommandExecutor_AddErrorCommand(t *testing.T) {
	executor := NewMockCommandExecutor()
	
	cmd := "failing-command"
	errMsg := "command failed"
	
	executor.AddErrorCommand(cmd, errMsg)
	
	// Verify error was added
	if storedErr, exists := executor.errorCmds[cmd]; !exists {
		t.Error("Error should be stored")
	} else if storedErr.Error() != errMsg {
		t.Errorf("Expected error message %s, got %s", errMsg, storedErr.Error())
	}
}

func TestMockCommandExecutor_SetActualExecution(t *testing.T) {
	executor := NewMockCommandExecutor()
	
	// Test enabling actual execution
	executor.SetActualExecution(true)
	if !executor.actualExec {
		t.Error("actualExec should be true after SetActualExecution(true)")
	}
	
	// Test disabling actual execution
	executor.SetActualExecution(false)
	if executor.actualExec {
		t.Error("actualExec should be false after SetActualExecution(false)")
	}
}

func TestMockCommandExecutor_Command(t *testing.T) {
	executor := NewMockCommandExecutor()
	
	// Test successful command
	successCmd := "echo hello"
	expectedOutput := []byte("hello world")
	executor.AddMockCommand(successCmd, expectedOutput)
	
	cmd := executor.Command("echo", "hello")
	if cmd == nil {
		t.Fatal("Command should not be nil")
	}
	
	output, err := cmd.Output()
	if err != nil {
		t.Errorf("Command should succeed: %v", err)
	}
	
	if string(output) != string(expectedOutput) {
		t.Errorf("Expected output %s, got %s", string(expectedOutput), string(output))
	}
}

func TestMockCommandExecutor_Command_Error(t *testing.T) {
	executor := NewMockCommandExecutor()
	
	// Test error command
	errorCmd := "failing-command"
	executor.AddErrorCommand(errorCmd, "command failed")
	
	cmd := executor.Command("failing-command")
	if cmd == nil {
		t.Fatal("Command should not be nil")
	}
	
	err := cmd.Run()
	if err == nil {
		t.Error("Command should fail")
	}
}

func TestMockCommandExecutor_Command_Default(t *testing.T) {
	executor := NewMockCommandExecutor()
	
	// Test unmocked command (should succeed with no output)
	cmd := executor.Command("unmocked-command")
	if cmd == nil {
		t.Fatal("Command should not be nil")
	}
	
	err := cmd.Run()
	if err != nil {
		t.Errorf("Unmocked command should succeed: %v", err)
	}
}

func TestMockCommandExecutor_Command_ActualExecution(t *testing.T) {
	executor := NewMockCommandExecutor()
	executor.SetActualExecution(true)
	
	// Test actual execution with a simple command
	cmd := executor.Command("echo", "test")
	if cmd == nil {
		t.Fatal("Command should not be nil")
	}
	
	output, err := cmd.Output()
	if err != nil {
		t.Errorf("Actual command should succeed: %v", err)
	}
	
	expectedOutput := "test\n"
	if string(output) != expectedOutput {
		t.Errorf("Expected output %s, got %s", expectedOutput, string(output))
	}
}

func TestMockCommandState_StateTransitions(t *testing.T) {
	// Set up temporary config path
	tempDir := t.TempDir()
	os.Setenv("CONTAINER_CONFIG_PATH", tempDir)
	defer os.Unsetenv("CONTAINER_CONFIG_PATH")
	
	var execCommand func(string, ...string) *exec.Cmd = exec.Command
	mockState, cleanup := SetupMockCommand(&execCommand)
	defer cleanup()
	
	// Disable debug output for cleaner test output
	mockState.SetDebug(false)
	
	// Add a container in stopped state
	mockState.AddContainer("test-container", "stopped")
	
	// Test start transition
	cmd := execCommand("lxc-start", "-n", "test-container")
	err := cmd.Run()
	if err != nil {
		t.Errorf("Start command should succeed: %v", err)
	}
	
	state, _ := mockState.GetContainerState("test-container")
	if state != "RUNNING" {
		t.Errorf("Expected state RUNNING after start, got %s", state)
	}
	
	// Test freeze transition
	cmd = execCommand("lxc-freeze", "-n", "test-container")
	err = cmd.Run()
	if err != nil {
		t.Errorf("Freeze command should succeed: %v", err)
	}
	
	state, _ = mockState.GetContainerState("test-container")
	if state != "FROZEN" {
		t.Errorf("Expected state FROZEN after freeze, got %s", state)
	}
	
	// Test unfreeze transition
	cmd = execCommand("lxc-unfreeze", "-n", "test-container")
	err = cmd.Run()
	if err != nil {
		t.Errorf("Unfreeze command should succeed: %v", err)
	}
	
	state, _ = mockState.GetContainerState("test-container")
	if state != "RUNNING" {
		t.Errorf("Expected state RUNNING after unfreeze, got %s", state)
	}
	
	// Test stop transition
	cmd = execCommand("lxc-stop", "-n", "test-container")
	err = cmd.Run()
	if err != nil {
		t.Errorf("Stop command should succeed: %v", err)
	}
	
	state, _ = mockState.GetContainerState("test-container")
	if state != "STOPPED" {
		t.Errorf("Expected state STOPPED after stop, got %s", state)
	}
}

func TestMockCommandState_InvalidStateTransitions(t *testing.T) {
	var execCommand func(string, ...string) *exec.Cmd = exec.Command
	mockState, cleanup := SetupMockCommand(&execCommand)
	defer cleanup()
	
	// Disable debug output for cleaner test output
	mockState.SetDebug(false)
	
	// Add a container in stopped state
	mockState.AddContainer("test-container", "stopped")
	
	// Test invalid transitions
	tests := []struct {
		name    string
		command string
		state   string
	}{
		{"stop already stopped", "lxc-stop", "stopped"},
		{"freeze stopped container", "lxc-freeze", "stopped"},
		{"unfreeze non-frozen container", "lxc-unfreeze", "stopped"},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set the container to the test state
			mockState.SetContainerState("test-container", tt.state)
			
			cmd := execCommand(tt.command, "-n", "test-container")
			err := cmd.Run()
			if err == nil {
				t.Errorf("Command %s should fail for invalid state transition from %s", tt.command, tt.state)
			}
		})
	}
}

func TestMockCommandState_Integration(t *testing.T) {
	// Test complete integration of mock command system
	tempDir := t.TempDir()
	os.Setenv("CONTAINER_CONFIG_PATH", tempDir)
	defer os.Unsetenv("CONTAINER_CONFIG_PATH")
	
	var execCommand func(string, ...string) *exec.Cmd = exec.Command
	mockState, cleanup := SetupMockCommand(&execCommand)
	defer cleanup()
	
	mockState.SetDebug(false)
	
	// Test complete container lifecycle
	containerName := "integration-test"
	
	// Add container
	mockState.AddContainer(containerName, "stopped")
	
	// Verify container exists
	if !mockState.ContainerExists(containerName) {
		t.Error("Container should exist after AddContainer")
	}
	
	// Start container
	cmd := execCommand("lxc-start", "-n", containerName)
	err := cmd.Run()
	if err != nil {
		t.Errorf("Start command failed: %v", err)
	}
	
	// Verify command was recorded
	if !mockState.CommandWasCalled("lxc-start", "-n", containerName) {
		t.Error("Start command should be recorded in history")
	}
	
	// Verify state change
	state, exists := mockState.GetContainerState(containerName)
	if !exists {
		t.Error("Container should exist")
	}
	if state != "RUNNING" {
		t.Errorf("Expected RUNNING state, got %s", state)
	}
	
	// Remove container
	mockState.RemoveContainer(containerName)
	if mockState.ContainerExists(containerName) {
		t.Error("Container should not exist after removal")
	}
}