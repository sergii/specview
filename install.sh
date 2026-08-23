#!/bin/sh
set -eu

REPO="sergii/specview"
BINARY="specview"

fail() {
  printf 'specview installer: %s\n' "$*" >&2
  exit 1
}

command -v curl >/dev/null 2>&1 || fail "curl is required"
command -v tar >/dev/null 2>&1 || fail "tar is required"

os=$(uname -s | tr '[:upper:]' '[:lower:]')
case "$os" in
  darwin|linux) ;;
  *) fail "unsupported operating system: $os" ;;
esac

arch=$(uname -m)
case "$arch" in
  x86_64|amd64) arch="amd64" ;;
  arm64|aarch64) arch="arm64" ;;
  *) fail "unsupported architecture: $arch" ;;
esac

asset="specview_${os}_${arch}.tar.gz"
base="https://github.com/${REPO}/releases/latest/download"
tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT HUP INT TERM

printf 'Installing Specview for %s/%s...\n' "$os" "$arch"
curl -fsSL "${base}/${asset}" -o "${tmp}/${asset}"
curl -fsSL "${base}/SHA256SUMS" -o "${tmp}/SHA256SUMS"

expected=$(awk -v file="$asset" '$2 == file { print $1 }' "${tmp}/SHA256SUMS")
[ -n "$expected" ] || fail "checksum for ${asset} not found"

if command -v sha256sum >/dev/null 2>&1; then
  actual=$(sha256sum "${tmp}/${asset}" | awk '{print $1}')
elif command -v shasum >/dev/null 2>&1; then
  actual=$(shasum -a 256 "${tmp}/${asset}" | awk '{print $1}')
else
  fail "sha256sum or shasum is required"
fi

[ "$actual" = "$expected" ] || fail "checksum verification failed"

tar -xzf "${tmp}/${asset}" -C "$tmp"

install_dir=${SPECVIEW_INSTALL_DIR:-"$HOME/.local/bin"}
mkdir -p "$install_dir"
install -m 0755 "${tmp}/${BINARY}" "${install_dir}/${BINARY}"

printf '✓ Installed Specview to %s/%s\n' "$install_dir" "$BINARY"
case ":$PATH:" in
  *":$install_dir:"*) ;;
  *)
    printf '\nAdd this directory to PATH:\n  export PATH="%s:$PATH"\n' "$install_dir"
    ;;
esac
printf '\nRun:\n  specview init\n  specview\n'
