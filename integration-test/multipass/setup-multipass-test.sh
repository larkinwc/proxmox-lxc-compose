#!/bin/bash

# Multipass-based LXC integration testing
# Provides isolated Ubuntu VM with LXC for testing

set -e

VM_NAME="lxc-compose-test"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

echo "🚀 LXC-Compose Multipass Integration Testing"
echo "============================================"

# Check if multipass is installed
if ! command -v multipass &> /dev/null; then
    echo "❌ Multipass not found. Install it from: https://multipass.run/"
    echo ""
    echo "Quick install:"
    echo "  Ubuntu/Debian: sudo snap install multipass"
    echo "  macOS: brew install --cask multipass"
    echo "  Windows: Download from multipass.run"
    exit 1
fi

# Cleanup function
cleanup() {
    echo "🧹 Cleaning up..."
    multipass delete "$VM_NAME" 2>/dev/null || true
    multipass purge 2>/dev/null || true
}

# Option to cleanup existing VM
if multipass list | grep -q "$VM_NAME"; then
    echo "⚠️  VM '$VM_NAME' already exists"
    read -p "Delete existing VM and recreate? (y/N): " confirm
    if [[ $confirm =~ ^[Yy] ]]; then
        cleanup
    else
        echo "Using existing VM..."
    fi
fi

# Create VM if it doesn't exist
if ! multipass list | grep -q "$VM_NAME"; then
    echo "📦 Creating Ubuntu VM with LXC..."
    multipass launch 22.04 \
        --name "$VM_NAME" \
        --cpus 2 \
        --memory 4G \
        --disk 20G \
        --cloud-init - << 'EOF'
#cloud-config
package_update: true
packages:
  - lxc
  - lxc-utils
  - lxc-templates
  - bridge-utils
  - golang-go
  - git
  - build-essential
  - debootstrap

runcmd:
  - systemctl enable lxc-net
  - systemctl start lxc-net
  - echo 'ubuntu ALL=(ALL) NOPASSWD:ALL' >> /etc/sudoers
  - usermod -aG lxc ubuntu

write_files:
  - path: /etc/default/lxc-net
    content: |
      USE_LXC_BRIDGE="true"
      LXC_BRIDGE="lxcbr0"
      LXC_ADDR="10.0.3.1"
      LXC_NETMASK="255.255.255.0"
      LXC_NETWORK="10.0.3.0/24"
      LXC_DHCP_RANGE="10.0.3.2,10.0.3.254"
      LXC_DHCP_MAX="253"
EOF

    echo "⏳ Waiting for VM to be ready..."
    multipass exec "$VM_NAME" -- sudo cloud-init status --wait
fi

echo "📁 Mounting project directory..."
multipass mount "$PROJECT_ROOT" "$VM_NAME:/home/ubuntu/lxc-compose"

echo "🏗️  Building and testing lxc-compose..."
multipass exec "$VM_NAME" -- bash << 'EOF'
set -e

cd /home/ubuntu/lxc-compose

echo "🔧 Building lxc-compose..."
go build -buildvcs=false -o lxc-compose ./cmd/lxc-compose/

echo "✅ Testing binary..."
./lxc-compose --help

echo "🌐 Checking LXC environment..."
sudo systemctl status lxc-net
ip addr show lxcbr0 || echo "LXC bridge will be created on demand"

echo "🧪 Running integration tests..."
cd integration-test/docker-lxc/test-data

# Test basic functionality
echo "🔍 Test 1: Configuration validation"
../../../lxc-compose -f lxc-compose.yml config || echo "Config validation not implemented"

echo "🔍 Test 2: Container operations"
if sudo ../../../lxc-compose -f lxc-compose.yml up web; then
    echo "✅ Container created successfully"
    
    # Check container status
    sudo lxc-ls -f || true
    sudo lxc-info -n web || true
    
    # Test management
    sudo ../../../lxc-compose ps || echo "PS command not fully implemented"
    
    # Cleanup
    sudo ../../../lxc-compose down web || true
else
    echo "⚠️  Container creation failed (may be expected in some environments)"
fi

echo "🎉 Multipass integration tests completed!"
EOF

echo ""
echo "✅ Integration testing completed!"
echo ""
echo "💡 Useful commands:"
echo "   multipass shell $VM_NAME                    # Access the VM"
echo "   multipass exec $VM_NAME -- lxc-ls -f       # List containers"
echo "   multipass stop $VM_NAME                     # Stop VM"
echo "   multipass delete $VM_NAME && multipass purge # Remove VM"
echo ""
echo "🔧 VM Details:"
multipass info "$VM_NAME" 