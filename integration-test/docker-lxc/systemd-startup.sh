#!/bin/bash

# Systemd startup script for Docker container
echo "🚀 Starting systemd-enabled LXC container..."

# Ensure systemd can work properly in container
mount -t tmpfs tmpfs /tmp
mount -t tmpfs tmpfs /run
mount -t tmpfs tmpfs /run/lock

# Start systemd as PID 1
exec /lib/systemd/systemd --system --unit=multi-user.target 