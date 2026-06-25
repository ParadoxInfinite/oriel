#!/bin/sh
# ---------------------------------------------------------------------------
# Oriel installer, downloads the right release binary for your platform,
# VERIFIES it against the published SHA256SUMS, and installs it onto your PATH.
#
# Running root-equivalent software from a piped script? Read it first. Or skip
# this and use the explicit per-platform commands in the README, they do
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

echo "Downloading ${asset}…"
fetch "$BASE/$asset" "$tmp/oriel" || die "download failed"
fetch "$BASE/SHA256SUMS.txt" "$tmp/sums" || die "could not fetch checksums"

# Match the checksum by exact filename column (field 2), not a line-suffix grep:
# anchoring on the whole field avoids selecting a line where the asset name is
# only a suffix of another entry. GoReleaser emits "<hex>  <name>" (optionally
# "*<name>" for binary mode), so accept either form.
expected=$(awk -v a="$asset" '$2==a || $2=="*"a {print $1; exit}' "$tmp/sums")
[ -n "$expected" ] || die "no checksum for $asset in SHA256SUMS.txt"

if command -v sha256sum >/dev/null 2>&1; then
  actual=$(sha256sum "$tmp/oriel" | awk '{print $1}')
else
  actual=$(shasum -a 256 "$tmp/oriel" | awk '{print $1}')
fi
[ "$expected" = "$actual" ] || die "checksum mismatch, refusing to install (expected $expected, got $actual)"
echo "Checksum verified."

# --- install location (env var > prompt > default) -------------------------
default_dir="$HOME/.local/bin"
if [ -d /usr/local/bin ] && [ -w /usr/local/bin ]; then default_dir=/usr/local/bin; fi
dir="${ORIEL_INSTALL_DIR:-}"
[ -n "$dir" ] || dir=$(ask "Install location [$default_dir]: " "$default_dir")

mkdir -p "$dir" 2>/dev/null || die "cannot create $dir, re-run with sudo, or set ORIEL_INSTALL_DIR to a writable path"
[ -w "$dir" ] || die "$dir is not writable, re-run with sudo, or set ORIEL_INSTALL_DIR to a writable path"
chmod +x "$tmp/oriel"
mv "$tmp/oriel" "$dir/oriel"
# Report the version we just installed (older binaries without the subcommand
# print nothing to stdout, so fall back to a plain name).
installed_as=$("$dir/oriel" version 2>/dev/null) || installed_as=""
echo "Installed ${installed_as:-oriel} to $dir/oriel"

case ":$PATH:" in
  *":$dir:"*) ;;
  *) echo "Note: $dir is not on your PATH, add it, or run $dir/oriel directly." ;;
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
  "$dir/oriel" service install || echo "Service setup failed, oriel is installed; run '$dir/oriel service install' yourself."
  # Reverse proxy / remote access is configured on the running instance (stored
  # in settings.json), not at install time. Point users at the commands.
  {
    echo
    echo "Behind a reverse proxy or reaching Oriel over a private network?"
    echo "  $dir/oriel config base-path /oriel       # serve under a sub-path"
    echo "  $dir/oriel remote allow <hostname>       # allow a host to reach /api"
    echo "  $dir/oriel doctor                        # check it's all wired up"
    echo "  ⚠ Oriel has no auth, only allow hosts on a trusted private network."
  } > /dev/tty 2>/dev/null || true
else
  echo
  echo "Run it:       $dir/oriel"
  echo "Or on login:  $dir/oriel service install"
fi
