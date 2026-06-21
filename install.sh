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
#     ORIEL_BASE_PATH=/oriel    serve behind a reverse proxy under this sub-path
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

echo "Downloading ${asset}…"
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
if [ -d /usr/local/bin ] && [ -w /usr/local/bin ]; then default_dir=/usr/local/bin; fi
dir="${ORIEL_INSTALL_DIR:-}"
[ -n "$dir" ] || dir=$(ask "Install location [$default_dir]: " "$default_dir")

mkdir -p "$dir" 2>/dev/null || die "cannot create $dir — re-run with sudo, or set ORIEL_INSTALL_DIR to a writable path"
[ -w "$dir" ] || die "$dir is not writable — re-run with sudo, or set ORIEL_INSTALL_DIR to a writable path"
chmod +x "$tmp/oriel"
mv "$tmp/oriel" "$dir/oriel"
# Report the version we just installed (older binaries without the subcommand
# print nothing to stdout, so fall back to a plain name).
installed_as=$("$dir/oriel" version 2>/dev/null) || installed_as=""
echo "Installed ${installed_as:-oriel} to $dir/oriel"

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
  # --- reverse proxy / network exposure (optional) ---------------------------
  # Both settings are baked into the service unit (env vars ORIEL_BASE_PATH /
  # ORIEL_ALLOWED_HOSTS) so they survive restarts, reinstalls, and self-updates.

  # Sub-path — where a reverse proxy mounts Oriel. Cosmetic routing; low risk.
  # Leave blank to serve at the host root.
  base="${ORIEL_BASE_PATH:-}"
  [ -n "$base" ] || base=$(ask "Reverse-proxy sub-path? e.g. /oriel (blank = served at root): " "")

  # Allowed hosts — SECURITY-SENSITIVE. By default Oriel answers /api only on
  # localhost. Allowing a hostname lets a browser on that host drive Oriel, which
  # has NO authentication and controls Docker as root on this machine.
  allow="${ORIEL_ALLOWED_HOSTS:-}"
  if [ -z "$allow" ] && [ "$INTERACTIVE" = 1 ]; then
    {
      echo
      echo "  Network access"
      echo "  --------------"
      echo "  By default Oriel is reachable only from this machine (127.0.0.1)."
      echo "  To reach it through a reverse proxy or a private network (e.g."
      echo "  Tailscale), you must allow that hostname here."
      echo
      echo "  ⚠  WARNING: Oriel has NO authentication and can control Docker as"
      echo "     root. Only allow hosts on a network you trust, and keep TLS + a"
      echo "     login in front of it on the proxy. NEVER expose Oriel directly"
      echo "     to the public internet."
      echo
    } > /dev/tty
    allow=$(ask "Allowed host(s), comma-separated? e.g. oriel.example.com (blank = local only): " "")
  fi

  set -- service install
  [ -n "$base" ]  && set -- "$@" --base-path "$base"
  [ -n "$allow" ] && set -- "$@" --allowed-hosts "$allow"
  "$dir/oriel" "$@" || echo "Service setup failed — oriel is installed; run '$dir/oriel service install' yourself."
else
  echo
  echo "Run it:       $dir/oriel"
  echo "Or on login:  $dir/oriel service install"
fi
