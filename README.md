<div align="center">

<img src="docs/img/logo.svg" alt="Oriel" width="88" height="88" />

# Oriel

**A fast, local, single-binary web GUI for Colima & Docker — the open-source Docker Desktop alternative.**

Manage containers, images, volumes, networks and Compose stacks from a clean,
themeable browser UI. The GUI Colima never had: ~15–30 MB RAM, no Electron, no
login, free and Apache-2.0. Binds to `127.0.0.1`; runs on macOS & Linux.

[![CI](https://github.com/ParadoxInfinite/oriel/actions/workflows/ci.yml/badge.svg)](https://github.com/ParadoxInfinite/oriel/actions/workflows/ci.yml)
[![Release](https://img.shields.io/github/v/release/ParadoxInfinite/oriel?sort=semver)](https://github.com/ParadoxInfinite/oriel/releases)
[![License: Apache-2.0](https://img.shields.io/badge/License-Apache_2.0-blue.svg)](LICENSE)

</div>

<p align="center">
  <img src="docs/img/studio-light.png" alt="Oriel — Studio (light)" width="49%" />
  <img src="docs/img/studio-dark.png" alt="Oriel — Studio (dark)" width="49%" />
</p>

## Install

**Homebrew** (macOS & Linux):

```sh
brew install ParadoxInfinite/oriel/oriel
```

**Quick install** — detects your platform, verifies the checksum, installs to your PATH:

```sh
curl -fsSL https://raw.githubusercontent.com/ParadoxInfinite/oriel/main/install.sh | sh
```

> [!CAUTION]
> This pipes a script into your shell, which runs as you. **[Read `install.sh`](https://github.com/ParadoxInfinite/oriel/blob/main/install.sh) first** — or use the explicit per-platform commands below, which do the same thing, one intentional step at a time.

**Manual** — copy the one-liner for your platform:

macOS · Apple Silicon (M1+)

```sh
curl -fL https://github.com/ParadoxInfinite/oriel/releases/latest/download/oriel-darwin-arm64 -o oriel && chmod +x oriel
```

macOS · Intel

```sh
curl -fL https://github.com/ParadoxInfinite/oriel/releases/latest/download/oriel-darwin-amd64 -o oriel && chmod +x oriel
```

Linux · arm64

```sh
curl -fL https://github.com/ParadoxInfinite/oriel/releases/latest/download/oriel-linux-arm64 -o oriel && chmod +x oriel
```

Linux · amd64

```sh
curl -fL https://github.com/ParadoxInfinite/oriel/releases/latest/download/oriel-linux-amd64 -o oriel && chmod +x oriel
```

**Or with Go:**

```sh
go install github.com/ParadoxInfinite/oriel@latest
```

**Or from source:** `make build` (builds the UI, embeds it, produces `./oriel`).

## Run

```sh
./oriel              # opens http://127.0.0.1:4321
```

Flags: `--port <n>` (default 4321), `--no-open`. To run on login as a background service:

```sh
./oriel service install      # launchd (macOS) / systemd (Linux); also: status, uninstall
```

To reach Oriel from another machine through a reverse proxy or private network,
allow the proxy's hostname (`oriel remote allow <host>`) and, for a sub-path,
`oriel config base-path /oriel` — config is stored in `settings.json`. Run
`oriel doctor` to check it. See [docs/REVERSE-PROXY.md](docs/REVERSE-PROXY.md).
Oriel has no auth, so read the security note there first.

Needs a Docker Engine–compatible runtime + the `docker` CLI. [Colima](https://github.com/abiosoft/colima)
is first-class (adds VM start/stop); Docker Engine, OrbStack, Rancher/Docker Desktop,
Podman, and remote daemons also work — see [docs/DAEMONS.md](docs/DAEMONS.md).

## Features

- **Containers** — live CPU/mem, exit codes, bulk actions, streaming logs, full inspect.
- **Images** — pull with registry search, prune, digest-pinned naming, one-click tag.
- **Compose** — manage running stacks, plus discover & deploy projects from disk.
- **Reclaim space** — selectable prune as cancelable background jobs that survive a refresh.
- **Dashboard** — CPU history, memory, disk, and uptime/outage tracking.
- **Command palette** (`⌘K`) — fuzzy-run any action; optional natural-language mode.
- **MCP server** — `oriel mcp` exposes Docker/Colima to any MCP client (Claude Desktop, Claude Code, Cursor, a local LLM) through the same validated, secret-masked tools the UI uses; destructive actions stay locked until you grant a window.
- **Editions & themes** — swap the whole UI (Studio / Classic), light/dark/system, custom accents.
- **Live** — everything streams over one SSE connection; the UI never polls.
- **Self-update** — service installs update in-app via checksum-verified downloads.

## How it compares

How Oriel stacks up against the common ways to manage containers on a Mac or Linux box. (Figures drift — treat as a snapshot.)

| | **Oriel** | Docker Desktop | OrbStack | lazydocker | Portainer |
|---|---|---|---|---|---|
| License | **Apache-2.0, free** | Proprietary (paid for larger orgs) | Proprietary (paid for commercial use) | MIT, free | Free (CE) |
| Interface | Graphical web UI | Desktop app | Native app | Terminal (TUI) | Web UI (server) |
| Footprint | **~15–30 MB RAM, ~13 MB binary** | Heavy (Electron + VM) | Light (native) | Light | Needs a container + ≥2 GB RAM |
| Install | **Single static binary** | Installer | Installer | Single binary | Run a container |
| Bring-your-own engine | **Colima · Docker · OrbStack · Podman · remote** | Bundled engine | Bundled engine | Any Docker socket | Any Docker socket |
| Runs locally, no account | **Yes** | Account/sign-in | Account | Yes | Server + auth |

Oriel's wedge: a real **graphical** UI (not a TUI), as a single local binary (not a server you deploy), that's **free and open source** (not commercially licensed), pointed at **whatever engine you already run** — most often Colima.

## Editions & themes

The presentation is a swappable plugin on a stable platform SDK. Ships with
**Studio** (default; light/dark/system) and **Classic** (dark teal); recolor
either, or drop in your own. See [docs/THEMES.md](docs/THEMES.md).

<p align="center">
  <img src="docs/img/classic.png" alt="Oriel — Classic edition" width="70%" />
</p>

## Security

Oriel has **no authentication**, and driving Docker is effectively **root on the host**.
Run it locally, or reach it remotely over a **private network only** (Tailscale,
ZeroTier, WireGuard, …). **Never put it on the public internet.** Full trust model:
[SECURITY.md](SECURITY.md).

## More

- **Reverse-proxy subpath:** `oriel config base-path /oriel` — one build serves root or a subpath.
- **Natural-language control:** set an AI resolver URL (Settings → AI); suggestions run
  through the same validated tool path, and the base binary links no ML code.
- **Develop:** `make dev` + `make dev-web` (Vite hot reload), `make test`. See [CONTRIBUTING.md](CONTRIBUTING.md).
- **Roadmap:** where Oriel is headed (auth, in-browser container shell, …) — [ROADMAP.md](ROADMAP.md).

## FAQ

**Is there a GUI for Colima?**
Yes — Oriel is a graphical web UI built for Colima. Colima ships CLI-only by design; Oriel fills that gap (and also drives Docker Engine, OrbStack, Podman, and remote daemons).

**Is Oriel a free Docker Desktop alternative?**
Yes. It's Apache-2.0, with no license fees and no account — point it at any Docker-compatible engine and manage everything from the browser.

**Does Oriel need Docker Desktop?**
No. It needs any Docker Engine–compatible runtime plus the `docker` CLI. [Colima](https://github.com/abiosoft/colima) is first-class; see [docs/DAEMONS.md](docs/DAEMONS.md).

**How is it different from lazydocker?**
lazydocker is a terminal UI. Oriel is a real graphical browser UI — dashboards, streaming logs, registry search, Compose discovery, and themeable editions.

**How much memory does it use?**
~15–30 MB. The binary is ~13 MB and there's no Electron.

**Which platforms are supported?**
macOS (Apple Silicon + Intel) and Linux (amd64 + arm64).

**Is it safe to run?**
Oriel has **no authentication** and driving Docker is root-equivalent on the host. Run it on `127.0.0.1`, or reach it over a private network only — never the public internet. See [SECURITY.md](SECURITY.md).

## License

[Apache-2.0](LICENSE) © The Oriel contributors. See [NOTICE](NOTICE).
