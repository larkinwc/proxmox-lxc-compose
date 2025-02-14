# proxmox-lxc-compose

A CLI tool that allows you to manage LXC containers using a docker-compose like syntax.

## Project Overview
The main purpose of the CLI is to read a lxc-compose.yml file and use it to create, start, stop, and update LXC containers. The tool validates YAML configuration files and ensures they contain the required minimum set of keys.

## Features

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
- Testing
  - Unit tests for state management
  - Unit tests for configuration parsing
  - Unit tests for container logging
  - Unit tests for container management (pause/resume)

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
- Advanced validation
  - Network configuration validation
  - Storage configuration validation
  - Resource limits validation
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

## Usage

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
```

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.
