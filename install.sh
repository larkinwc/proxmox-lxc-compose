#!/bin/sh
# install.sh - install the lxc-compose binary from GitHub Releases.
#
# Designed to run on a bare Proxmox VE node: needs only `curl` (or `wget`) and
# `tar`. No Go toolchain required.
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
  VERSION="$(http_get "https://api.github.com/repos/${REPO}/releases/latest" \
    | sed -n 's/.*"tag_name":[[:space:]]*"\([^"]*\)".*/\1/p' | head -n1)"
  [ -n "$VERSION" ] || err "could not determine latest release tag; set VERSION=vX.Y.Z"
fi

# Archive name matches .goreleaser.yml: {ProjectName}_{Os}_{Arch}.tar.gz
archive="${BINARY}_${os_name}_${arch_name}.tar.gz"
base_url="https://github.com/${REPO}/releases/download/${VERSION}"

info "Installing ${BINARY} ${VERSION} (${os_name}/${arch_name})..."

tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT

http_dl "${base_url}/${archive}" "${tmp}/${archive}" \
  || err "failed to download ${base_url}/${archive}"

# --- verify checksum if available -------------------------------------------
if http_dl "${base_url}/checksums.txt" "${tmp}/checksums.txt" 2>/dev/null; then
  if command -v sha256sum >/dev/null 2>&1; then
    expected="$(grep " ${archive}\$" "${tmp}/checksums.txt" | awk '{print $1}')"
    if [ -n "$expected" ]; then
      actual="$(sha256sum "${tmp}/${archive}" | awk '{print $1}')"
      [ "$expected" = "$actual" ] || err "checksum mismatch for ${archive}"
      info "Checksum verified."
    fi
  fi
fi

tar -xzf "${tmp}/${archive}" -C "$tmp"
[ -f "${tmp}/${BINARY}" ] || err "archive did not contain ${BINARY}"
chmod +x "${tmp}/${BINARY}"

# --- install -----------------------------------------------------------------
if [ -w "$INSTALL_DIR" ] || [ "$(id -u)" = "0" ]; then
  mv "${tmp}/${BINARY}" "${INSTALL_DIR}/${BINARY}"
elif command -v sudo >/dev/null 2>&1; then
  info "Elevating with sudo to write to ${INSTALL_DIR}..."
  sudo mv "${tmp}/${BINARY}" "${INSTALL_DIR}/${BINARY}"
else
  err "cannot write to ${INSTALL_DIR}; re-run as root or set INSTALL_DIR=\$HOME/bin"
fi

info "Installed to ${INSTALL_DIR}/${BINARY}"
"${INSTALL_DIR}/${BINARY}" version 2>/dev/null || info "Run '${BINARY} version' to verify."
