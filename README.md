# proxmox-lxc-compose

A docker-compose like tool for managing LXC containers in Proxmox.

## Project Overview
`lxc-compose` reads an `lxc-compose.yml` file and uses it to create, start,
stop, and inspect **Proxmox** LXC containers via the `pct` CLI. Services map to
auto-allocated VMIDs, and OCI images are converted to LXC templates on demand.

## Features

- **Docker-compose style YAML** describing one or more LXC services.
- **`up` / `down` / `ps` / `pause` / `unpause`** lifecycle commands driven
  through the Proxmox `pct` CLI.
- **Automatic VMID allocation** with a persisted service→VMID mapping.
- **OCI image support**: any `image:` that isn't a Proxmox template volid is
  auto-converted to an LXC template on `up` (cached, refreshable). Image process
  defaults can be overridden per service without changing the shared image cache.
  Template services also support command, entrypoint, and environment overrides.
- **Resource / storage / network translation**: CPU, memory, rootfs, extra
  mounts, and bridged networking (DHCP or static) map to `pct` options.
- **OCI registry helpers** (`images pull/push/list/remove`) with local caching,
  and a standalone `convert` command.

> **Scope note.** This tool targets **Proxmox** LXC (via `pct`), so it must run
> on a Proxmox node. Some compose fields are accepted by the parser but **not**
> yet applied to Proxmox containers — see
> [Compose → Proxmox field mapping](#compose--proxmox-field-mapping) and
> [Limitations](#limitations).

## Prerequisites

- A Proxmox VE node (the `up`/`down`/`ps`/`pause`/`unpause` commands must run
  **on** the node as root)
- Docker only when converting OCI images; existing Proxmox templates do not
  require Docker
- Go 1.23 or later only when building from source, not for binary installation

## Proxmox Integration

This tool manages **Proxmox** LXC containers, not upstream LXC. Container
lifecycle operations are executed through a pluggable backend; the default
backend shells out to the Proxmox `pct` CLI (a REST API backend can be added
behind the same `proxmox.Backend` interface).

### VMID mapping

Proxmox addresses containers by a numeric **VMID**, while compose files name
services. On `up`, each service is auto-allocated the next free VMID (starting
at 100) and the `name -> vmid` mapping is persisted to
`~/.lxc-compose/vmids.json`. Subsequent commands (`down`, `pause`, `ps`, ...)
reuse that mapping so a service always maps to the same container.

### Repeatable lifecycle and recreation

`up` records the successfully applied service configuration and effective node
defaults in `~/.lxc-compose/vmids.json.state.json`. Repeating it leaves a matching
running container unchanged, starts a matching stopped container, and leaves a
paused container paused. If the mapped container is missing, it is provisioned
again at the same VMID.

Changing a service or an effective node default requires explicit recreation:

```bash
lxc-compose up --recreate app
lxc-compose up --pull --recreate app
```

Recreation stops and destroys the selected container, including its managed
disks, then provisions its replacement at the same VMID. Back up data first.
Selected configurations and template availability are checked before stopping
existing containers; a hostname or VMID ownership mismatch is rejected.
Mappings created by older versions have no recorded desired configuration and
also require `--recreate`; they are never silently adopted. `down --rm` removes
both the mapping and desired-state record.

The Proxmox smoke covered legacy mapping refusal, repeated `up`, starting a
stopped container, refusing changed memory without recreation, explicit
replacement, refresh flag refusal, recovery of an externally removed container,
and teardown.

### Node defaults

Some Proxmox concepts have no compose-file equivalent and are supplied via
environment variables:

| Variable             | Default     | Purpose                                            |
|----------------------|-------------|----------------------------------------------------|
| `PROXMOX_STORAGE`    | `local-lvm` | Storage ID for the container rootfs                |
| `PROXMOX_BRIDGE`     | `vmbr0`     | Default bridge for interfaces without one          |
| `PROXMOX_ROOTFS_GB`  | `8`         | Default rootfs size (GB) when `storage.root` unset |
| `PROXMOX_VMID_BASE`  | `100`       | Lowest VMID auto-allocation will assign            |

The template to provision from is taken directly from each service's
`image:` field (see [Auto-conversion via `up`](#auto-conversion-via-up)).

### Compose → Proxmox field mapping

These fields are translated to `pct` options on `up`:

| Compose field            | Proxmox (`pct`) option            |
|--------------------------|-----------------------------------|
| `image`                  | OS template (volid, or converted) |
| `cpu.cores`              | `--cores`                         |
| `cpu.shares`             | `--cpuunits`                      |
| `cpu.quota` / `period`   | `--cpulimit` (cores)              |
| `memory.limit`           | `--memory` (MB)                   |
| `memory.swap`            | `--swap` (MB)                     |
| `storage.pool`           | rootfs storage ID                 |
| `storage.root`           | rootfs size (GB, rounded up)      |
| `storage.mounts[]`       | `--mpN`                           |
| `network.interfaces[]` / legacy `network.*` | `--netN name=,bridge=,ip=,gw=` |
| `network.dns`            | `--nameserver`                    |
| `security.isolation` / `security.privileged` | `--unprivileged` (privileged → 0) |

If a service declares no network, it gets a DHCP `eth0` on the default bridge
(like docker-compose).

### Command, entrypoint, and environment

`command` and `entrypoint` are argument arrays; no implicit shell is added.
For OCI images, omitted arrays inherit image defaults, while `[]` explicitly
clears that array. An explicit `entrypoint` also clears the image's default
command unless `command` is supplied. Image environment and working directory
are retained.

Environment precedence is **image → `env` → `environment`**. An empty value
overrides the image value; an empty map does not clear inherited variables.
Values are quoted literally, including spaces, quotes, shell metacharacters,
and newlines.

```yaml
services:
  web:
    image: nginx:alpine
    command: ["nginx", "-g", "daemon off;"]
    environment:
      NGINX_ENTRYPOINT_QUIET_LOGS: "1"
```

For native templates, a command/entrypoint override replaces distro init as
PID 1. With environment-only overrides, the wrapper still executes `/sbin/init`.
Systemd-based templates also receive a `DefaultEnvironment` drop-in so the
variables reach actual system services; inspecting `/proc/1/environ` is not a
reliable check because systemd sanitizes its own environment.

Overrides produce separate content-addressed templates; the original template
and shared OCI image archive are not modified. Cached archives and JSON metadata
contain environment values and are owner-only. Generated guest runtime files
are also owner-only. Existing metadata-less caches are rebuilt rather than
guessing an init command. Template derivation uses `tar`, plus `xz` or `zstd`
for those source compression formats.

The wrapper brings up loopback and the configured interface names, and honors
static addresses/gateways. It requests DHCP only for DHCP interfaces; without
network configuration, it uses DHCP `eth0` on the default bridge.

Changing runtime overrides is configuration drift: use `up --recreate` to apply
them to an existing container, with the same disk-destruction warning described
above.


### Limitations

The parser accepts these fields, but `up` does **not** yet apply them to
Proxmox containers:

- `ports` / `network.port_forwards` (no port forwarding), `devices`.
- `security.apparmor_profile`, `selinux_context`, `seccomp_profile`, and
  individual Linux `capabilities`.

Other notes:

- Must run on the Proxmox node (the `pct` backend is local-only).
- Real-node behavior is covered by integration tests gated behind the
  `integration` build tag and `PROXMOX_INTEGRATION=1` (see below).
- `pause` / `unpause` freeze and thaw guest processes in memory using
  `lxc-freeze` / `lxc-unfreeze`; they do not use CRIU or save checkpoints.
  `ps` inspects `lxc-info` to distinguish frozen guests from running ones.
  These LXC utilities must be available on the Proxmox node.

## Installation

`lxc-compose` runs **on the Proxmox node**. A stock Proxmox VE node has no Go
toolchain, so the recommended path is the prebuilt binary.

### Quick install (recommended)

Downloads the latest release for your OS/arch, requires a matching SHA-256
checksum, and installs to `/usr/local/bin`. Requires `curl` or `wget`, `tar`,
`install`, and `sha256sum` or `shasum` (available on a stock Proxmox node).

```bash
curl -fsSL https://raw.githubusercontent.com/larkinwc/proxmox-lxc-compose/main/install.sh | sh
```

Pin a version or change the destination:

```bash
curl -fsSL https://raw.githubusercontent.com/larkinwc/proxmox-lxc-compose/main/install.sh \
  | VERSION=v1.2.3 INSTALL_DIR="$HOME/bin" sh
```

### Manual download

Grab the archive for your platform from the
[Releases page](https://github.com/larkinwc/proxmox-lxc-compose/releases),
then:

```bash
tar -xzf lxc-compose_Linux_x86_64.tar.gz
sudo install -m 0755 lxc-compose /usr/local/bin/
```

### From source

Requires Go 1.23+ (handy on a dev box; not typically present on a Proxmox node).

```bash
# via go install
go install github.com/larkinwc/proxmox-lxc-compose/cmd/lxc-compose@latest

# or clone and build (embeds version metadata, installs to /usr/local/bin)
git clone https://github.com/larkinwc/proxmox-lxc-compose.git
cd proxmox-lxc-compose
sudo make install          # or: make build && sudo mv lxc-compose /usr/local/bin/
```

### Verify

```bash
lxc-compose version
```

> **Release cutover:** older releases through v1.0.6 contain a
> `proxmox-lxc-compose` binary and differently named archives/checksums. This
> installer targets the new `lxc-compose` release layout and requires a new
> release published with the configuration in this checkout. It cannot install
> v1.0.6; do not use the quick-install command until the new release is published.

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

# Refresh OCI images and explicitly replace existing containers
lxc-compose up --pull --recreate

# OCI image registry helpers
lxc-compose images pull [registry/repository:tag]
lxc-compose images list

# Convert an OCI image to an LXC template (standalone)
lxc-compose convert [image] -o templates/out.tar.gz

# Manage templates
lxc-compose template create [container] [template] -d "description"
lxc-compose template ls
lxc-compose template apply [template] [new_container]
lxc-compose template rm [template]
```

### Converting OCI Images to LXC Templates

```bash
lxc-compose convert nginx:alpine -o templates/nginx.tar.gz
```

This pulls the image with Docker, flattens its filesystem, applies the
OCI→LXC compatibility fixes described below, and writes a gzip-compressed
LXC template that Proxmox (`pct` / `pveam`) can provision from.

#### What the converter does

OCI images and LXC containers differ in several ways that prevent a raw
`docker export` from "just working" as an LXC template. The converter
bridges these gaps automatically:

1. **Real gzip output.** `docker export` produces an *uncompressed* tar.
   Proxmox expects a compressed archive, so the converter repacks the
   rootfs as a genuine `.tar.gz` (the output name is normalized to end in
   `.tar.gz`).
2. **Log symlinks.** Many images symlink `/var/log/.../*.log` to
   `/dev/stdout` and `/dev/stderr`. In an *unprivileged* LXC container
   those targets aren't writable the way they are under Docker, which
   makes daemons like nginx abort with "Permission denied". The converter
   replaces these symlinks with regular files.
3. **Entrypoint / command.** A Docker image's `CMD`/`ENTRYPOINT` is
   metadata, not a service, so a normal LXC init never starts it. The
   converter captures the image's entrypoint+command via `docker inspect`
   and generates a small init wrapper
   (`/usr/local/bin/lxc-compose-init.sh`) that reproduces it under LXC.
4. **Networking.** Docker configures networking externally. The wrapper
   waits for `eth0` to be plumbed into the container's network namespace,
   brings the link up, and runs a DHCP client before exec'ing the image
   command. A `/etc/network/interfaces` file is also written as a
   documented fallback for images booted with their own init.

#### Auto-conversion via `up`

`lxc-compose up` resolves each service's `image:` automatically — no flags
or env vars required:

- An image that is already a **Proxmox template volid** (it contains a
  `:vztmpl/` segment, e.g. `local:vztmpl/alpine-3.22.tar.xz`) is used as-is.
- Any other reference is treated as an **OCI image**, converted, imported
  into the local template cache, and used to create the container. The
  captured entrypoint/command is applied via the container's
  `lxc.init.cmd`.

```yaml
services:
  web:
    image: nginx:alpine          # OCI image -> auto-converted
  base:
    image: local:vztmpl/alpine-3.22.tar.xz   # template volid; no derivation without overrides
```

```bash
lxc-compose up                  # converts nginx:alpine -> template -> running CT
```

Conversion is **cached**: an already-converted image is reused on
subsequent `up` runs. To refresh against an updated upstream image:

```bash
lxc-compose up --pull --recreate # existing containers require explicit recreation
```

Like docker-compose, every service receives a DHCP `eth0` on the default
bridge unless the compose file specifies its own network configuration.

> **Init mechanism & fallback.** The primary mechanism is the LXC init
> command (`lxc.init.cmd`) pointing at the generated wrapper. Because the
> wrapper becomes PID 1 (rather than a full init), `lxc-compose down`
> shuts containers down with `pct shutdown --forceStop` so they always
> stop cleanly. For images you'd rather run under a real init, install the
> command as a normal service (e.g. an OpenRC/systemd unit) inside the
> image and omit the init wrapper.

> **Note:** conversion requires Docker to be installed on the machine
> running `lxc-compose` (the Proxmox node), and currently targets
> Alpine/Debian-family images.

### Configuration File (lxc-compose.yml)

Two runnable examples live in [`examples/`](examples/):

- [`examples/nginx-oci.yml`](examples/nginx-oci.yml) — run an OCI image
  (`nginx:alpine`) directly; `up` auto-converts and starts it. Validated
  end-to-end (DHCP IP → HTTP 200 → teardown).
- [`examples/alpine-template.yml`](examples/alpine-template.yml) — provision
  from an existing Proxmox template volid with static IP and an extra mount.

Minimal OCI service:

```yaml
version: "1.0"
services:
  web:
    image: nginx:alpine     # not a vztmpl volid -> auto-converted on `up`
    cpu:
      cores: 1
    memory:
      limit: 256M
    storage:
      root: 2G
    # No network block: gets a DHCP eth0 on the default bridge.
```

Provision from a template volid with static networking:

```yaml
version: "1.0"
services:
  app:
    image: local:vztmpl/alpine-3.19-default_20240207_amd64.tar.xz
    cpu:
      cores: 2
    memory:
      limit: 512M
      swap: 256M
    storage:
      pool: local-lvm
      root: 4G
    network:
      interfaces:
        - type: bridge
          bridge: vmbr0
          ip: 192.168.1.50/24
          gateway: 192.168.1.1
          dns: [1.1.1.1]
    security:
      isolation: strict     # -> unprivileged
```

> Only the fields in
> [Compose → Proxmox field mapping](#compose--proxmox-field-mapping) take
> effect. Fields like `command`, `environment`, `ports`, and the AppArmor/
> SELinux/capability security knobs are parsed but not yet applied to Proxmox
> containers (see [Limitations](#limitations)).

## Development

### Prerequisites

- Go 1.23 or higher (including the Go 1.23.0 toolchain)
- Access to Proxmox system
- Docker (for image conversion)

### Testing

Unit tests run anywhere (Proxmox commands are mocked):

```bash
make test
```

Integration tests exercise the real `pct` CLI and only run on a Proxmox node:

```bash
PROXMOX_INTEGRATION=1 \
  PROXMOX_TEST_TEMPLATE="local:vztmpl/alpine-3.19-default_20240207_amd64.tar.xz" \
  PROXMOX_TEST_STORAGE=local-lvm PROXMOX_TEST_VMID=999 \
  go test -tags integration ./pkg/proxmox/ -run Integration -v
```

To exercise native systemd service environment propagation, additionally set
`PROXMOX_TEST_SYSTEMD_TEMPLATE` to a Debian/systemd template volid. This test
refuses to replace an existing VMID and cleans up its derived template and guest.
`PROXMOX_TEST_VMID` must always identify a dedicated disposable guest.


### Binary releases

GoReleaser v2 builds static, CGO-free binaries for Linux and macOS
(amd64/arm64) and Windows (amd64). Archives contain `lxc-compose` (or
`lxc-compose.exe`), README, and LICENSE; `checksums.txt` lists SHA-256 hashes.
The binary's `version` command includes the release version, commit, and date.

Validate and build locally without publishing:

```bash
make release-check
make release-dry-run
```

Artifacts are written to `dist/`. CI also builds snapshot artifacts for pushes
and pull requests. The tag-triggered release workflow runs tests, then publishes
with the repository's built-in `GITHUB_TOKEN` (`contents: write`); no custom
`RELEASE_TOKEN` secret is required. Both workflows use the Go version in `go.mod`.

To publish, commit and push the desired changes, then push a new `v*` tag.
Check the [existing releases](https://github.com/larkinwc/proxmox-lxc-compose/releases)
first: the cutover release must be newer than v1.0.6. The `release-major`,
`release-minor`, and `release-patch` Make targets push tags immediately and
calculate from local tags, so do not use them with a stale local tag set.

### Real-node smoke verification

The Linux amd64 snapshot and installer were exercised on Proxmox VE 9.2.10
using staged release artifacts (not a published GitHub release). Installer
checks rejected missing checksums, missing archive entries in the checksum
manifest, checksum mismatches, and corrupt archives without replacing the
existing binary. Installing into a new user-writable directory does not
require sudo.

A disposable Debian 13 container passed `up`, `ps`, guest execution, and
`down --rm`; its configuration matched 1 core, 256 MB RAM, 128 MB swap, a
2 GB `local-lvm` rootfs, `vmbr0`, and unprivileged isolation. Teardown removed
the container and its VMID mapping. `TestIntegrationContainerLifecycle` passed
on the node using the Go 1.23.0 compiled test binary, including freezer
transitions: a guest counter stopped changing while paused and resumed afterward,
with the same guest init PID.

Further live-node checks covered repeated `up` without replacement, starting a
stopped container, recreating a missing mapped container at the same VMID,
sorted drift-field reporting, explicit recreation, and refusal to destroy a
running container when its replacement template is unavailable.

Native-template command overrides preserved literal environment values and
configured a named static interface. CLI `pause`/`ps`/`up`/`unpause` preserved
frozen state until explicit resume. A real systemd unit inherited environment
values containing quotes, percent specifiers, backslashes, tabs, and newlines;
`TestIntegrationSystemdEnvironment` passed on a disposable Debian 13 container.

OCI nginx conversion was then exercised using an isolated Docker daemon. A
service command override captured its inherited environment before starting
nginx, proving image defaults and canonical environment precedence without
relying on nginx's overwritten process-title/environment memory. Guest
loopback HTTP returned 200, repeated `up` left the guest unchanged, generated
cache files were owner-only, and `down --rm` removed the guest and mapping.

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.
