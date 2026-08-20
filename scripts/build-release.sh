#!/bin/sh
set -eu

version=${1:-dev}
rm -rf dist
mkdir -p dist

for target in linux/amd64 linux/arm64 darwin/amd64 darwin/arm64; do
  os=${target%/*}
  arch=${target#*/}
  name="specview_${os}_${arch}"
  stage=$(mktemp -d)

  printf 'Building %s/%s\n' "$os" "$arch"
  CGO_ENABLED=0 GOOS="$os" GOARCH="$arch" go build \
    -trimpath \
    -ldflags "-s -w -X main.version=${version}" \
    -o "${stage}/specview" \
    ./cmd/specview

  tar -C "$stage" -czf "dist/${name}.tar.gz" specview
  rm -rf "$stage"
done

(
  cd dist
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum specview_*.tar.gz > SHA256SUMS
  else
    shasum -a 256 specview_*.tar.gz > SHA256SUMS
  fi
)
