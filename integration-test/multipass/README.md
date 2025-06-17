# Multipass-based LXC Integration Testing

This directory provides **Multipass-based integration testing** for lxc-compose, offering a clean and isolated Ubuntu VM environment for testing LXC functionality.

## 🚀 Quick Start

```bash
cd integration-test/multipass
./setup-multipass-test.sh
```

## 📋 What This Provides

- **Clean Ubuntu 22.04 VM** with LXC pre-installed
- **Isolated testing environment** (no Docker complexity)
- **Real LXC functionality** (not containerized limitations)
- **Automatic project mounting** (live code changes)
- **Easy cleanup and recreation**

## 🛠️ Prerequisites

Install Multipass:

```bash
# Ubuntu/Debian
sudo snap install multipass

# macOS
brew install --cask multipass

# Windows
# Download from https://multipass.run/
```

## 🧪 Testing Workflow

1. **VM Creation**: Automatically creates Ubuntu VM with LXC
2. **Environment Setup**: Installs Go, LXC tools, networking
3. **Project Mounting**: Mounts your source code into VM
4. **Build & Test**: Compiles and tests lxc-compose
5. **Cleanup**: Easy VM removal when done

## 💡 Advantages over Docker-based Testing

- ✅ **No DinD complexity** - Real systemd and LXC
- ✅ **Native LXC networking** - Proper bridge setup
- ✅ **Real container isolation** - Not limited by Docker
- ✅ **Clean environment** - Fresh VM every time
- ✅ **Live development** - Mounted source directory

## 🔧 Manual Testing

Access the VM for manual testing:

```bash
multipass shell lxc-compose-test

# Inside VM
cd /home/ubuntu/lxc-compose
go build -o lxc-compose ./cmd/lxc-compose/

# Test with real LXC
sudo ./lxc-compose --help
sudo lxc-ls -f
```

## 🧹 Cleanup

```bash
multipass delete lxc-compose-test
multipass purge
```

## 🔄 Comparison with Other Methods

| Method | Speed | Accuracy | Setup | Dependencies |
|--------|-------|----------|-------|--------------|
| **Multipass** | ⭐⭐⭐ | ⭐⭐⭐⭐⭐ | ⭐⭐⭐⭐ | Multipass only |
| Docker (DinD) | ⭐⭐⭐⭐ | ⭐⭐ | ⭐⭐ | Docker + complexity |
| SSH Remote | ⭐⭐ | ⭐⭐⭐⭐⭐ | ⭐⭐⭐ | Remote host |
| Vagrant | ⭐⭐ | ⭐⭐⭐⭐ | ⭐⭐ | VirtualBox + Vagrant |

## 🎯 When to Use

- **Local development testing**
- **CI/CD integration** (if Multipass available)
- **Clean environment verification**
- **Before pushing to production**
- **When Docker-in-Docker is problematic** 