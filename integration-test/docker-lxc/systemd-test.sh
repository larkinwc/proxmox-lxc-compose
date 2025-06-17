#!/bin/bash

set -e

echo "🚀 Systemd-enabled LXC Integration Test"
echo "======================================="

# Wait for systemd to fully initialize
echo "⏳ Waiting for systemd initialization..."
sleep 10

echo "🔧 Test 1: Systemd Status"
systemctl --version
systemctl is-system-running || echo "Systemd state: $(systemctl is-system-running)"

echo ""
echo "🔧 Test 2: LXC Service Status"
systemctl status lxc-net || echo "LXC-net service status checked"

echo ""
echo "🔧 Test 3: Manual LXC Network Start"
systemctl start lxc-net || echo "LXC-net start attempted"
sleep 3

echo ""
echo "🔧 Test 4: Bridge Status"
if ip link show lxcbr0 >/dev/null 2>&1; then
    echo "✅ LXC bridge (lxcbr0) exists"
    ip addr show lxcbr0 | head -5
else
    echo "⚠️  LXC bridge not available"
    echo "   Attempting manual bridge creation..."
    brctl addbr lxcbr0 || echo "Bridge creation failed"
    ip addr add 10.0.3.1/24 dev lxcbr0 || echo "IP assignment failed"
    ip link set lxcbr0 up || echo "Bridge activation failed"
    
    if ip link show lxcbr0 >/dev/null 2>&1; then
        echo "✅ Manual bridge creation successful"
        ip addr show lxcbr0 | head -3
    else
        echo "❌ Bridge creation completely failed"
    fi
fi

echo ""
echo "🔧 Test 5: LXC Environment Check"
lxc-create --version
echo "LXC config directory: $(ls -la /etc/lxc/ 2>/dev/null || echo 'Not accessible')"
echo "LXC var directory: $(ls -la /var/lib/lxc/ 2>/dev/null || echo 'Not accessible')"

echo ""
echo "🔧 Test 6: Go Build Test"
if [ -d "/opt/lxc-compose-test/source" ]; then
    cd /opt/lxc-compose-test/source
    echo "📦 Building lxc-compose..."
    if GOCACHE=/tmp/gocache go build -buildvcs=false -o /var/tmp/lxc-compose ./cmd/lxc-compose/; then
        echo "✅ Build successful!"
        
        echo "📋 Testing help command:"
        /var/tmp/lxc-compose --help
        
        echo ""
        echo "🧪 Testing configuration parsing:"
        cd /opt/lxc-compose-test/test-data
        /var/tmp/lxc-compose --config lxc-compose.yml ps
        
        echo ""
        echo "🧪 Testing container creation (may fail):"
        /var/tmp/lxc-compose up -f lxc-compose.yml web || echo "Container creation failed (expected)"
        
    else
        echo "❌ Build failed"
    fi
else
    echo "⚠️  Source code not available"
fi

echo ""
echo "🔧 Test 7: Service Status Summary"
echo "Services status:"
systemctl list-units --failed || echo "Failed to list failed units"

echo ""
echo "🎉 Systemd LXC Integration Test Complete!"
echo "========================================"

echo ""
echo "📊 Summary:"
echo "• Systemd: $(systemctl is-system-running 2>/dev/null || echo 'Unknown')"
echo "• LXC Tools: $(lxc-create --version 2>/dev/null || echo 'Not available')"
echo "• Bridge: $(ip link show lxcbr0 >/dev/null 2>&1 && echo 'Available' || echo 'Not available')"
echo "• Go Build: $([ -f /var/tmp/lxc-compose ] && echo 'Successful' || echo 'Failed')"
echo ""
echo "💡 For interactive testing:"
echo "   docker-compose -f docker-compose.systemd.yml exec lxc-systemd-test bash" 