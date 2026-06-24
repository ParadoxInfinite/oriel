<div align="center">

<img src="docs/img/logo.svg" alt="Oriel" width="88" height="88" />

# Oriel

**The local Docker GUI an AI can drive — safely.**

Manage containers, images, volumes, networks, and Compose from a clean browser UI — or hand the job to any MCP client (Claude, Cursor, a local LLM), through the same secret-masking and destructive-action grant the UI uses. Works with **any Docker engine — Colima, Docker Engine, OrbStack, Podman, remote — on macOS and Linux**. A free, open-source Docker Desktop alternative: one ~13 MB binary, ~15–30 MB RAM, no Electron, no account, Apache-2.0.

[![CI](https://github.com/ParadoxInfinite/oriel/actions/workflows/ci.yml/badge.svg)](https://github.com/ParadoxInfinite/oriel/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/ParadoxInfinite/oriel)](https://goreportcard.com/report/github.com/ParadoxInfinite/oriel)
[![Release](https://img.shields.io/github/v/release/ParadoxInfinite/oriel?sort=semver)](https://github.com/ParadoxInfinite/oriel/releases)
[![License: Apache-2.0](https://img.shields.io/badge/License-Apache_2.0-blue.svg)](LICENSE)

</div>

<p align="center">
  <img src="docs/img/studio-light.png" alt="Oriel Studio (light)" width="49%" />
  <img src="docs/img/studio-dark.png" alt="Oriel Studio (dark)" width="49%" />
</p>

## Drive it with AI

Point any MCP client at Oriel and ask in plain English. It calls the same validated, secret-masked tools the UI does — chaining several to actually answer, not just firing one:

> **you** — why does `api` keep restarting?
>
> **assistant** — *reads `container.list` → `container.inspect` → the logs* — it's exiting **137 (OOM)**: the container is capped at 256 MB and the Node heap runs past it. Raise the memory limit or fix the leak in `worker.js`.

```sh
oriel mcp        # stdio MCP server for Claude, Cursor, or a local LLM
```

No model ships in the binary — your client brings it. Reads run anytime; destructive actions stay locked behind a grant. [Setup & tool list ↓](#ai-control-mcp)

## Install

**Homebrew** (macOS & Linux):

```sh
brew install ParadoxInfinite/oriel/oriel
```

**Script** — detects your platform, verifies the checksum ([read it first](https://github.com/ParadoxInfinite/oriel/blob/main/install.sh)):

```sh
curl -fsSL https://raw.githubusercontent.com/ParadoxInfinite/oriel/main/install.sh | sh
```

<details>
<summary>Manual binary, Go, or source</summary>

Download a binary from [releases](https://github.com/ParadoxInfinite/oriel/releases/latest) — `oriel-darwin-arm64` (Apple Silicon), `oriel-darwin-amd64` (Intel), `oriel-linux-arm64`, or `oriel-linux-amd64` — then `chmod +x`. Or:

```sh
go install github.com/ParadoxInfinite/oriel@latest   # with Go
make build                                           # from source (builds + embeds the UI)
```

</details>

## Run

```sh
./oriel            # opens http://127.0.0.1:4321
```

Flags: `--port <n>` (default 4321), `--no-open`. Run on login: `./oriel service install` (launchd / systemd; also `status`, `uninstall`).

Needs any Docker Engine–compatible runtime + the `docker` CLI. [Colima](https://github.com/abiosoft/colima) is first-class (adds VM start/stop); Docker Engine, OrbStack, Rancher/Docker Desktop, Podman, and remote daemons also work ([docs/DAEMONS.md](docs/DAEMONS.md)). For remote access over a private network, see [docs/REVERSE-PROXY.md](docs/REVERSE-PROXY.md) — and the [security note](#security) first.

## Features

- **Containers** — live CPU/mem, exit codes, bulk actions, streaming logs, full inspect.
- **AI control (MCP)** — any MCP client drives Docker/Colima through the same validated, secret-masked tools, with destructive actions behind a grant. [More ↓](#ai-control-mcp)
- **Images** — pull with registry search, prune, one-click tag.
- **Compose** — manage stacks, plus discover & deploy projects from disk.
- **Dashboard** — CPU history, memory, disk, uptime/outage tracking.
- **Command palette** (`⌘K`) — fuzzy-run any action or jump to any view.
- **Editions & themes** — swap the whole UI (Studio / Classic), light/dark/system, custom accents.
- **Light & live** — ~15–30 MB RAM, one SSE stream (no polling), checksum-verified self-update.

## How it compares

The usual ways to run containers on a Mac or Linux box, and where Oriel fits. (Figures drift; treat as a snapshot.)

| | **Oriel** | Docker Desktop | OrbStack | lazydocker | Portainer |
|---|---|---|---|---|---|
| License | **Apache-2.0, free** | Proprietary (paid for larger orgs) | Proprietary (paid for commercial use) | MIT, free | Free (CE) |
| Interface | Graphical web UI | Desktop app | Native app | Terminal (TUI) | Web UI (server) |
| Footprint | **~15–30 MB RAM, ~13 MB binary** | Heavy (Electron + VM) | Light (native) | Light | Needs a container + ≥2 GB RAM |
| Install | **Single static binary** | Installer | Installer | Single binary | Run a container |
| Bring-your-own engine | **Colima · Docker · OrbStack · Podman · remote** | Bundled engine | Bundled engine | Any Docker socket | Any Docker socket |
| Runs locally, no account | **Yes** | Account/sign-in | Account | Yes | Server + auth |
| AI control (MCP) | **Built-in, safety-gated** | MCP Toolkit (runs *other* servers) | No | No | No |

It's the only one here an AI can drive directly, through the same checks the UI gives you. Reach for it over Docker Desktop / OrbStack (no paid license, bundled VM, or menu-bar app), over lazydocker (a real graphical UI, not a terminal one), or over Portainer (a binary you run for yourself, not a server to deploy and lock down).

> **Coming:** read-only & audited MCP access, Colima VM control, an in-browser shell, MCP over HTTP, and optional auth. [Roadmap](ROADMAP.md).

## AI control (MCP)

`oriel mcp` runs Oriel as a [Model Context Protocol](https://modelcontextprotocol.io) server, so an MCP client (Claude Desktop, Claude Code, Cursor, a local LLM) manages your Docker/Colima in plain English — headless, no GUI needed. Same tools, same guardrails:

- **Secrets stay masked** — `container.inspect` never hands raw env values to a model.
- **Destructive actions are locked** until you open a window (`oriel ai allow-destructive --for 6h`). Reads always work; remove/prune don't, until you say so.
- **No model in the binary** — your client brings the model; Oriel stays vendor-neutral.

```json
{ "mcpServers": { "oriel": { "command": "oriel", "args": ["mcp"] } } }
```

Setup and the full tool list: [docs/MCP.md](docs/MCP.md).

## Editions & themes

The UI is a swappable plugin on a stable platform SDK: **Studio** (default; light/dark/system) and **Classic** (dark teal). Recolor either, or drop in your own — [docs/THEMES.md](docs/THEMES.md).

<p align="center">
  <img src="docs/img/classic.png" alt="Oriel Classic edition" width="70%" />
</p>

## Security

Oriel has **no authentication**, and driving Docker is effectively **root on the host**. Run it locally, or over a **private network only** (Tailscale, ZeroTier, WireGuard, …) — **never the public internet**. Full trust model: [SECURITY.md](SECURITY.md).

## FAQ

**Is there a GUI for Colima?** Yes — Colima ships CLI-only by design, and Oriel is the browser UI it never shipped. It also drives Docker Engine, OrbStack, Podman, and remote daemons.

**A free Docker Desktop alternative?** Yes — Apache-2.0, no license fees, no account. Point it at any Docker-compatible engine.

**Can an AI manage my containers?** Yes — run `oriel mcp` and point any MCP client (Claude, Cursor, a local LLM) at it. Same validated, secret-masked tools as the UI; destructive actions stay locked until you grant them.

**How is it different from lazydocker?** lazydocker is a terminal UI; Oriel is a graphical browser UI with dashboards, streaming logs, registry search, Compose discovery, and themeable editions.

**Footprint and platforms?** ~15–30 MB RAM, ~13 MB binary, no Electron. macOS (Apple Silicon + Intel) and Linux (amd64 + arm64).

## Develop

`make dev` + `make dev-web` (Vite hot reload), `make test`. See [CONTRIBUTING.md](CONTRIBUTING.md).

## License

[Apache-2.0](LICENSE) © The Oriel contributors. See [NOTICE](NOTICE).
