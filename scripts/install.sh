#!/usr/bin/env bash
# Install the vc binary from GitHub Releases.
set -euo pipefail

REPO="${VC_INSTALL_REPO:-vector-charts/vector-charts-cli}"
VERSION="${VC_INSTALL_VERSION:-latest}"
INSTALL_DIR="${VC_INSTALL_DIR:-/usr/local/bin}"

need_cmd() {
  command -v "$1" >/dev/null 2>&1 || {
    echo "error: required command not found: $1" >&2
    exit 1
  }
}

need_cmd curl
need_cmd tar
need_cmd uname

os="$(uname -s | tr '[:upper:]' '[:lower:]')"
arch="$(uname -m)"
case "$arch" in
  x86_64|amd64) arch="amd64" ;;
  aarch64|arm64) arch="arm64" ;;
  *)
    echo "error: unsupported architecture: $arch" >&2
    exit 1
    ;;
esac

case "$os" in
  linux|darwin) ;;
  *)
    echo "error: unsupported OS: $os" >&2
    exit 1
    ;;
esac

if [[ "$VERSION" == "latest" ]]; then
  tag="$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" | sed -n 's/.*"tag_name": *"\([^"]*\)".*/\1/p' | head -n1)"
  if [[ -z "$tag" ]]; then
    echo "error: could not resolve latest release for ${REPO}" >&2
    exit 1
  fi
else
  tag="$VERSION"
fi

# Strip leading v for archive name if present in asset naming.
ver="${tag#v}"
asset="vc_${ver}_${os}_${arch}.tar.gz"
url="https://github.com/${REPO}/releases/download/${tag}/${asset}"

tmpdir="$(mktemp -d)"
trap 'rm -rf "$tmpdir"' EXIT

echo "Downloading ${url}"
curl -fsSL "$url" -o "${tmpdir}/${asset}"
tar -xzf "${tmpdir}/${asset}" -C "$tmpdir"

if [[ ! -f "${tmpdir}/vc" ]]; then
  echo "error: archive did not contain vc binary" >&2
  exit 1
fi

dest="${INSTALL_DIR}/vc"
if [[ ! -w "$INSTALL_DIR" ]]; then
  echo "Installing to ${dest} (sudo)"
  sudo install -m 755 "${tmpdir}/vc" "$dest"
else
  echo "Installing to ${dest}"
  install -m 755 "${tmpdir}/vc" "$dest"
fi

echo "Installed $($dest version)"
