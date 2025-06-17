#!/bin/bash

echo "🚀 Starting LXC Integration Test Environment"

# Check if systemd is available and working
if systemctl --version >/dev/null 2>&1 && [ -f /lib/systemd/systemd ]; then
    echo "🔧 Systemd detected - using systemd services..."
    
    # Start essential systemd services
    systemctl start systemd-logind 2>/dev/null || echo "⚠️ systemd-logind not available"
    systemctl start dbus 2>/dev/null || echo "⚠️ dbus not available"
    
    # Start LXC networking via systemd
    echo "🌐 Starting LXC networking via systemd..."
    systemctl start lxc-net 2>/dev/null || echo "⚠️ LXC networking service not available"
    
    # Give systemd services time to start
    sleep 5
else
    echo "🔧 No systemd detected - using manual setup..."
fi

echo "🌐 Setting up LXC networking..."

# Check if LXC bridge exists, create manually if needed
if ! ip link show lxcbr0 >/dev/null 2>&1; then
    echo "🔧 Creating LXC bridge manually..."
    brctl addbr lxcbr0 2>/dev/null || echo "⚠️ Bridge creation failed"
    ip addr add 10.0.3.1/24 dev lxcbr0 2>/dev/null || echo "⚠️ IP assignment failed"
    ip link set lxcbr0 up 2>/dev/null || echo "⚠️ Bridge activation failed"
fi

# Verify bridge status
if ip link show lxcbr0 >/dev/null 2>&1; then
    echo "✅ LXC bridge (lxcbr0) is available"
    ip addr show lxcbr0 | head -3
else
    echo "⚠️ LXC bridge setup failed - container networking may be limited"
fi

echo "🔧 Setting up LXC environment..."

# Ensure LXC directories exist with proper permissions
mkdir -p /var/lib/lxc /var/cache/lxc /var/log/lxc
chmod 755 /var/lib/lxc /var/cache/lxc /var/log/lxc

# Setup LXC configuration if not exists
if [ ! -f /etc/lxc/default.conf.backup ]; then
    cp /etc/lxc/default.conf /etc/lxc/default.conf.backup 2>/dev/null || echo "⚠️ Could not backup LXC config"
fi

echo "🧪 LXC Environment Setup Complete"
echo "================================="
echo "Available test commands:"
echo "  /opt/lxc-compose-test/basic-test.sh     - Basic integration test"
echo "  /opt/lxc-compose-test/simple-test.sh    - Simple LXC functionality test"
echo "  /opt/lxc-compose-test/hybrid-test.sh    - Smart auto-detection test"
echo ""
echo "💡 Interactive mode: docker-compose exec lxc-test-env bash"
echo ""

# Start SSH daemon if requested
if [ "$START_SSH" = "true" ]; then
    echo "🔑 Starting SSH daemon..."
    service ssh start 2>/dev/null || echo "⚠️ SSH daemon not available"
fi

# Keep container running based on mode
if [ "$1" = "systemd" ]; then
    echo "🔧 Switching to systemd init..."
    exec /lib/systemd/systemd --system --unit=multi-user.target
elif [ "$1" = "interactive" ]; then
    echo "🎯 Starting interactive shell..."
    exec bash
else
    echo "🏃 Running in daemon mode (default)..."
    # Keep container running indefinitely
    tail -f /dev/null
fi