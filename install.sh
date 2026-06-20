#!/bin/sh
# ---------------------------------------------------------------------------
# Oriel installer — downloads the right release binary for your platform,
# VERIFIES it against the published SHA256SUMS, and installs it onto your PATH.
#
# Running root-equivalent software from a piped script? Read it first. Or skip
# this and use the explicit per-platform commands in the README — they do
# exactly what this does, one step at a time.
#
# Run interactively and it asks where to install and whether to set up the
# background service. For unattended installs, set these and it won't prompt:
#     ORIEL_INSTALL_DIR=/path   install location (default: /usr/local/bin or ~/.local/bin)
#     ORIEL_SERVICE=1           also install + start the background service (0 to skip)
# ---------------------------------------------------------------------------
set -eu

REPO="ParadoxInfinite/oriel"
BASE="https://github.com/$REPO/releases/latest/download"

die() { echo "oriel-install: $*" >&2; exit 1; }

# Prompts read the terminal directly, so they work even under `curl … | sh`.
# With no terminal (CI, automation), we fall back to env vars / defaults.
if [ -e /dev/tty ]; then INTERACTIVE=1; else INTERACTIVE=0; fi

ask() { # ask PROMPT DEFAULT  → chosen value (default if blank / non-interactive)
  _ans=""
  if [ "$INTERACTIVE" = 1 ]; then
    printf "%s" "$1" > /dev/tty
    read _ans < /dev/tty || _ans=""
  fi
  [ -n "$_ans" ] && printf "%s" "$_ans" || printf "%s" "$2"
}

confirm() { # confirm PROMPT  → 0 if yes
  [ "$INTERACTIVE" = 1 ] || return 1
  printf "%s" "$1" > /dev/tty
  read _c < /dev/tty || _c=""
  case "$_c" in y | Y | yes | YES) return 0 ;; *) return 1 ;; esac
}

# --- detect platform -------------------------------------------------------
os=$(uname -s)
case "$os" in
  Darwin) os=darwin ;;
  Linux) os=linux ;;
  *) die "unsupported OS: $os (macOS and Linux only)" ;;
esac
arch=$(uname -m)
case "$arch" in
  x86_64 | amd64) arch=amd64 ;;
  arm64 | aarch64) arch=arm64 ;;
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

# --- install location (env var > prompt > default) -------------------------
default_dir="$HOME/.local/bin"
if [ -w /usr/local/bin ]; then default_dir=/usr/local/bin; fi
dir="${ORIEL_INSTALL_DIR:-}"
[ -n "$dir" ] || dir=$(ask "Install location [$default_dir]: " "$default_dir")

mkdir -p "$dir"
chmod +x "$tmp/oriel"
mv "$tmp/oriel" "$dir/oriel"
echo "Installed oriel to $dir/oriel"

case ":$PATH:" in
  *":$dir:"*) ;;
  *) echo "Note: $dir is not on your PATH — add it, or run $dir/oriel directly." ;;
esac

# --- background service (env var > prompt) ---------------------------------
if [ -n "${ORIEL_SERVICE:-}" ]; then
  [ "$ORIEL_SERVICE" = 1 ] && want_service=1 || want_service=0
elif confirm "Start Oriel on login as a background service? [y/N]: "; then
  want_service=1
else
  want_service=0
fi

if [ "$want_service" = 1 ]; then
  "$dir/oriel" service install
else
  echo
  echo "Run it:       $dir/oriel"
  echo "Or on login:  $dir/oriel service install"
fi
