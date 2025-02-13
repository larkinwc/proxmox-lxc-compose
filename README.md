# proxmox-lxc-compose

A CLI tool that allows you to manage LXC containers using a docker-compose like syntax.

## Features

- Define LXC containers using a familiar docker-compose like syntax
- Convert OCI images to LXC templates
- Manage container lifecycle (create, start, stop, update)
- Configure container resources (CPU, memory, storage)
- Set up networking and storage
- Define environment variables and entry points

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
