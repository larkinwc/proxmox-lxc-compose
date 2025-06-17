# Systemd-enabled Docker LXC Testing

⚠️ **WARNING: This approach is complex and has significant limitations compared to real VMs or SSH testing.**

## 🤔 Should You Use This?

**Short Answer: Probably not.** Here's why:

### ❌ **Limitations of Systemd in Docker**

1. **Security Issues**: Requires `--privileged` mode (major security risk)
2. **Complex Setup**: Many systemd services don't work properly in containers
3. **Limited Functionality**: Some LXC features still won't work
4. **Maintenance Overhead**: Difficult to debug and maintain
5. **Performance Impact**: Slower than lightweight alternatives

### ✅ **Better Alternatives**

| Method | Setup Time | Accuracy | Security | Maintenance |
|--------|------------|----------|----------|-------------|
| **SSH Testing** | 2 minutes | 100% | ✅ Secure | ✅ Simple |
| **Multipass VM** | 3 minutes | 100% | ✅ Secure | ✅ Simple |
| **Systemd Docker** | 10+ minutes | 70% | ❌ Privileged | ❌ Complex |

## 🚀 If You Still Want to Try It...

### Prerequisites

- Docker with privileged container support
- Host system with systemd (Linux)
- 4GB+ available RAM
- Understanding of security implications

### Quick Start

```bash
# Build and start systemd container
docker-compose -f docker-compose.systemd.yml build
docker-compose -f docker-compose.systemd.yml up -d

# Wait for systemd to initialize (important!)
sleep 30

# Run tests
docker-compose -f docker-compose.systemd.yml exec lxc-systemd-test /opt/lxc-compose-test/systemd-test.sh
```

### Expected Results

```
🚀 Systemd-enabled LXC Integration Test
=======================================

✅ Systemd Status: OK
⚠️  LXC Service: Partially working
✅ Bridge Creation: Manual setup works
✅ Go Build: Successful
⚠️  Container Creation: May fail (Docker dependency issues)
```

## 🔧 Troubleshooting

### Common Issues

1. **"System has not been booted with systemd"**
   - This is normal during startup, wait 30+ seconds

2. **"Failed to connect to bus"**
   - Container may still be initializing
   - Check: `docker logs lxc-systemd-integration-test`

3. **LXC networking fails**
   - Some systemd services are masked for container compatibility
   - Manual bridge creation should work

4. **Container creation fails**
   - Expected - Docker-in-Docker limitations remain
   - Build and CLI testing should work

### Debugging Commands

```bash
# Check systemd status
docker-compose -f docker-compose.systemd.yml exec lxc-systemd-test systemctl status

# Check service logs
docker-compose -f docker-compose.systemd.yml exec lxc-systemd-test journalctl -u lxc-net

# Interactive debugging
docker-compose -f docker-compose.systemd.yml exec lxc-systemd-test bash
```

## 📊 What Actually Works

| Feature | Status | Notes |
|---------|--------|-------|
| ✅ Systemd Init | Works | With limitations |
| ✅ LXC Tools | Works | Installation and basic commands |
| ✅ Go Compilation | Works | Full build process |
| ✅ CLI Testing | Works | Help, config parsing |
| ⚠️ Network Bridge | Partial | Manual creation works |
| ❌ LXC Containers | Fails | Docker dependency issues |
| ❌ Full Integration | Fails | Nested container limitations |

## 🎯 Recommended Workflow

Instead of this complex setup, use:

1. **For Development**: Multipass VMs
   ```bash
   cd ../multipass
   ./setup-multipass-test.sh
   ```

2. **For Production Testing**: SSH to real hosts
   ```bash
   cd ../ssh-runner
   ./enhanced-ssh-test.sh
   ```

3. **For Quick Validation**: Original simplified Docker
   ```bash
   docker-compose up -d
   docker-compose exec lxc-test-env /opt/lxc-compose-test/basic-test.sh
   ```

## 🧹 Cleanup

```bash
# Stop and remove systemd container
docker-compose -f docker-compose.systemd.yml down
docker system prune -f

# Remove systemd images
docker rmi $(docker images | grep systemd | awk '{print $3}')
```

## 💡 Key Takeaways

- **Systemd in Docker is possible** but comes with significant trade-offs
- **Real VMs or SSH testing** provide much better results with less complexity
- **This approach mainly useful** for understanding systemd/Docker integration challenges
- **For actual LXC testing**, use the recommended alternatives

## 🔗 Better Alternatives

- [Multipass Testing](../multipass/README.md) - Clean VMs, best for development
- [SSH Testing](../ssh-runner/enhanced-ssh-test.sh) - Real hosts, best for accuracy
- [Simple Docker](./basic-test.sh) - Quick validation without systemd complexity 