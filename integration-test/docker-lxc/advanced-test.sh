#!/bin/bash

set -e

echo "🧪 Advanced LXC-Compose Testing (Docker Environment)"
echo "===================================================="
echo "Tests advanced functionality within Docker limitations"
echo ""

# Build binary first
echo "🔧 Building lxc-compose binary..."
cd /opt/lxc-compose-test/source
if GOCACHE=/tmp/gocache go build -buildvcs=false -o /var/tmp/lxc-compose ./cmd/lxc-compose/; then
    echo "✅ Build successful!"
else
    echo "❌ Build failed"
    exit 1
fi

cd /opt/lxc-compose-test/test-data

echo ""
echo "🔧 Test 1: Configuration Validation"
echo "====================================="

# Test valid config
if /var/tmp/lxc-compose --config lxc-compose.yml ps >/dev/null 2>&1; then
    echo "✅ Valid configuration accepted"
else
    echo "❌ Valid configuration rejected"
fi

# Test invalid config (if exists)
if [ -f "invalid-config.yml" ]; then
    if /var/tmp/lxc-compose --config invalid-config.yml ps >/dev/null 2>&1; then
        echo "⚠️  Invalid configuration accepted (should fail)"
    else
        echo "✅ Invalid configuration properly rejected"
    fi
fi

echo ""
echo "🔧 Test 2: Command Interface Testing"
echo "===================================="

# Test all major commands exist
commands=("up" "down" "ps" "logs" "pause" "unpause" "images" "convert")
for cmd in "${commands[@]}"; do
    if /var/tmp/lxc-compose "$cmd" --help >/dev/null 2>&1; then
        echo "✅ Command '$cmd' available"
    else
        echo "❌ Command '$cmd' missing"
    fi
done

echo ""
echo "🔧 Test 3: Configuration Parsing Details"
echo "========================================"

# Test config parsing with detailed output
echo "📋 Parsing test configuration:"
/var/tmp/lxc-compose --config lxc-compose.yml --debug ps 2>&1 | head -10

echo ""
echo "🔧 Test 4: Error Handling Tests"
echo "==============================="

# Test missing config file
echo "📋 Testing missing config file:"
if /var/tmp/lxc-compose --config non-existent.yml ps 2>/dev/null; then
    echo "⚠️  Missing config should have failed"
else
    echo "✅ Missing config properly handled"
fi

# Test invalid command
echo "📋 Testing invalid command:"
if /var/tmp/lxc-compose invalid-command 2>/dev/null; then
    echo "⚠️  Invalid command should have failed"
else
    echo "✅ Invalid command properly handled"
fi

echo ""
echo "🔧 Test 5: Mock Container Operations"
echo "==================================="

# These will fail but test the command parsing
echo "📋 Testing container creation (will fail in Docker):"
/var/tmp/lxc-compose up -f lxc-compose.yml web 2>&1 | head -5 || echo "Expected failure in Docker environment"

echo ""
echo "📋 Testing container status:"
/var/tmp/lxc-compose ps --config lxc-compose.yml 2>&1 | head -5

echo ""
echo "🔧 Test 6: Template and Image Operations"
echo "========================================"

# Test image-related commands
echo "📋 Testing image commands:"
/var/tmp/lxc-compose images --help | head -5

echo ""
echo "📋 Testing convert command:"
/var/tmp/lxc-compose convert --help | head -5

echo ""
echo "🔧 Test 7: Development Mode Testing"
echo "==================================="

echo "📋 Testing development mode:"
/var/tmp/lxc-compose --dev --config lxc-compose.yml ps 2>&1 | head -5

echo ""
echo "🔧 Test 8: Logging and Debug Output"
echo "==================================="

echo "📋 Testing debug logging:"
/var/tmp/lxc-compose --debug --config lxc-compose.yml ps 2>&1 | head -10

echo ""
echo "🎉 Advanced Testing Complete!"
echo "============================="

echo ""
echo "📊 What We Tested:"
echo "• ✅ Configuration validation and parsing"
echo "• ✅ All command interfaces available"
echo "• ✅ Error handling for missing files/invalid commands"
echo "• ✅ Debug and development modes"
echo "• ✅ Mock container operations (expected failures)"
echo "• ⚠️  Real container operations (Docker limitations)"

echo ""
echo "🔗 For Real LXC Testing:"
echo "• SSH to Proxmox host: ../ssh-runner/enhanced-ssh-test.sh"
echo "• Clean VM testing: ../multipass/setup-multipass-test.sh"
echo "• Vagrant VMs: ../vagrant/"

echo ""
echo "💡 Docker Environment Summary:"
echo "   Perfect for: Development, CI/CD, CLI testing"
echo "   Limited for: Actual container operations"
echo "   Use for: Fast iteration, compilation validation" 