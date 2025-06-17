#!/bin/bash

set -e

echo "🚀 Hybrid LXC Integration Test"
echo "==============================="
echo "Detects systemd vs manual setup automatically"
echo ""

# Detect environment type
if systemctl --version >/dev/null 2>&1 && [ -f /lib/systemd/systemd ]; then
    ENVIRONMENT="systemd"
    echo "🔧 Environment: Systemd-enabled Docker"
else
    ENVIRONMENT="manual"
    echo "🔧 Environment: Manual setup Docker"
fi

echo ""
echo "🔧 Test 1: Basic LXC Installation"
if lxc-create --version >/dev/null 2>&1; then
    echo "✅ LXC is installed: $(lxc-create --version)"
else
    echo "❌ LXC installation failed"
    exit 1
fi

echo ""
echo "🔧 Test 2: Go Installation"
if go version >/dev/null 2>&1; then
    echo "✅ Go is installed: $(go version)"
else
    echo "❌ Go installation failed"
    exit 1
fi

echo ""
echo "🔧 Test 3: Network Bridge Status"
if ip link show lxcbr0 >/dev/null 2>&1; then
    echo "✅ LXC bridge exists:"
    ip addr show lxcbr0 | head -3
    BRIDGE_STATUS="available"
else
    echo "⚠️  LXC bridge not available"
    BRIDGE_STATUS="missing"
fi

echo ""
echo "🔧 Test 4: LXC Configuration"
if [ -f /etc/lxc/default.conf ]; then
    echo "✅ LXC configuration exists:"
    echo "   $(grep -c "lxc\." /etc/lxc/default.conf) configuration lines found"
else
    echo "⚠️  LXC configuration missing"
fi

echo ""
echo "🔧 Test 5: Build Test"
if [ -d "/opt/lxc-compose-test/source" ]; then
    echo "📦 Building lxc-compose..."
    cd /opt/lxc-compose-test/source
    
    if GOCACHE=/tmp/gocache go build -buildvcs=false -o /var/tmp/lxc-compose ./cmd/lxc-compose/; then
        echo "✅ Build successful!"
        BUILD_STATUS="success"
        
        echo ""
        echo "🔧 Test 6: CLI Functionality"
        echo "📋 Help command test:"
        /var/tmp/lxc-compose --help | head -10
        
        echo ""
        echo "📋 Configuration parsing test:"
        if [ -f "/opt/lxc-compose-test/test-data/lxc-compose.yml" ]; then
            cd /opt/lxc-compose-test/test-data
            /var/tmp/lxc-compose --config lxc-compose.yml ps
            CLI_STATUS="success"
        else
            echo "⚠️  Test configuration file not found"
            CLI_STATUS="config_missing"
        fi
        
    else
        echo "❌ Build failed"
        BUILD_STATUS="failed"
        CLI_STATUS="skipped"
    fi
else
    echo "⚠️  Source code not available"
    BUILD_STATUS="source_missing"
    CLI_STATUS="skipped"
fi

# Environment-specific tests
echo ""
if [ "$ENVIRONMENT" = "systemd" ]; then
    echo "🔧 Test 7: Systemd-specific Tests"
    echo "📋 Systemd status:"
    systemctl is-system-running 2>/dev/null || echo "   State: $(systemctl is-system-running 2>/dev/null || echo 'unknown')"
    
    echo "📋 LXC service status:"
    systemctl status lxc-net --no-pager -l 2>/dev/null || echo "   LXC service not active"
else
    echo "🔧 Test 7: Manual Setup Tests"
    echo "📋 Manual bridge creation test:"
    if [ "$BRIDGE_STATUS" = "missing" ]; then
        echo "   Attempting manual bridge creation..."
        brctl addbr testbr0 2>/dev/null && ip link delete testbr0 2>/dev/null && echo "   ✅ Bridge creation capability: OK" || echo "   ⚠️  Bridge creation capability: Limited"
    else
        echo "   ✅ Bridge already available"
    fi
fi

echo ""
echo "🎉 Hybrid Integration Test Complete!"
echo "====================================="

echo ""
echo "📊 Test Results Summary:"
echo "• Environment: $ENVIRONMENT"
echo "• LXC Tools: $([ "$(lxc-create --version 2>/dev/null)" ] && echo 'OK' || echo 'Failed')"
echo "• Go Build: $BUILD_STATUS"
echo "• CLI Tests: $CLI_STATUS"
echo "• Network Bridge: $BRIDGE_STATUS"

echo ""
echo "💡 Usage Examples:"
if [ "$BUILD_STATUS" = "success" ]; then
    echo "   # Test binary directly:"
    echo "   /var/tmp/lxc-compose --help"
    echo ""
    echo "   # Test with config:"
    echo "   cd /opt/lxc-compose-test/test-data"
    echo "   /var/tmp/lxc-compose --config lxc-compose.yml ps"
fi

echo ""
echo "🔗 Next Steps:"
if [ "$ENVIRONMENT" = "systemd" ]; then
    echo "   For full systemd testing: /opt/lxc-compose-test/systemd-test.sh"
fi
echo "   For basic validation: /opt/lxc-compose-test/basic-test.sh"
echo "   For simple LXC tests: /opt/lxc-compose-test/simple-test.sh" 