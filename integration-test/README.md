# LXC-Compose Integration Testing

This directory contains comprehensive integration testing solutions for the proxmox-lxc-compose project.

## 🚀 Quick Start

Run the interactive test selector to choose your preferred method:

```bash
cd integration-test
./quick-test.sh
```

### Environment Variables for SSH Testing

For automated SSH testing, configure these environment variables:

**Quick Setup (Recommended):**
```bash
./setup-env.sh    # Interactive configuration with SSH testing
```

**Manual Setup:**
```bash
export LXC_COMPOSE_REMOTE_HOST=192.168.1.100    # Proxmox/LXC host IP
export LXC_COMPOSE_REMOTE_USER=root             # SSH username
export LXC_COMPOSE_SSH_KEY=~/.ssh/id_rsa        # SSH private key path
```

With these set, SSH tests run without prompts:

```bash
./ssh-runner/enhanced-ssh-test.sh    # Direct test execution
./quick-test.sh                      # Option 5 will use env vars
```

## 📋 Testing Methods (Recommended Order)

### 1. 🔐 SSH-based Testing (Most Accurate)

**Best for production validation** - Tests on actual Proxmox/LXC infrastructure:

```bash
cd ssh-runner
./enhanced-ssh-test.sh
# Follow interactive prompts for host selection
```

**Pros:**
- ✅ Real Proxmox/LXC environment
- ✅ Actual container isolation and networking
- ✅ Production-like testing
- ✅ Most accurate results

**Cons:**
- ⚠️ Requires existing host with SSH access
- ⚠️ Slower setup for first-time use

### 2. 🖥️ Multipass VM Testing (Recommended for Development)

**Best for local development** - Clean Ubuntu VM with native LXC:

```bash
cd multipass
./setup-multipass-test.sh
```

**Pros:**
- ✅ Clean, isolated Ubuntu environment
- ✅ Real LXC (not containerized)
- ✅ Fast VM creation and cleanup
- ✅ Live code mounting for development
- ✅ No external dependencies

**Cons:**
- ⚠️ Requires Multipass installation
- ⚠️ Uses local VM resources

### 3. 🐳 Simplified Docker Testing (Quick Validation)

**Fastest option** - Basic LXC environment in container:

```bash
cd docker-lxc
docker-compose up -d
docker-compose exec lxc-test-env /opt/lxc-compose-test/basic-test.sh
```

**Pros:**
- ✅ Fastest setup (2-3 minutes)
- ✅ No external dependencies
- ✅ Good for basic validation
- ✅ Removed DinD complexity

**Cons:**
- ⚠️ Limited LXC functionality in containers
- ⚠️ Not 100% accurate to real environment

### 4. 📦 Vagrant VM Testing (Traditional)

**Good for compatibility testing** - Full VM with VirtualBox/VMware:

```bash
cd vagrant
vagrant up
vagrant ssh
```

**Pros:**
- ✅ Full VM environment
- ✅ Good compatibility testing
- ✅ Traditional virtualization

**Cons:**
- ⚠️ Slower than other methods
- ⚠️ Requires VirtualBox/VMware
- ⚠️ Higher resource usage

## 🧪 What Gets Tested

### Core Functionality
- ✅ Binary compilation and execution
- ✅ Configuration file parsing and validation
- ✅ Container creation (`up` command)
- ✅ Container listing (`ps` command)
- ✅ Container management (pause/unpause)
- ✅ Container cleanup (`down` command)

### Advanced Features
- ✅ Multi-container orchestration
- ✅ Network configuration and bridges
- ✅ Storage management and mounts
- ✅ Security settings and isolation
- ✅ Resource limits (CPU/Memory)

### Error Handling
- ✅ Invalid configurations
- ✅ Missing dependencies
- ✅ Network conflicts
- ✅ Resource constraints

## 📁 Directory Structure

```
integration-test/
├── docker-lxc/              # Simplified Docker testing (no DinD)
│   ├── Dockerfile           # Ubuntu + LXC + Go
│   ├── docker-compose.yml   # Container orchestration
│   ├── basic-test.sh        # Environment validation
│   ├── simple-test.sh       # Basic LXC functionality
│   └── test-data/           # Test configurations
├── ssh-runner/              # SSH-based testing
│   ├── enhanced-ssh-test.sh # Interactive SSH testing
│   └── ssh-integration-test.sh # Original SSH script
├── multipass/               # Multipass VM testing
│   ├── setup-multipass-test.sh # VM creation and testing
│   └── README.md           # Multipass-specific docs
├── vagrant/                 # Vagrant VM testing
│   └── Vagrantfile         # VM configuration
├── proxmox-real/           # Real Proxmox configurations
│   └── proxmox-lxc-compose.yml
├── performance/            # Performance testing
│   └── load-test.sh
├── quick-test.sh          # Interactive test selector
└── README.md              # This file
```

## 🔧 Prerequisites by Method

### SSH Testing
- SSH access to Proxmox/LXC host
- Go compiler on target host
- Root or sudo access

### Multipass Testing
- Multipass installed (`snap install multipass`)
- 4GB+ available RAM

### Docker Testing
- Docker and Docker Compose
- 2GB+ available RAM

### Vagrant Testing
- Vagrant and VirtualBox/VMware
- 8GB+ available RAM

## 🚀 Recommended Workflow

1. **Development**: Use Multipass for iterative testing
2. **Pre-commit**: Use simplified Docker for quick validation
3. **Pre-release**: Use SSH testing on real Proxmox
4. **CI/CD**: GitHub Actions with LXC setup

## 📊 Test Results Interpretation

### ✅ Success Indicators
- LXC tools found and working
- Go compilation successful
- Container creation and management
- Network bridge configuration
- Resource limit enforcement

### ⚠️ Expected Limitations
- Some LXC features limited in containers
- Bridge creation may fail (Docker environments)
- Template downloads may be restricted
- Systemd services limited in containers

### ❌ Failure Indicators
- LXC tools not found
- Go compiler missing or version incompatible
- Permission denied errors
- Network configuration failures

## 🔍 Debugging Tips

### Enable Detailed Logging
```bash
# Add debug flags to lxc-compose
./lxc-compose --debug -v -f lxc-compose.yml up web
```

### Check LXC Environment
```bash
# Verify LXC installation
lxc-create --version
lxc-ls -f

# Check networking
ip addr show lxcbr0
brctl show

# Test container creation
sudo lxc-create -n test -t download -- -d ubuntu -r focal -a amd64
sudo lxc-start -n test
sudo lxc-info -n test
sudo lxc-destroy -n test
```

### Network Troubleshooting
```bash
# Restart LXC networking
sudo systemctl restart lxc-net

# Manual bridge setup
sudo brctl addbr lxcbr0
sudo ip addr add 10.0.3.1/24 dev lxcbr0
sudo ip link set lxcbr0 up
```

## 🤝 Contributing

To add new integration tests:

1. Choose the appropriate testing method directory
2. Add test scenarios to existing scripts
3. Create new test configurations in `test-data/`
4. Update method-specific README files
5. Test across multiple methods for validation

## 🎯 Next Steps

1. **Immediate**: Try the quick test runner: `./quick-test.sh`
2. **Development**: Set up Multipass for ongoing work
3. **Production**: Validate with SSH testing on real hosts
4. **Automation**: Integrate with your CI/CD pipeline

## 📞 Support

If you encounter issues:

1. Try the quick test runner first: `./quick-test.sh`
2. Check prerequisites for your chosen method
3. Use SSH method for most accurate validation
4. Check the project's main documentation
5. Enable debug logging for detailed error information