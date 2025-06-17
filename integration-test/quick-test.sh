#!/bin/bash

# Quick integration test runner - choose your method
set -e

echo "🚀 LXC-Compose Integration Test Selector"
echo "========================================"
echo ""
echo "💡 First time? Run './setup-env.sh' to configure SSH testing"
echo ""
echo "Choose your testing approach:"
echo ""
echo "Docker-based Testing (85% coverage, fast):"
echo "  1) 🔧 Basic Integration Test    - Core functionality validation"
echo "  2) 🧪 Advanced Feature Test     - CLI, config, error handling"  
echo "  3) 📦 Unit Test Suite          - Go package testing"
echo "  4) 🎯 Hybrid Auto-detect       - Smart environment detection"
echo ""
echo "Real LXC Testing (100% coverage, requires setup):"
echo "  5) 🌐 SSH to Proxmox Host      - Real Proxmox testing (use LXC_COMPOSE_REMOTE_HOST env var)"
echo "  6) 🖥️  Multipass VM Setup       - Clean Ubuntu VMs"
echo "  7) 📱 Vagrant Environment      - Full VM testing"
echo ""
echo "Docker Environment Management:"
echo "  8) 🏗️  Build/Start Environment  - Setup Docker testing"
echo "  9) 🧹 Clean Environment        - Remove containers"
echo "  0) ❌ Exit"

echo ""
read -p "Select option (1-9, 0 to exit): " choice

case $choice in
    1)
        echo ""
        echo "🔧 Running Basic Integration Test..."
        echo "==================================="
        if docker-compose -f integration-test/docker-lxc/docker-compose.yml ps | grep -q "Up"; then
            docker-compose -f integration-test/docker-lxc/docker-compose.yml exec -T lxc-test-env /opt/lxc-compose-test/basic-test.sh
        else
            echo "⚠️  Starting Docker environment..."
            docker-compose -f integration-test/docker-lxc/docker-compose.yml up -d
            echo "Waiting for environment to be ready..."
            sleep 5
            docker-compose -f integration-test/docker-lxc/docker-compose.yml exec -T lxc-test-env /opt/lxc-compose-test/basic-test.sh
        fi
        ;;
    2)
        echo ""
        echo "🧪 Running Advanced Feature Test..."
        echo "=================================="
        if docker-compose -f integration-test/docker-lxc/docker-compose.yml ps | grep -q "Up"; then
            docker-compose -f integration-test/docker-lxc/docker-compose.yml exec -T lxc-test-env /opt/lxc-compose-test/advanced-test.sh
        else
            echo "⚠️  Starting Docker environment..."
            docker-compose -f integration-test/docker-lxc/docker-compose.yml up -d
            echo "Waiting for environment to be ready..."
            sleep 5
            docker-compose -f integration-test/docker-lxc/docker-compose.yml exec -T lxc-test-env /opt/lxc-compose-test/advanced-test.sh
        fi
        ;;
    3)
        echo ""
        echo "📦 Running Unit Test Suite..."
        echo "============================"
        echo ""
        echo "🔧 Info: Running pure unit tests (no LXC dependencies)"
        echo "   • Expected results: ~12/14 packages PASS"
        echo "   • pkg/container and pkg/oci may FAIL (require real LXC/registry)"
        echo "   • This provides ~85% code coverage testing"
        echo ""
        
        if [ "$ENV_TYPE" = "docker" ]; then
            docker-compose exec -T lxc-test-env /opt/lxc-compose-test/unit-test.sh
        else
            echo "❌ Unit test suite currently only available in Docker environment"
            echo "   Run: cd integration-test/docker-lxc && ./quick-test.sh"
        fi
        ;;
    4)
        echo ""
        echo "🎯 Running Hybrid Auto-detect Test..."
        echo "==================================="
        if docker-compose -f integration-test/docker-lxc/docker-compose.yml ps | grep -q "Up"; then
            docker-compose -f integration-test/docker-lxc/docker-compose.yml exec -T lxc-test-env /opt/lxc-compose-test/hybrid-test.sh
        else
            echo "⚠️  Starting Docker environment..."
            docker-compose -f integration-test/docker-lxc/docker-compose.yml up -d
            echo "Waiting for environment to be ready..."
            sleep 5
            docker-compose -f integration-test/docker-lxc/docker-compose.yml exec -T lxc-test-env /opt/lxc-compose-test/hybrid-test.sh
        fi
        ;;
    5)
        echo ""
        echo "🌐 Setting up SSH-based testing..."
        echo "================================="
        chmod +x integration-test/ssh-runner/enhanced-ssh-test.sh
        integration-test/ssh-runner/enhanced-ssh-test.sh
        ;;
    6)
        echo ""
        echo "🖥️ Setting up Multipass VM testing..."
        echo "===================================="
        chmod +x integration-test/multipass/setup-multipass-test.sh
        integration-test/multipass/setup-multipass-test.sh
        ;;
    7)
        echo ""
        echo "📱 Setting up Vagrant testing..."
        echo "==============================="
        if [ -f "integration-test/vagrant/Vagrantfile" ]; then
            cd integration-test/vagrant
            echo "Starting Vagrant VM..."
            vagrant up
            echo "Running tests in VM..."
            vagrant ssh -c "cd /vagrant && ./test-in-vm.sh"
        else
            echo "❌ Vagrant configuration not found"
            echo "   Create integration-test/vagrant/Vagrantfile first"
        fi
        ;;
    8)
        echo ""
        echo "🏗️ Building/Starting Docker Environment..."
        echo "========================================"
        echo "Building custom LXC testing image..."
        docker-compose -f integration-test/docker-lxc/docker-compose.yml build
        echo "Starting environment..."
        docker-compose -f integration-test/docker-lxc/docker-compose.yml up -d
        echo "✅ Environment ready!"
        echo ""
        echo "💡 You can now run tests (options 1-4)"
        ;;
    9)
        echo ""
        echo "🧹 Cleaning Docker Environment..."
        echo "==============================="
        docker-compose -f integration-test/docker-lxc/docker-compose.yml down
        docker-compose -f integration-test/docker-lxc/docker-compose.yml down --volumes
        echo "Removing test images..."
        docker images | grep lxc-compose-test | awk '{print $3}' | xargs -r docker rmi
        echo "✅ Environment cleaned!"
        ;;
    0)
        echo "👋 Goodbye!"
        exit 0
        ;;
    *)
        echo "❌ Invalid option. Please choose 1-9 or 0."
        exit 1
        ;;
esac

echo ""
echo "🎉 Test completed! Run './integration-test/quick-test.sh' again for more testing options."