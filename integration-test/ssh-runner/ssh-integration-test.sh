#!/bin/bash

# SSH-based integration test runner for real Proxmox hosts
# Usage: ./ssh-integration-test.sh <proxmox-host> <username>

set -e

PROXMOX_HOST=${1:-"proxmox.local"}
USERNAME=${2:-"root"}
SSH_KEY=${3:-"~/.ssh/id_rsa"}
REMOTE_TEST_DIR="/tmp/lxc-compose-integration"

echo "🚀 Starting SSH-based integration tests on $PROXMOX_HOST"

# Function to run commands on remote host
run_remote() {
    ssh -i "$SSH_KEY" -o StrictHostKeyChecking=no "$USERNAME@$PROXMOX_HOST" "$@"
}

# Function to copy files to remote host
copy_to_remote() {
    scp -i "$SSH_KEY" -o StrictHostKeyChecking=no -r "$1" "$USERNAME@$PROXMOX_HOST:$2"
}

echo "📦 Preparing remote environment..."

# Create remote test directory
run_remote "mkdir -p $REMOTE_TEST_DIR"

# Copy source code
echo "📁 Copying source code..."
copy_to_remote "../../" "$REMOTE_TEST_DIR/source"

# Copy test data
copy_to_remote "../docker-lxc/test-data" "$REMOTE_TEST_DIR/"

echo "🔧 Building binary on remote host..."
run_remote "cd $REMOTE_TEST_DIR/source && go build -o $REMOTE_TEST_DIR/lxc-compose ./cmd/lxc-compose/"

echo "🧪 Running integration tests on remote host..."

# Create remote test script
cat > /tmp/remote-test.sh << 'EOF'
#!/bin/bash
set -e

cd /tmp/lxc-compose-integration

echo "✅ Testing binary..."
./lxc-compose --help

echo "🧪 Test 1: Basic container operations"
cd test-data
../lxc-compose -f lxc-compose.yml up web

echo "🔍 Verifying container..."
lxc-ls | grep web || (echo "❌ Container not found" && exit 1)

echo "📊 Container status:"
lxc-info -n web

echo "🧪 Test 2: Container management"
../lxc-compose ps
../lxc-compose pause web
../lxc-compose unpause web

echo "🧪 Test 3: Multi-container"
../lxc-compose up

echo "🧪 Test 4: Cleanup"
../lxc-compose down --rm

echo "🎉 Remote integration tests completed!"
EOF

# Copy and run the test script
copy_to_remote "/tmp/remote-test.sh" "$REMOTE_TEST_DIR/remote-test.sh"
run_remote "chmod +x $REMOTE_TEST_DIR/remote-test.sh && $REMOTE_TEST_DIR/remote-test.sh"

echo "🧹 Cleaning up remote environment..."
run_remote "rm -rf $REMOTE_TEST_DIR"

echo "✅ SSH integration tests completed successfully!"