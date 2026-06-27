#!/bin/sh
# ---------------------------------------------------------------------------
# Oriel installer: downloads the right release binary for your platform,
# VERIFIES it against the published SHA256SUMS, and installs it onto your PATH.
#
# Running root-equivalent software from a piped script? Read it first, or use
# the explicit per-platform commands in the README instead.
#
# Run interactively and it asks where to install and whether to set up the
# background service. For unattended installs, set any of:
#     ORIEL_INSTALL_DIR=/path        install location (default: /usr/local/bin or ~/.local/bin)
#     ORIEL_SERVICE=1                also install + start the background service (0 to skip)
#     ORIEL_CHANNEL=stable|edge      release channel (default stable; edge = newest, incl. pre-releases)
#     ORIEL_UNINSTALL=1              remove Oriel (and its login service) instead of installing
# ---------------------------------------------------------------------------
set -eu

REPO="ParadoxInfinite/oriel"

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

# --- options ---------------------------------------------------------------
# Flags work when piped, e.g.  curl … | sh -s -- --edge
# (the env vars above only reach the script if set on `sh`, not on `curl`).
while [ "$#" -gt 0 ]; do
  case "$1" in
    --edge) ORIEL_CHANNEL=edge ;;
    --stable) ORIEL_CHANNEL=stable ;;
    --channel) shift; ORIEL_CHANNEL="${1:-}" ;;
    --channel=*) ORIEL_CHANNEL="${1#*=}" ;;
    --uninstall) ORIEL_UNINSTALL=1 ;;
    --service) ORIEL_SERVICE=1 ;;
    --no-service) ORIEL_SERVICE=0 ;;
    -h | --help)
      echo "usage: install.sh [--edge|--stable|--channel C] [--uninstall] [--service|--no-service]"
      exit 0 ;;
    *) die "unknown option: $1 (try --help)" ;;
  esac
  shift
done

# --- pick a downloader -----------------------------------------------------
if command -v curl >/dev/null 2>&1; then
  fetch() { curl -fSL "$1" -o "$2"; }   # to a file
  fetch_out() { curl -fsSL "$1"; }      # to stdout
elif command -v wget >/dev/null 2>&1; then
  fetch() { wget -qO "$2" "$1"; }
  fetch_out() { wget -qO- "$1"; }
else
  die "need curl or wget"
fi

# --- detect an existing install --------------------------------------------
existing=$(command -v oriel 2>/dev/null || true)

resolve() { # follow symlinks to the real file (macOS readlink has no -f)
  _p=$1
  while [ -L "$_p" ]; do
    _l=$(readlink "$_p")
    case "$_l" in /*) _p=$_l ;; *) _p=$(dirname "$_p")/$_l ;; esac
  done
  printf '%s' "$_p"
}

# Is the oriel we'd touch managed by Homebrew? Check the RESOLVED path, not the
# brew DB: a user can have both a brew oriel and a script one, so we must defer to
# brew only when the binary actually on PATH is the brew-managed file (brew stores
# formulae under Cellar/, casks under Caskroom/).
is_brew_oriel() {
  [ -n "$existing" ] || return 1
  case "$(resolve "$existing")" in
    */Cellar/* | */Caskroom/*) return 0 ;;
    *) return 1 ;;
  esac
}

# --- uninstall -------------------------------------------------------------
if [ "${ORIEL_UNINSTALL:-}" = 1 ]; then
  [ -n "$existing" ] || { echo "Oriel doesn't appear to be installed."; exit 0; }
  if is_brew_oriel; then
    echo "Oriel was installed with Homebrew. Uninstall it with:"
    echo "  brew uninstall oriel"
    exit 0
  fi
  target=$(resolve "$existing")
  "$existing" service uninstall >/dev/null 2>&1 || true # remove the login service if present
  rm -f "$target" && echo "Removed $target"
  echo "Your settings in the OS config dir were left in place; delete them for a clean slate."
  exit 0
fi

# --- release channel -------------------------------------------------------
CHANNEL="${ORIEL_CHANNEL:-stable}"
case "$CHANNEL" in
  stable | edge) ;;
  *) die "ORIEL_CHANNEL must be 'stable' or 'edge'" ;;
esac

# --- already installed? upgrade in place; never clobber a package manager ---
if [ -n "$existing" ]; then
  cur=$("$existing" version 2>/dev/null || echo "an existing build")
  if is_brew_oriel; then
    echo "Oriel ($cur) is already installed via Homebrew."
    echo "Upgrade it with:  brew upgrade oriel"
    echo "(Re-running this script would overwrite the Homebrew copy and desync brew, so it won't.)"
    exit 0
  fi
  echo "Oriel is already installed ($cur) at $existing."
  if [ "$INTERACTIVE" = 1 ]; then
    confirm "Replace it with the latest $CHANNEL build? [y/N]: " || { echo "Left the existing install in place."; exit 0; }
  else
    echo "Replacing it with the latest $CHANNEL build…"
  fi
  # Upgrade in place: install over the existing binary's directory unless the
  # caller pinned ORIEL_INSTALL_DIR.
  : "${ORIEL_INSTALL_DIR:=$(dirname "$(resolve "$existing")")}"
fi

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

# --- resolve the download base for the channel -----------------------------
if [ "$CHANNEL" = edge ]; then
  # Newest release of ANY kind (GitHub lists newest-first; the first tag_name
  # is it, pre-release or not). No jq dependency.
  tag=$(fetch_out "https://api.github.com/repos/$REPO/releases" 2>/dev/null \
    | grep -m1 '"tag_name":' | sed 's/.*"tag_name":[ ]*"\([^"]*\)".*/\1/')
  [ -n "$tag" ] || die "could not resolve the latest edge release from GitHub"
  BASE="https://github.com/$REPO/releases/download/$tag"
  echo "Channel: edge ($tag)"
else
  BASE="https://github.com/$REPO/releases/latest/download"
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
    echo "  $dir/oriel config base-path /oriel          # serve under a sub-path"
    echo "  $dir/oriel config auth-token --generate     # require a login for remote access"
    echo "  $dir/oriel remote allow <hostname>          # allow a host to reach /api"
    echo "  $dir/oriel doctor                           # check it's all wired up"
    echo "  ⚠ Driving Docker is root-equivalent; only allow hosts on a trusted private network."
  } > /dev/tty 2>/dev/null || true
else
  echo
  echo "Run it:       $dir/oriel"
  echo "Or on login:  $dir/oriel service install"
fi
