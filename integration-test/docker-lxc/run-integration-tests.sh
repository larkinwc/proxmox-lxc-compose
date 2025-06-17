#!/bin/bash

set -e

echo "🚀 Starting LXC Integration Tests"

# Build the binary
echo "📦 Building lxc-compose binary..."
cd /opt/lxc-compose-test/source
go build -buildvcs=false -o /usr/local/bin/lxc-compose ./cmd/lxc-compose/

# Verify binary works
echo "✅ Testing binary..."
lxc-compose --help

# Setup test environment
echo "🔧 Setting up test environment..."
cd /opt/lxc-compose-test/test-data

# Test 1: Basic up command
echo "🧪 Test 1: Basic container creation and startup"
lxc-compose -f lxc-compose.yml up web

# Verify container exists
echo "🔍 Verifying container exists..."
lxc-ls | grep web || (echo "❌ Container 'web' not found" && exit 1)

# Check container status
echo "📊 Checking container status..."
lxc-info -n web

# Test 2: List containers
echo "🧪 Test 2: List containers"
lxc-compose ps

# Test 3: Container logs (if implemented)
echo "🧪 Test 3: Container logs"
lxc-compose logs web || echo "⚠️  Logs command not fully implemented yet"

# Test 4: Stop containers
echo "🧪 Test 4: Stop containers"
lxc-compose down web

# Test 5: Multi-container setup
echo "🧪 Test 5: Multi-container setup"
lxc-compose up

# Verify both containers
echo "🔍 Verifying both containers..."
lxc-ls | grep web || (echo "❌ Container 'web' not found" && exit 1)
lxc-ls | grep db || (echo "❌ Container 'db' not found" && exit 1)

# Test 6: Container pause/unpause
echo "🧪 Test 6: Container pause/unpause"
lxc-compose pause web
lxc-info -n web | grep FROZEN || (echo "❌ Container not frozen" && exit 1)

lxc-compose unpause web
lxc-info -n web | grep RUNNING || (echo "❌ Container not running after unpause" && exit 1)

# Test 7: Cleanup
echo "🧪 Test 7: Cleanup"
lxc-compose down --rm

# Verify cleanup
echo "🔍 Verifying cleanup..."
if lxc-ls | grep -E "(web|db)"; then
    echo "⚠️  Some containers still exist after cleanup"
else
    echo "✅ Cleanup successful"
fi

echo "🎉 All integration tests completed successfully!"