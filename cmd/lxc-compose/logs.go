package main

import (
	"fmt"
	"io"
	"os"
	"time"

	"github.com/larkinwc/proxmox-lxc-compose/pkg/container"

	"github.com/spf13/cobra"
)

func init() {
	var follow bool
	var tail int
	var since string
	var timestamp bool

	var logsCmd = &cobra.Command{
		Use:   "logs [container]",
		Short: "View container logs",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			name := args[0]

			// Create container manager
			manager, err := container.NewLXCManager("/var/lib/lxc")
			if err != nil {
				return fmt.Errorf("failed to create container manager: %w", err)
			}

			// Parse since time if provided
			var sinceTime time.Time
			if since != "" {
				if since == "1h" || since == "24h" {
					duration, _ := time.ParseDuration(since)
					sinceTime = time.Now().Add(-duration)
				} else {
					sinceTime, err = time.Parse(time.RFC3339, since)
					if err != nil {
						return fmt.Errorf("invalid time format for --since: %w", err)
					}
				}
			}

			// Get logs
			logs, err := manager.GetLogs(name, container.LogOptions{
				Follow:    follow,
				Since:     sinceTime,
				Tail:      tail,
				Timestamp: timestamp,
			})
			if err != nil {
				return fmt.Errorf("failed to get logs: %w", err)
			}
			defer logs.Close()

			// Copy logs to stdout
			_, err = io.Copy(os.Stdout, logs)
			return err
		},
	}

	logsCmd.Flags().BoolVarP(&follow, "follow", "f", false, "Follow log output")
	logsCmd.Flags().IntVarP(&tail, "tail", "n", 0, "Number of lines to show from the end of the logs")
	logsCmd.Flags().StringVar(&since, "since", "", "Show logs since timestamp (RFC3339) or relative (e.g., 1h, 24h)")
	logsCmd.Flags().BoolVarP(&timestamp, "timestamps", "t", false, "Show timestamps")

	rootCmd.AddCommand(logsCmd)
}
