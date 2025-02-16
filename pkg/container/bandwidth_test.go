package container_test

import (
	"fmt"
	"testing"

	"proxmox-lxc-compose/pkg/common"
	"proxmox-lxc-compose/pkg/container"
	"proxmox-lxc-compose/pkg/internal/mock"
	testing_internal "proxmox-lxc-compose/pkg/internal/testing"
	"proxmox-lxc-compose/pkg/logging"
)

func TestNetworkBandwidthControl(t *testing.T) {
	// Create mock manager
	manager := container.NewMockLXCManager()

	// Setup command mocking
	oldExec := container.ExecCommand
	logging.Debug("Setting up mock command", "old_exec", fmt.Sprintf("%p", oldExec))
	mock, cleanup := mock.SetupMockCommand(&container.ExecCommand)
	logging.Debug("Mock command setup complete", "new_exec", fmt.Sprintf("%p", container.ExecCommand))
	defer func() {
		cleanup()
		container.ExecCommand = oldExec
		logging.Debug("Mock command cleanup complete", "restored_exec", fmt.Sprintf("%p", container.ExecCommand))
	}()

	containerName := "test-bandwidth"
	containerConfig := &common.Container{
		Network: &common.NetworkConfig{
			Interfaces: []common.NetworkInterface{
				{
					Type:      "bridge",
					Bridge:    "lxcbr0",
					Interface: "eth0",
					Bandwidth: &common.BandwidthLimit{
						IngressRate:  "1mbit",
						IngressBurst: "2mbit",
						EgressRate:   "500kbit",
						EgressBurst:  "1mbit",
					},
				},
			},
		},
	}

	// Setup container in mock
	mock.AddContainer(containerName, "RUNNING")
	err := manager.Create(containerName, containerConfig)
	testing_internal.AssertNoError(t, err)

	// Set initial bandwidth limits
	initialLimits := &common.BandwidthLimit{
		IngressRate:  "1mbit",
		IngressBurst: "2mbit",
		EgressRate:   "500kbit",
		EgressBurst:  "1mbit",
	}
	err = manager.UpdateNetworkBandwidthLimits(containerName, "eth0", initialLimits)
	testing_internal.AssertNoError(t, err)

	t.Run("verify_bandwidth_limits", func(t *testing.T) {
		logging.Debug("Running verify_bandwidth_limits test")
		limits, err := manager.GetNetworkBandwidthLimits(containerName, "eth0")
		if err != nil {
			logging.Debug("Error getting bandwidth limits",
				"error", err,
				"was_called", mock.WasCalled("lxc-attach -n test-bandwidth -- tc class show dev eth0"),
				"command", "lxc-attach -n test-bandwidth -- tc class show dev eth0")
		} else {
			logging.Debug("Got bandwidth limits",
				"ingress_rate", limits.IngressRate,
				"ingress_burst", limits.IngressBurst,
				"egress_rate", limits.EgressRate,
				"egress_burst", limits.EgressBurst)
		}
		testing_internal.AssertNoError(t, err)
		testing_internal.AssertEqual(t, "1mbit", limits.IngressRate)
		testing_internal.AssertEqual(t, "2mbit", limits.IngressBurst)
		testing_internal.AssertEqual(t, "500kbit", limits.EgressRate)
		testing_internal.AssertEqual(t, "1mbit", limits.EgressBurst)
	})

	t.Run("update_bandwidth_limits", func(t *testing.T) {
		newLimits := &common.BandwidthLimit{
			IngressRate:  "2mbit",
			IngressBurst: "4mbit",
			EgressRate:   "1mbit",
			EgressBurst:  "2mbit",
		}

		err := manager.UpdateNetworkBandwidthLimits(containerName, "eth0", newLimits)
		testing_internal.AssertNoError(t, err)

		limits, err := manager.GetNetworkBandwidthLimits(containerName, "eth0")
		testing_internal.AssertNoError(t, err)
		testing_internal.AssertEqual(t, "2mbit", limits.IngressRate)
		testing_internal.AssertEqual(t, "4mbit", limits.IngressBurst)
		testing_internal.AssertEqual(t, "1mbit", limits.EgressRate)
		testing_internal.AssertEqual(t, "2mbit", limits.EgressBurst)
	})
}
