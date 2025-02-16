package container

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"proxmox-lxc-compose/pkg/logging"
)

// NetworkStats represents network interface statistics
type NetworkStats struct {
	Interface string
	RxBytes   int64
	RxPackets int64
	RxErrors  int64
	RxDropped int64
	TxBytes   int64
	TxPackets int64
	TxErrors  int64
	TxDropped int64
	Timestamp time.Time
}

// GetNetworkStats retrieves network statistics for a container
func (m *LXCManager) GetNetworkStats(name string) ([]NetworkStats, error) {
	netPath := filepath.Join("/sys/fs/cgroup/lxc", name, "devices")
	if _, err := os.Stat(netPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("container %s is not running", name)
	}

	// Get list of network interfaces in the container
	procNetDev := filepath.Join(netPath, "net/dev")
	file, err := os.Open(procNetDev)
	if err != nil {
		return nil, fmt.Errorf("failed to read network stats: %w", err)
	}
	defer file.Close()

	var stats []NetworkStats
	scanner := bufio.NewScanner(file)
	// Skip header lines
	scanner.Scan()
	scanner.Scan()

	// Parse interface statistics
	now := time.Now()
	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(line)
		if len(fields) < 17 {
			continue
		}

		// Remove colon from interface name
		iface := strings.TrimSuffix(fields[0], ":")

		// Skip loopback interface
		if iface == "lo" {
			continue
		}

		stat := NetworkStats{
			Interface: iface,
			Timestamp: now,
		}

		// Parse receive statistics
		if v, err := strconv.ParseInt(fields[1], 10, 64); err == nil {
			stat.RxBytes = v
		}
		if v, err := strconv.ParseInt(fields[2], 10, 64); err == nil {
			stat.RxPackets = v
		}
		if v, err := strconv.ParseInt(fields[3], 10, 64); err == nil {
			stat.RxErrors = v
		}
		if v, err := strconv.ParseInt(fields[4], 10, 64); err == nil {
			stat.RxDropped = v
		}

		// Parse transmit statistics
		if v, err := strconv.ParseInt(fields[9], 10, 64); err == nil {
			stat.TxBytes = v
		}
		if v, err := strconv.ParseInt(fields[10], 10, 64); err == nil {
			stat.TxPackets = v
		}
		if v, err := strconv.ParseInt(fields[11], 10, 64); err == nil {
			stat.TxErrors = v
		}
		if v, err := strconv.ParseInt(fields[12], 10, 64); err == nil {
			stat.TxDropped = v
		}

		stats = append(stats, stat)
		logging.Debug("Collected network stats",
			"container", name,
			"interface", stat.Interface,
			"rx_bytes", stat.RxBytes,
			"tx_bytes", stat.TxBytes,
		)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading network stats: %w", err)
	}

	return stats, nil
}
