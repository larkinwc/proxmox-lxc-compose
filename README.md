# lxc-compose

`lxc-compose` is a CLI tool that allows you to manage LXC containers using a `docker-compose` like syntax.

## Features

- Define LXC containers using a `docker-compose` like syntax
- Start, stop, and manage LXC containers per service in the `lxc-compose.yml` file
- Convert OCI images to LXC templates
- Retrieve OCI images from a registry for use as template
- Push LXC templates to a registry as OCI images

## Commands

### `lxc-compose up`

### `lxc-compose down`
 
### `lxc-compose pull <image>`

### `lxc-compose push <image>`

## Usage

```bash 
lxc-compose up
```

```bash
lxc-compose down
```

```bash
lxc-compose pull <image>
```

```bash
lxc-compose push <image>
```
