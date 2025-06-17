package testutil

import (
	"os/exec"
	"testing"
)

func TestExecCommand(t *testing.T) {
	tests := []struct {
		name string
		cmd  string
		args []string
	}{
		{"simple command", "echo", []string{"hello"}},
		{"command with multiple args", "ls", []string{"-l", "-a"}},
		{"command with no args", "pwd", []string{}},
		{"command with single arg", "cat", []string{"/dev/null"}},
		{"command with complex args", "find", []string{".", "-name", "*.go", "-type", "f"}},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := ExecCommand(tt.cmd, tt.args...)
			
			// Verify command was created
			if cmd == nil {
				t.Fatal("ExecCommand should return a non-nil command")
			}
			
			// Verify command path
			if cmd.Path == "" && cmd.Args[0] != tt.cmd {
				t.Errorf("Expected command to be %s, but got %s", tt.cmd, cmd.Args[0])
			}
			
			// Verify arguments
			expectedArgs := append([]string{tt.cmd}, tt.args...)
			if len(cmd.Args) != len(expectedArgs) {
				t.Errorf("Expected %d args, got %d", len(expectedArgs), len(cmd.Args))
			}
			
			for i, expectedArg := range expectedArgs {
				if i < len(cmd.Args) && cmd.Args[i] != expectedArg {
					t.Errorf("Expected arg %d to be %s, got %s", i, expectedArg, cmd.Args[i])
				}
			}
		})
	}
}

func TestExecCommand_ReturnType(t *testing.T) {
	// Test that ExecCommand returns the correct type
	cmd := ExecCommand("echo", "test")
	
	// Verify it's an *exec.Cmd by checking the type
	if cmd == nil {
		t.Fatal("ExecCommand should return a non-nil command")
	}
	
	// Check that it has the expected structure of *exec.Cmd
	if cmd.Args == nil {
		t.Error("Command should have Args field")
	}
}

func TestExecCommand_NoArgs(t *testing.T) {
	// Test command with no arguments
	cmd := ExecCommand("pwd")
	
	if cmd == nil {
		t.Fatal("ExecCommand should return a non-nil command")
	}
	
	// Should have exactly one argument (the command itself)
	if len(cmd.Args) != 1 {
		t.Errorf("Expected 1 arg for command with no args, got %d", len(cmd.Args))
	}
	
	if cmd.Args[0] != "pwd" {
		t.Errorf("Expected first arg to be 'pwd', got %s", cmd.Args[0])
	}
}

func TestExecCommand_EmptyCommand(t *testing.T) {
	// Test with empty command name
	cmd := ExecCommand("")
	
	if cmd == nil {
		t.Fatal("ExecCommand should return a non-nil command even for empty command")
	}
	
	// The command should still be created, even if it's invalid
	if len(cmd.Args) != 1 {
		t.Errorf("Expected 1 arg for empty command, got %d", len(cmd.Args))
	}
	
	if cmd.Args[0] != "" {
		t.Errorf("Expected first arg to be empty string, got %s", cmd.Args[0])
	}
}

func TestExecCommand_ManyArgs(t *testing.T) {
	// Test with many arguments
	args := []string{"-l", "-a", "-h", "--color=auto", "/tmp", "/var", "/usr"}
	cmd := ExecCommand("ls", args...)
	
	if cmd == nil {
		t.Fatal("ExecCommand should return a non-nil command")
	}
	
	expectedArgs := append([]string{"ls"}, args...)
	if len(cmd.Args) != len(expectedArgs) {
		t.Errorf("Expected %d args, got %d", len(expectedArgs), len(cmd.Args))
	}
	
	for i, expectedArg := range expectedArgs {
		if i < len(cmd.Args) && cmd.Args[i] != expectedArg {
			t.Errorf("Expected arg %d to be %s, got %s", i, expectedArg, cmd.Args[i])
		}
	}
}

func TestExecCommand_SpecialCharacters(t *testing.T) {
	// Test with arguments containing special characters
	specialArgs := []string{
		"file with spaces.txt",
		"file-with-dashes.txt",
		"file_with_underscores.txt",
		"file.with.dots.txt",
		"file@with@symbols.txt",
		"file$with$dollar.txt",
	}
	
	cmd := ExecCommand("touch", specialArgs...)
	
	if cmd == nil {
		t.Fatal("ExecCommand should return a non-nil command")
	}
	
	expectedArgs := append([]string{"touch"}, specialArgs...)
	if len(cmd.Args) != len(expectedArgs) {
		t.Errorf("Expected %d args, got %d", len(expectedArgs), len(cmd.Args))
	}
	
	for i, expectedArg := range expectedArgs {
		if i < len(cmd.Args) && cmd.Args[i] != expectedArg {
			t.Errorf("Expected arg %d to be %s, got %s", i, expectedArg, cmd.Args[i])
		}
	}
}

func TestExecCommand_CompareWithStdLib(t *testing.T) {
	// Test that our ExecCommand behaves the same as exec.Command
	testCases := []struct {
		name string
		cmd  string
		args []string
	}{
		{"echo command", "echo", []string{"hello", "world"}},
		{"ls command", "ls", []string{"-l"}},
		{"no args", "pwd", []string{}},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ourCmd := ExecCommand(tc.cmd, tc.args...)
			stdCmd := exec.Command(tc.cmd, tc.args...)
			
			// Compare the commands
			if len(ourCmd.Args) != len(stdCmd.Args) {
				t.Errorf("Args length mismatch: our=%d, std=%d", len(ourCmd.Args), len(stdCmd.Args))
			}
			
			for i := range ourCmd.Args {
				if i < len(stdCmd.Args) && ourCmd.Args[i] != stdCmd.Args[i] {
					t.Errorf("Arg %d mismatch: our=%s, std=%s", i, ourCmd.Args[i], stdCmd.Args[i])
				}
			}
		})
	}
}

func TestExecCommand_Execution(t *testing.T) {
	// Test that the returned command can actually be executed
	// We'll use simple, safe commands that should be available on most systems
	
	t.Run("echo command", func(t *testing.T) {
		cmd := ExecCommand("echo", "test")
		output, err := cmd.Output()
		
		if err != nil {
			t.Errorf("Failed to execute echo command: %v", err)
		}
		
		expectedOutput := "test\n"
		if string(output) != expectedOutput {
			t.Errorf("Expected output %q, got %q", expectedOutput, string(output))
		}
	})
	
	t.Run("pwd command", func(t *testing.T) {
		cmd := ExecCommand("pwd")
		output, err := cmd.Output()
		
		if err != nil {
			t.Errorf("Failed to execute pwd command: %v", err)
		}
		
		// Output should be non-empty and end with newline
		if len(output) == 0 {
			t.Error("Expected non-empty output from pwd command")
		}
		
		if output[len(output)-1] != '\n' {
			t.Error("Expected pwd output to end with newline")
		}
	})
}

func TestExecCommand_InvalidCommand(t *testing.T) {
	// Test with a command that doesn't exist
	cmd := ExecCommand("this-command-does-not-exist")
	
	if cmd == nil {
		t.Fatal("ExecCommand should return a non-nil command even for invalid commands")
	}
	
	// The command should be created, but execution should fail
	_, err := cmd.Output()
	if err == nil {
		t.Error("Expected error when executing non-existent command")
	}
}

func TestExecCommand_VariadicArgs(t *testing.T) {
	// Test that variadic arguments work correctly
	
	// Test with no variadic args
	cmd1 := ExecCommand("echo")
	if len(cmd1.Args) != 1 {
		t.Errorf("Expected 1 arg with no variadic args, got %d", len(cmd1.Args))
	}
	
	// Test with one variadic arg
	cmd2 := ExecCommand("echo", "hello")
	if len(cmd2.Args) != 2 {
		t.Errorf("Expected 2 args with one variadic arg, got %d", len(cmd2.Args))
	}
	
	// Test with multiple variadic args
	cmd3 := ExecCommand("echo", "hello", "world", "test")
	if len(cmd3.Args) != 4 {
		t.Errorf("Expected 4 args with three variadic args, got %d", len(cmd3.Args))
	}
	
	// Test with slice expansion
	args := []string{"arg1", "arg2", "arg3"}
	cmd4 := ExecCommand("echo", args...)
	expectedLen := 1 + len(args)
	if len(cmd4.Args) != expectedLen {
		t.Errorf("Expected %d args with slice expansion, got %d", expectedLen, len(cmd4.Args))
	}
}

// Benchmark the ExecCommand function
func BenchmarkExecCommand(b *testing.B) {
	for i := 0; i < b.N; i++ {
		cmd := ExecCommand("echo", "benchmark", "test")
		if cmd == nil {
			b.Fatal("ExecCommand returned nil")
		}
	}
}

func BenchmarkExecCommand_ManyArgs(b *testing.B) {
	args := make([]string, 100)
	for i := range args {
		args[i] = "arg" + string(rune(i))
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cmd := ExecCommand("echo", args...)
		if cmd == nil {
			b.Fatal("ExecCommand returned nil")
		}
	}
}