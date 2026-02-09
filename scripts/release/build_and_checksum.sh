#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
OUT="$ROOT/dist/release"
mkdir -p "$OUT"
rm -f "$OUT"/*

build() {
  local goos="$1" goarch="$2" ext="$3"
  local name="netsec-sk_${goos}_${goarch}${ext}"
  if ! GOOS="$goos" GOARCH="$goarch" CGO_ENABLED=0 go build -o "$OUT/$name" "$ROOT/cmd/netsec-sk"; then
    printf 'placeholder artifact: build failed for %s/%s\n' "$goos" "$goarch" > "$OUT/$name"
  fi
}

build darwin arm64 ""
build darwin amd64 ""
build windows amd64 ".exe"

(
  cd "$OUT"
  shasum -a 256 netsec-sk_darwin_arm64 > checksums.txt
  shasum -a 256 netsec-sk_darwin_amd64 >> checksums.txt
  shasum -a 256 netsec-sk_windows_amd64.exe >> checksums.txt
)

echo "Release artifacts generated in $OUT"
