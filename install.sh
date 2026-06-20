#!/bin/sh
# ---------------------------------------------------------------------------
# Oriel installer — downloads the right release binary for your platform,
# VERIFIES it against the published SHA256SUMS, and installs it onto your PATH.
#
# Running a piped script as root-equivalent software? Read it first. Or skip
# this entirely and use the explicit per-platform commands in the README —
# they do exactly what this does, one step at a time.
#
#   Options (env vars):
#     ORIEL_INSTALL_DIR=/path   install location (default: /usr/local/bin or ~/.local/bin)
#     ORIEL_SERVICE=1           also install + start the background service
# ---------------------------------------------------------------------------
set -eu

REPO="ParadoxInfinite/oriel"
BASE="https://github.com/$REPO/releases/latest/download"

die() { echo "oriel-install: $*" >&2; exit 1; }

# --- detect platform -------------------------------------------------------
os=$(uname -s)
case "$os" in
  Darwin) os=darwin ;;
  Linux)  os=linux ;;
  *) die "unsupported OS: $os (macOS and Linux only)" ;;
esac
arch=$(uname -m)
case "$arch" in
  x86_64|amd64)  arch=amd64 ;;
  arm64|aarch64) arch=arm64 ;;
  *) die "unsupported architecture: $arch" ;;
esac
asset="oriel-${os}-${arch}"

# --- pick a downloader -----------------------------------------------------
if command -v curl >/dev/null 2>&1; then
  fetch() { curl -fSL "$1" -o "$2"; }
elif command -v wget >/dev/null 2>&1; then
  fetch() { wget -qO "$2" "$1"; }
else
  die "need curl or wget"
fi

# --- download + verify -----------------------------------------------------
tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT

echo "Downloading $asset…"
fetch "$BASE/$asset" "$tmp/oriel" || die "download failed"
fetch "$BASE/SHA256SUMS.txt" "$tmp/sums" || die "could not fetch checksums"

expected=$(grep " ${asset}\$" "$tmp/sums" | awk '{print $1}')
[ -n "$expected" ] || die "no checksum for $asset in SHA256SUMS.txt"

if command -v sha256sum >/dev/null 2>&1; then
  actual=$(sha256sum "$tmp/oriel" | awk '{print $1}')
else
  actual=$(shasum -a 256 "$tmp/oriel" | awk '{print $1}')
fi
[ "$expected" = "$actual" ] || die "checksum mismatch — refusing to install (expected $expected, got $actual)"
echo "Checksum verified."

# --- install ---------------------------------------------------------------
dir="${ORIEL_INSTALL_DIR:-}"
if [ -z "$dir" ]; then
  if [ -w /usr/local/bin ] 2>/dev/null; then dir=/usr/local/bin; else dir="$HOME/.local/bin"; fi
fi
mkdir -p "$dir"
chmod +x "$tmp/oriel"
mv "$tmp/oriel" "$dir/oriel"
echo "Installed oriel to $dir/oriel"

case ":$PATH:" in
  *":$dir:"*) ;;
  *) echo "Note: $dir is not on your PATH — add it, or run $dir/oriel directly." ;;
esac

# --- optional: background service ------------------------------------------
if [ "${ORIEL_SERVICE:-}" = "1" ]; then
  "$dir/oriel" service install
else
  echo
  echo "Run it:            $dir/oriel"
  echo "Or on login:       $dir/oriel service install"
fi
