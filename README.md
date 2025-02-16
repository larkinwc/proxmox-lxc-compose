# proxmox-lxc-compose

A docker-compose like tool for managing LXC containers in Proxmox.

## Project Overview
The main purpose of the CLI is to read a lxc-compose.yml file and use it to create, start, stop, and update LXC containers. The tool validates YAML configuration files and ensures they contain the required minimum set of keys.

## Features

- Docker-compose style configuration for LXC containers
- Image management with OCI registry support
- Local image caching
- Structured logging
- Robust error handling with retries
- Configuration management via YAML/environment variables

### Completed Features
- Basic project structure and CLI framework setup
- Basic container management
  - Create and remove containers
  - Start and stop containers
  - List containers
  - Container state persistence
- Container operations
  - Pause/resume functionality
  - Container restart
  - Container update
- Container logs
  - Basic log retrieval
  - Log following
  - Log filtering (tail, since)
  - Timestamp support
- Configuration file parsing and validation
  - Basic YAML parsing
  - Configuration type definitions
  - Basic validation logic
  - Advanced configuration validation
    - Network validation (types, IP, DNS, interfaces, etc.)
    - Storage validation
    - Resource limits validation
- Advanced configuration options
  - CPU and memory limits
  - Network settings
    - DHCP/static IP
    - DNS settings
    - Hostname configuration
    - MTU settings
    - MAC address configuration
  - Storage settings
  - Environment variables
  - Entrypoint and command
  - Device/cgroups sharing
- OCI Image Support
  - Pull OCI images from registry
  - Export images as tarballs
  - Convert to LXC templates using docker2lxc
  - Local image caching with TTL
  - Automatic cache cleanup
  - Push/pull from OCI registries
  - Local image storage and retrieval
- Container networking features
  - Multiple network interfaces
    - Support for bridge, veth, macvlan, and physical interfaces
    - DHCP and static IP configuration
    - Interface-specific DNS settings
    - Custom MTU and MAC address configuration
  - Advanced DNS configuration
    - Per-interface DNS servers
    - Search domain support
  - Port forwarding
    - TCP/UDP port mapping
    - Automatic iptables rule management
  - Network isolation
    - Container network isolation
    - Interface-level network control
- Testing
  - Unit tests for state management
  - Unit tests for configuration parsing
  - Unit tests for container logging
  - Unit tests for container management
    - Pause/resume operations
    - Start/stop operations

### In Progress Features
- Container templates
  - Create template from container
  - List available templates
  - Delete template
  - Create container from template
- Additional unit tests
  - Start/stop operation tests
  - Create/remove operation tests
- OCI image management
  - Store OCI images locally
  - Retrieve OCI images locally
  - Push OCI images to registry
  - Pull OCI images from registry

### Planned Features
- Proper error handling and logging
  - Structured logging
  - Debug mode
  - Error recovery strategies
- Enhanced container networking
  - Multiple network interfaces
  - Advanced DNS configuration
  - Port forwarding
  - Network isolation
- Enhanced container storage
  - Multiple storage backends
  - Volume management
  - Backup/restore functionality
- Security features
  - Container isolation
  - Resource limits enforcement
  - Privilege management
  - SELinux/AppArmor support
- Testing improvements
  - Integration tests
  - End-to-end tests
  - Test coverage reporting

## Prerequisites

- Go 1.21 or later
- Docker (for OCI image conversion)
- LXC tools

## Installation

```bash
go install github.com/larkinwc/proxmox-lxc-compose@latest
```

Or build from source:

```bash
git clone https://github.com/larkinwc/proxmox-lxc-compose.git
cd proxmox-lxc-compose
go build -o lxc-compose ./cmd/lxc-compose
```

## Configuration

Configuration can be provided via:
- Configuration file (default: ~/.lxc-compose.yaml)
- Environment variables
- Command line flags

### Global Flags

- `--config`: Config file path (default: ~/.lxc-compose.yaml)
- `--debug`: Enable debug logging
- `--dev`: Enable development mode

### Image Cache Configuration
The tool includes an intelligent caching system for OCI images:

- Default cache location: ~/.lxc-compose/images
- Default TTL: 24 hours
- Automatic cleanup of expired images
- Cache can be configured via:
  - `LXC_COMPOSE_CACHE_TTL`: Cache TTL in seconds (default: 86400)
  - `LXC_COMPOSE_CACHE_DIR`: Custom cache directory path

## Usage

```bash
# Start containers
lxc-compose up

# Stop containers
lxc-compose down

# View container status
lxc-compose ps

# View container logs
lxc-compose logs [container_name]

# Pull container images
lxc-compose pull [image_name]

# Convert Docker images to LXC
lxc-compose convert [image_name]
```

### Converting OCI Images to LXC Templates

```bash
lxc-compose convert ubuntu:20.04
```

This will convert the Ubuntu 20.04 Docker image to an LXC template.

### Configuration File (lxc-compose.yml)

```yaml
version: "1.0"
services:
  web:
    image: ubuntu:20.04
    security:
      isolation: strict
      apparmor_profile: lxc-container-default-restricted
      capabilities:
        - NET_ADMIN
        - SYS_TIME
      selinux_context: system_u:system_r:container_t:s0
    cpu:
      cores: 2
      shares: 1024
    memory:
      limit: 2G
      swap: 1G
    network:
      type: bridge
      bridge: vmbr0
      ip: 192.168.1.100
    storage:
      root: 10G
      mounts:
        - source: /path/to/data
          target: /data
    environment:
      DB_HOST: db
      DB_PORT: 5432
    command: ["nginx", "-g", "daemon off;"]
  
  privileged-service:
    image: ubuntu:20.04
    security:
      isolation: privileged
      privileged: true
```

### Security Configuration

The tool supports comprehensive security configuration for containers:

- **Isolation Levels**:
  - `default`: System default security settings
  - `strict`: Enhanced security with restricted capabilities
  - `privileged`: Full system access (use with caution)

- **AppArmor Profiles**: Specify custom AppArmor profiles
  ```yaml
  security:
    apparmor_profile: lxc-container-default-restricted
  ```

- **SELinux Contexts**: Configure SELinux security contexts
  ```yaml
  security:
    selinux_context: system_u:system_r:container_t:s0
  ```

- **Linux Capabilities**: Fine-grained capability control
  ```yaml
  security:
    capabilities:
      - NET_ADMIN
      - SYS_TIME
  ```

## Development

### Prerequisites

- Go 1.23.4 or higher
- Access to Proxmox system
- Docker (for image conversion)

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.
