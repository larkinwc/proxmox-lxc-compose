#!/bin/sh
# install.sh - install the lxc-compose binary from GitHub Releases.
#
# Designed to run on a bare Proxmox VE node: needs `curl` (or `wget`), `tar`,
# `install`, and a SHA-256 tool. No Go toolchain required.
#
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/larkinwc/proxmox-lxc-compose/main/install.sh | sh
#
# Environment overrides:
#   VERSION     release tag to install (default: latest, e.g. v1.2.3)
#   INSTALL_DIR install destination   (default: /usr/local/bin)
#
set -eu

REPO="larkinwc/proxmox-lxc-compose"
BINARY="lxc-compose"
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"
VERSION="${VERSION:-latest}"

info() { printf '%s\n' "$*" >&2; }
err() { printf 'error: %s\n' "$*" >&2; exit 1; }

# --- pick a downloader -------------------------------------------------------
if command -v curl >/dev/null 2>&1; then
  http_get() { curl -fsSL "$1"; }
  http_dl() { curl -fsSL -o "$2" "$1"; }
elif command -v wget >/dev/null 2>&1; then
  http_get() { wget -qO- "$1"; }
  http_dl() { wget -qO "$2" "$1"; }
else
  err "neither curl nor wget found; please install one and retry"
fi

command -v tar >/dev/null 2>&1 || err "tar is required but was not found"
command -v install >/dev/null 2>&1 || err "install is required but was not found"
if command -v sha256sum >/dev/null 2>&1; then
  sha256() { sha256sum "$1"; }
elif command -v shasum >/dev/null 2>&1; then
  sha256() { shasum -a 256 "$1"; }
else
  err "a SHA-256 tool is required (sha256sum or shasum)"
fi
# --- detect OS/arch ----------------------------------------------------------
os="$(uname -s)"
case "$os" in
  Linux) os_name="Linux" ;;
  Darwin) os_name="Darwin" ;;
  *) err "unsupported OS: $os (use 'go install' instead)" ;;
esac

arch="$(uname -m)"
case "$arch" in
  x86_64 | amd64) arch_name="x86_64" ;;
  aarch64 | arm64) arch_name="arm64" ;;
  *) err "unsupported architecture: $arch (use 'go install' instead)" ;;
esac

# --- resolve version ---------------------------------------------------------
if [ "$VERSION" = "latest" ]; then
  info "Resolving latest release..."
  release="$(http_get "https://api.github.com/repos/${REPO}/releases/latest")" \
    || err "could not fetch latest release; set VERSION=vX.Y.Z"
  VERSION="$(printf '%s\n' "$release" \
    | sed -n 's/.*"tag_name":[[:space:]]*"\([^"]*\)".*/\1/p')"
  [ -n "$VERSION" ] || err "could not determine latest release tag; set VERSION=vX.Y.Z"
fi

# Archive name matches .goreleaser.yml: {ProjectName}_{Os}_{Arch}.tar.gz
archive="${BINARY}_${os_name}_${arch_name}.tar.gz"
base_url="https://github.com/${REPO}/releases/download/${VERSION}"

info "Installing ${BINARY} ${VERSION} (${os_name}/${arch_name})..."

tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' 0
trap 'exit 1' HUP INT TERM

http_dl "${base_url}/${archive}" "${tmp}/${archive}" \
  || err "failed to download ${base_url}/${archive}"

# --- verify checksum before extracting or installing -------------------------
http_dl "${base_url}/checksums.txt" "${tmp}/checksums.txt" \
  || err "failed to download checksums.txt"
expected="$(awk -v name="$archive" '$2 == name { print $1 }' "${tmp}/checksums.txt")"
[ "${#expected}" = 64 ] || err "missing or invalid checksum for ${archive}"
case "$expected" in
  *[!0-9a-fA-F]*) err "invalid checksum for ${archive}" ;;
esac
actual="$(sha256 "${tmp}/${archive}")"
actual="${actual%% *}"
[ "$expected" = "$actual" ] || err "checksum mismatch for ${archive}"
info "Checksum verified."

tar -xzf "${tmp}/${archive}" -C "$tmp" "$BINARY"
[ -f "${tmp}/${BINARY}" ] && [ ! -L "${tmp}/${BINARY}" ] \
  || err "archive did not contain a regular ${BINARY} binary"
chmod +x "${tmp}/${BINARY}"
"${tmp}/${BINARY}" version

# --- install -----------------------------------------------------------------
if mkdir -p "$INSTALL_DIR" 2>/dev/null && [ -w "$INSTALL_DIR" ]; then
  install -m 0755 "${tmp}/${BINARY}" "${INSTALL_DIR}/${BINARY}"
elif command -v sudo >/dev/null 2>&1; then
  info "Elevating with sudo to write to ${INSTALL_DIR}..."
  sudo mkdir -p "$INSTALL_DIR"
  sudo install -m 0755 "${tmp}/${BINARY}" "${INSTALL_DIR}/${BINARY}"
else
  err "cannot write to ${INSTALL_DIR}; re-run as root or set INSTALL_DIR=\$HOME/bin"
fi

info "Installed to ${INSTALL_DIR}/${BINARY}"
"${INSTALL_DIR}/${BINARY}" version
