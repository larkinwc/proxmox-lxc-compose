# LXC-Compose Integration Testing - Usage Guide

## 🚀 Quick Start

### 1. Basic Environment Test
```bash
cd integration-test/docker-lxc
docker-compose up -d
docker-compose exec lxc-test-env /opt/lxc-compose-test/basic-test.sh
```

### 2. Interactive Testing
```bash
# Enter the container for manual testing
docker-compose exec lxc-test-env bash

# Inside container:
cd /opt/lxc-compose-test/source
go build -buildvcs=false -o lxc-compose ./cmd/lxc-compose/
./lxc-compose --help
```

### 3. SSH-based Testing (Real Proxmox)
```bash
cd integration-test/ssh-runner
./ssh-integration-test.sh your-proxmox-host.com root
```

## 🧪 Available Tests

### Docker-based Tests
- `basic-test.sh` - Environment validation ✅ **WORKING**
- `simple-test.sh` - LXC functionality test
- `run-integration-tests.sh` - Full integration suite

### SSH-based Tests
- `ssh-integration-test.sh` - Remote Proxmox testing

### Vagrant-based Tests
- `Vagrantfile` - Local VM testing

## 🔧 Troubleshooting

### Build Issues
If you get Go dependency errors:
```bash
# Fix go.mod version
sed -i 's/go 1.23.4/go 1.18/' go.mod

# Or update container Go version
# See Dockerfile modifications in main README
```

### Network Issues
LXC bridge creation may fail in Docker - this is expected and doesn't prevent testing.

### Permission Issues
```bash
# If you get permission errors:
sudo usermod -aG docker $USER
# Log out and back in
```

## 📊 Test Results Interpretation

### ✅ Success Indicators
- LXC tools found and working
- Go compilation successful
- Source code accessible
- Test configurations loaded

### ⚠️ Expected Warnings
- Bridge creation failures (Docker limitation)
- Systemd service issues (containerized environment)
- Some LXC features limited (nested containers)

### ❌ Failure Indicators
- LXC tools not found
- Go compiler missing
- Source code not mounted
- Permission denied errors

## 🎯 Next Steps

1. **Fix Dependencies**: Update go.mod for Go 1.18 compatibility
2. **Test Your Code**: Use the working environment to test lxc-compose
3. **Real Proxmox Testing**: Use SSH method for production validation
4. **CI/CD Integration**: Add to GitHub Actions for automated testing

## 📞 Support

If you encounter issues:
1. Check Docker is running and you have permissions
2. Verify source code is in the correct directory
3. Try the basic-test.sh first to validate environment
4. Use SSH method for most accurate Proxmox testing