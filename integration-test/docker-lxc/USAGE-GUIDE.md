# Docker LXC Integration Testing - Complete Guide

This directory provides **3 different Docker approaches** for LXC integration testing, from simple to complex.

## 🎯 **Quick Decision Guide**

| Use Case | Recommended Approach | Time Investment | Accuracy |
|----------|---------------------|-----------------|----------|
| **Quick Development Testing** | [Standard Docker](#1-standard-docker-recommended) | 2 minutes | 85% |
| **CI/CD Pipeline** | [Standard Docker](#1-standard-docker-recommended) | 2 minutes | 85% |
| **Systemd Understanding** | [Systemd Docker](#2-systemd-enabled-docker) | 10+ minutes | 90% |
| **Production Testing** | [SSH/Multipass](../README.md) | 3 minutes | 100% |

## 1. **Standard Docker (Recommended)**

**Enhanced docker-compose.yml with systemd support but simple usage**

### ✅ **What You Get**
- ✅ LXC tools installation and testing
- ✅ Go compilation and CLI testing
- ✅ Network bridge setup (manual fallback)
- ✅ Optional systemd support if available
- ✅ Quick iteration cycles
- ✅ Secure (no privileged mode issues)

### 🚀 **Usage**

```bash
# Quick start
docker-compose up -d

# Run the smart hybrid test (detects environment automatically)
docker-compose exec lxc-test-env /opt/lxc-compose-test/hybrid-test.sh

# Interactive mode
docker-compose exec lxc-test-env bash
```

### 📊 **Expected Results**
```
🚀 Hybrid LXC Integration Test
===============================
🔧 Environment: Manual setup Docker

✅ LXC is installed: 5.0.0
✅ Go is installed: go version go1.23.4 linux/amd64
⚠️  LXC bridge not available (expected in Docker)
✅ Build successful!
✅ CLI tests pass
```

## 2. **Systemd-enabled Docker**

**Full systemd support with privileged containers**

### ⚠️ **Security Warning**
This approach requires `--privileged` mode, which has significant security implications.

### ✅ **What You Get**
- ✅ Real systemd PID 1
- ✅ Systemd service management
- ✅ Better LXC networking simulation
- ⚠️ Still limited by Docker container restrictions
- ❌ Security risks from privileged mode

### 🚀 **Usage**

```bash
# Build and start systemd container
docker-compose -f docker-compose.systemd.yml build
docker-compose -f docker-compose.systemd.yml up -d

# IMPORTANT: Wait for systemd initialization
sleep 30

# Run full systemd tests
docker-compose -f docker-compose.systemd.yml exec lxc-systemd-test /opt/lxc-compose-test/systemd-test.sh

# Interactive mode
docker-compose -f docker-compose.systemd.yml exec lxc-systemd-test bash
```

### 📊 **Expected Results**
```
🚀 Systemd-enabled LXC Integration Test
=======================================

✅ Systemd Status: OK (degraded)
⚠️  LXC Service: Partially working
✅ Bridge Creation: Manual setup works
✅ Go Build: Successful
⚠️  Container Creation: May fail (Docker limitations)
```

## 3. **Simple Legacy Docker**

**Original simple approach without systemd complexity**

### 🚀 **Usage**
```bash
# Use original simple tests
docker-compose exec lxc-test-env /opt/lxc-compose-test/basic-test.sh
docker-compose exec lxc-test-env /opt/lxc-compose-test/simple-test.sh
```

## 🔧 **Available Test Scripts**

| Script | Purpose | Environment | Duration |
|--------|---------|-------------|----------|
| `hybrid-test.sh` | **Smart auto-detection** | Any | 30s |
| `basic-test.sh` | Manual setup validation | Standard | 15s |
| `simple-test.sh` | LXC tools only | Standard | 10s |
| `systemd-test.sh` | Full systemd testing | Systemd | 60s |

## 🎯 **Workflow Examples**

### **Development Workflow (Recommended)**
```bash
# 1. Quick validation
docker-compose up -d
docker-compose exec lxc-test-env /opt/lxc-compose-test/hybrid-test.sh

# 2. Interactive development
docker-compose exec lxc-test-env bash
cd /opt/lxc-compose-test/source
go build -o /tmp/lxc-compose ./cmd/lxc-compose/
/tmp/lxc-compose --help

# 3. Test configuration changes
cd /opt/lxc-compose-test/test-data
/tmp/lxc-compose --config lxc-compose.yml ps
```

### **CI/CD Workflow**
```bash
# In your CI script
docker-compose up -d
docker-compose exec -T lxc-test-env /opt/lxc-compose-test/hybrid-test.sh
docker-compose exec -T lxc-test-env bash -c "
  cd /opt/lxc-compose-test/source && 
  go build -buildvcs=false -o /tmp/lxc-compose ./cmd/lxc-compose/ &&
  cd /opt/lxc-compose-test/test-data &&
  /tmp/lxc-compose --config lxc-compose.yml ps
"
docker-compose down
```

### **Systemd Research Workflow**
```bash
# For understanding systemd/LXC interactions
docker-compose -f docker-compose.systemd.yml up -d
sleep 30
docker-compose -f docker-compose.systemd.yml exec lxc-systemd-test bash

# Inside container
systemctl status
journalctl -u lxc-net
systemctl start lxc-net
ip addr show lxcbr0
```

## 📊 **Feature Comparison**

| Feature | Standard | Systemd | Legacy |
|---------|----------|---------|--------|
| Setup Time | ⚡ 30s | 🐌 5+ min | ⚡ 20s |
| Security | ✅ Safe | ❌ Privileged | ✅ Safe |
| LXC Tools | ✅ Full | ✅ Full | ✅ Full |
| Go Build | ✅ Works | ✅ Works | ✅ Works |
| CLI Testing | ✅ Works | ✅ Works | ✅ Works |
| Systemd Services | ⚠️ Detection | ✅ Full | ❌ None |
| Network Bridge | ⚠️ Manual | ✅ Service | ⚠️ Manual |
| Container Creation | ❌ Limited | ❌ Limited | ❌ Limited |

## 🧹 **Cleanup**

```bash
# Standard cleanup
docker-compose down
docker system prune -f

# Systemd cleanup
docker-compose -f docker-compose.systemd.yml down
docker rmi $(docker images | grep systemd | awk '{print $3}') 2>/dev/null || true

# Full cleanup
docker system prune -a -f
```

## 💡 **Key Insights**

1. **Standard Docker is usually sufficient** for development and CI/CD
2. **Systemd Docker is educational** but has limited practical benefits
3. **Real LXC container creation will fail** in all Docker approaches due to nested container limitations
4. **For actual container testing**, use [SSH](../ssh-runner/) or [Multipass](../multipass/) approaches
5. **The hybrid test script adapts** to whatever environment it's in

## 🔗 **When to Use Alternatives**

- **For real LXC testing**: Use [SSH Testing](../ssh-runner/enhanced-ssh-test.sh)
- **For clean environments**: Use [Multipass VMs](../multipass/setup-multipass-test.sh)
- **For maximum accuracy**: Test on actual Proxmox hosts

## 🎓 **Learning Outcomes**

Using these Docker approaches, you'll understand:
- How LXC tools work in containerized environments
- Systemd service management challenges in Docker
- Network bridge setup and management
- Go compilation and CLI testing workflows
- The limitations of nested containerization

This knowledge transfers directly to real LXC/Proxmox environments while providing fast iteration cycles during development. 