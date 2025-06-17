#!/bin/bash

echo "🔧 Testing init system detection fix..."

# Build the fixed version
echo "Building fixed version..."
go build -o /tmp/lxc-compose-fixed ./cmd/lxc-compose

# Clean up any existing test containers
echo "Cleaning up existing containers..."
sudo /tmp/lxc-compose-fixed down -f test-simple.yml 2>/dev/null || true

# Create a simple container to test init detection
echo "Creating container with init detection..."
sudo /tmp/lxc-compose-fixed up -f test-simple.yml --debug

# Check if the container was created successfully
echo "Checking container status..."
sudo lxc-ls -f

# Try to get logs if available
echo "Checking for container logs..."
if [ -d "/var/lib/lxc/web" ]; then
    echo "Container directory exists"
    ls -la /var/lib/lxc/web/
    if [ -f "/var/lib/lxc/web/start.log" ]; then
        echo "=== Container startup log ==="
        sudo cat /var/lib/lxc/web/start.log | tail -20
    fi
    if [ -f "/var/lib/lxc/web/config" ]; then
        echo "=== Container config ==="
        sudo cat /var/lib/lxc/web/config | grep -E "(init|cmd)"
    fi
fi

# Clean up
echo "Cleaning up..."
sudo /tmp/lxc-compose-fixed down -f test-simple.yml 