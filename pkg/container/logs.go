package container

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// LogOptions represents options for log streaming
type LogOptions struct {
	Follow    bool
	Tail      int
	Since     time.Time
	Timestamp bool
}

// GetLogs returns the logs for a container
func (m *LXCManager) GetLogs(name string, opts LogOptions) (io.ReadCloser, error) {
	if !m.ContainerExists(name) {
		return nil, fmt.Errorf("container %s does not exist", name)
	}

	logPath := filepath.Join(m.configPath, name, "console.log")
	file, err := os.Open(logPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file: %w", err)
	}

	if opts.Follow {
		return m.followLogs(name, file, opts)
	}

	// If we're not following, handle tail and since options
	if opts.Tail > 0 || !opts.Since.IsZero() {
		filtered, err := m.filterLogs(file, opts)
		if err != nil {
			file.Close()
			return nil, err
		}
		return io.NopCloser(filtered), nil
	}

	return file, nil
}

// FollowLogs follows the logs of a container and writes them to the given writer
func (m *LXCManager) FollowLogs(name string, w io.Writer) error {
	logs, err := m.GetLogs(name, LogOptions{Follow: true, Timestamp: true})
	if err != nil {
		return err
	}
	defer logs.Close()

	_, err = io.Copy(w, logs)
	return err
}

// followLogs returns a ReadCloser that follows the log output
func (m *LXCManager) followLogs(name string, file io.ReadCloser, _ LogOptions) (io.ReadCloser, error) {
	// Use lxc-attach to tail the logs
	logPath := filepath.Join(m.configPath, name, "console.log")
	cmd := ExecCommand("lxc-attach", "-n", name, "--", "tail", "-f", logPath)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		file.Close()
		return nil, fmt.Errorf("failed to get stdout pipe: %w", err)
	}
	if err := cmd.Start(); err != nil {
		file.Close()
		return nil, fmt.Errorf("failed to start log following: %w", err)
	}
	return &logReader{
		cmd:    cmd,
		stdout: stdout,
		file:   file,
	}, nil
}

// filterLogs returns a reader that filters log lines based on options
func (m *LXCManager) filterLogs(r io.Reader, opts LogOptions) (io.Reader, error) {
	var lines []string
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := scanner.Text()

		// Parse timestamp if present
		var lineTime time.Time
		if strings.HasPrefix(line, "[") && strings.Contains(line, "]") {
			ts := strings.TrimPrefix(strings.Split(line, "]")[0], "[")
			if t, err := time.Parse(time.RFC3339, ts); err == nil {
				lineTime = t
			}
		}

		// Filter by since if specified
		if !opts.Since.IsZero() && !lineTime.IsZero() && lineTime.Before(opts.Since) {
			continue
		}

		lines = append(lines, line)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to read logs: %w", err)
	}

	// Apply tail option if specified
	if opts.Tail > 0 && len(lines) > opts.Tail {
		lines = lines[len(lines)-opts.Tail:]
	}

	return strings.NewReader(strings.Join(lines, "\n")), nil
}

// logReader implements io.ReadCloser for log following
type logReader struct {
	cmd    *exec.Cmd
	stdout io.ReadCloser
	file   io.ReadCloser
}

func (r *logReader) Read(p []byte) (n int, err error) {
	return r.stdout.Read(p)
}

func (r *logReader) Close() error {
	r.file.Close()
	if err := r.cmd.Process.Kill(); err != nil {
		return fmt.Errorf("failed to kill log process: %w", err)
	}
	return r.cmd.Wait()
}
