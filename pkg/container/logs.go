package container

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// LogOptions represents options for retrieving container logs
type LogOptions struct {
	Follow    bool      // Follow log output
	Since     time.Time // Show logs since timestamp
	Tail      int       // Number of lines to show from the end of the logs
	Timestamp bool      // Show timestamps
}

// GetLogs retrieves logs for a container
func (m *LXCManager) GetLogs(name string, opts LogOptions) (io.ReadCloser, error) {
	// Check if container exists
	if _, err := m.Get(name); err != nil {
		return nil, fmt.Errorf("container not found: %w", err)
	}

	// Get log file path
	logPath := filepath.Join(m.configPath, name, "console.log")
	if _, err := os.Stat(logPath); err != nil {
		return nil, fmt.Errorf("log file not found: %w", err)
	}

	// If not following logs, just return the file
	if !opts.Follow {
		file, err := os.Open(logPath)
		if err != nil {
			return nil, fmt.Errorf("failed to open log file: %w", err)
		}

		// Apply filters
		if opts.Since.IsZero() && opts.Tail == 0 {
			return file, nil
		}

		// Create filtered reader
		pr, pw := io.Pipe()
		go func() {
			defer pw.Close()
			defer file.Close()

			scanner := bufio.NewScanner(file)
			var lines []string
			for scanner.Scan() {
				line := scanner.Text()

				// Filter by timestamp if requested
				if !opts.Since.IsZero() {
					// Extract timestamp from brackets [TIMESTAMP]
					if len(line) > 2 && line[0] == '[' {
						closeBracket := strings.Index(line, "]")
						if closeBracket > 0 {
							tsStr := line[1:closeBracket]
							if ts, err := time.Parse(time.RFC3339, tsStr); err == nil {
								if ts.Before(opts.Since) {
									continue
								}
							}
						}
					}
				}

				if opts.Tail > 0 {
					// Store lines for tail
					lines = append(lines, line)
					if len(lines) > opts.Tail {
						lines = lines[1:]
					}
				} else {
					fmt.Fprintln(pw, line)
				}
			}

			// Write tail lines
			if opts.Tail > 0 {
				for _, line := range lines {
					fmt.Fprintln(pw, line)
				}
			}
		}()

		return pr, nil
	}

	// For following logs, use lxc-attach to tail the log file
	cmd := execCommand("lxc-attach", "-n", name, "--", "tail", "-f", "/var/log/console.log")
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to get stdout pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start log tail: %w", err)
	}

	// Create a pipe that will be closed when the command exits
	pr, pw := io.Pipe()
	go func() {
		defer pw.Close()
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			line := scanner.Text()
			if opts.Timestamp {
				fmt.Fprintf(pw, "[%s] %s\n", time.Now().Format(time.RFC3339), line)
			} else {
				fmt.Fprintln(pw, line)
			}
		}
		if err := cmd.Wait(); err != nil {
			fmt.Printf("Warning: log following ended with error: %v\n", err)
		}
	}()

	return pr, nil
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
