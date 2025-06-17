#!/bin/bash

# Performance and load testing for lxc-compose
set -e

echo "🚀 LXC-Compose Performance Testing"
echo "=================================="

# Build the binary
go build -o lxc-compose ../../cmd/lxc-compose/

# Test 1: Single container creation time
echo "🧪 Test 1: Single container creation performance"
time ./lxc-compose -f ../docker-lxc/test-data/lxc-compose.yml up web

# Test 2: Multiple container creation
echo "🧪 Test 2: Multiple container creation performance"
time ./lxc-compose -f ../docker-lxc/test-data/lxc-compose.yml up

# Test 3: Container lifecycle performance
echo "🧪 Test 3: Container lifecycle performance"
time (
    ./lxc-compose -f ../docker-lxc/test-data/lxc-compose.yml up web
    ./lxc-compose pause web
    ./lxc-compose unpause web
    ./lxc-compose down web
)

# Test 4: Stress test - multiple operations
echo "🧪 Test 4: Stress test - 10 rapid up/down cycles"
for i in {1..10}; do
    echo "Cycle $i/10"
    ./lxc-compose -f ../docker-lxc/test-data/lxc-compose.yml up web >/dev/null 2>&1
    ./lxc-compose down web >/dev/null 2>&1
done

# Test 5: Memory usage monitoring
echo "🧪 Test 5: Memory usage monitoring"
echo "Starting memory monitor..."
(
    while true; do
        ps aux | grep lxc-compose | grep -v grep || true
        sleep 1
    done
) &
MONITOR_PID=$!

./lxc-compose -f ../docker-lxc/test-data/lxc-compose.yml up
sleep 5
./lxc-compose down --rm

kill $MONITOR_PID 2>/dev/null || true

echo "✅ Performance testing completed!"