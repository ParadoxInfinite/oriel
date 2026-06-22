<div align="center">

<img src="docs/img/logo.svg" alt="Oriel" width="88" height="88" />

# Oriel

**A fast, local, single-binary GUI for Colima & Docker that's also an MCP server — manage your containers yourself, or let your AI do it. The open-source Docker Desktop alternative.**

Manage containers, images, volumes, networks and Compose stacks from a clean,
themeable browser UI — **or from any MCP client** (Claude Desktop, Claude Code,
Cursor, a local LLM), with every AI call behind the same validation,
secret-masking, and time-boxed destructive grant the UI uses. The GUI Colima
never had: ~15–30 MB RAM, no Electron, free and Apache-2.0. **So light it fades
into the background — run it on demand, or install it as a featherweight
service.** It runs **entirely on your machine — no cloud account, no central
server; any auth is yours to add, not a login to ours.** Binds to `127.0.0.1`;
runs on macOS & Linux.

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
- **AI control (MCP)** — `oriel mcp` lets any MCP client (Claude, Cursor, a local LLM) drive Docker/Colima through the same validated, secret-masked tools, with destructive actions locked behind a grant. [More ↓](#ai-control-mcp)
- **Images** — pull with registry search, prune, digest-pinned naming, one-click tag.
- **Compose** — manage running stacks, plus discover & deploy projects from disk.
- **Reclaim space** — selectable prune as cancelable background jobs that survive a refresh.
- **Dashboard** — CPU history, memory, disk, and uptime/outage tracking.
- **Command palette** (`⌘K`) — fuzzy-run any action; optional natural-language mode.
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
| AI control (MCP) | **Built-in, safety-gated** | MCP Toolkit (runs *other* servers) | No | No | No |

Oriel's wedge: a real **graphical** UI (not a TUI) **that's also an MCP server** (so an AI can drive it through the same safety checks you do), as a single local binary (not a server you deploy), that's **free and open source** (not commercially licensed), pointed at **whatever engine you already run** — most often Colima.

## AI control (MCP)

`oriel mcp` turns Oriel into a [Model Context Protocol](https://modelcontextprotocol.io)
server, so any MCP client — Claude Desktop, Claude Code, Cursor, a local LLM host
— can manage your Docker/Colima in natural language. Prefer the terminal? You
never have to open the GUI: `oriel mcp` is a full headless control surface on its
own. Either way, the AI goes through the **same validated tools the UI uses**, so
it gets the same guardrails you do:

- **Secrets stay masked** — `container.inspect` never hands raw env values to a model.
- **Destructive actions are locked** until you open a window on purpose (`oriel ai allow-destructive --for 6h`). Reads always work; remove/prune don't, until you say so.
- **No model in the binary** — your client brings the model; Oriel stays vendor-neutral.

```json
{ "mcpServers": { "oriel": { "command": "oriel", "args": ["mcp"] } } }
```

It's early — [the roadmap](ROADMAP.md) adds read-only & audited MCP access and
Colima-native AI control (start/stop the VM, not just the containers on it). Full
details: [docs/MCP.md](docs/MCP.md).

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
- **AI / natural-language control:** point any MCP client at `oriel mcp` (see [AI control](#ai-control-mcp)), or wire the in-app command palette to your own resolver URL (Settings → AI) — either way the base binary links no ML code.
- **Develop:** `make dev` + `make dev-web` (Vite hot reload), `make test`. See [CONTRIBUTING.md](CONTRIBUTING.md).
- **Roadmap:** actively developed — optional auth, read-only & audited MCP access, Colima-native AI control, in-browser shell — [ROADMAP.md](ROADMAP.md).

## FAQ

**Is there a GUI for Colima?**
Yes — Oriel is a graphical web UI built for Colima. Colima ships CLI-only by design; Oriel fills that gap (and also drives Docker Engine, OrbStack, Podman, and remote daemons).

**Is Oriel a free Docker Desktop alternative?**
Yes. It's Apache-2.0, with no license fees and no account — point it at any Docker-compatible engine and manage everything from the browser.

**Does Oriel need Docker Desktop?**
No. It needs any Docker Engine–compatible runtime plus the `docker` CLI. [Colima](https://github.com/abiosoft/colima) is first-class; see [docs/DAEMONS.md](docs/DAEMONS.md).

**Can an AI manage my containers?**
Yes — run `oriel mcp` and point any MCP client (Claude Desktop, Claude Code, Cursor, a local LLM) at it. The AI uses the same validated, secret-masked tools the UI does, and destructive actions stay locked until you open a window (`oriel ai allow-destructive --for 6h`). See [AI control](#ai-control-mcp).

**How is it different from lazydocker?**
lazydocker is a terminal UI. Oriel is a real graphical browser UI — dashboards, streaming logs, registry search, Compose discovery, and themeable editions.

**How much memory does it use?**
~15–30 MB. The binary is ~13 MB and there's no Electron.

**Which platforms are supported?**
macOS (Apple Silicon + Intel) and Linux (amd64 + arm64).

**Is it safe to run?**
Oriel has **no authentication** and driving Docker is root-equivalent on the host. Run it on `127.0.0.1`, or reach it over a private network only — never the public internet. See [SECURITY.md](SECURITY.md).

## Where it's headed

Actively developed. Next up:

- **Optional auth** — opt-in, your auth, not a login to anyone's server.
- **Read-only & audited MCP access** — scope what an AI can do, and keep a trail of what it did.
- **Colima-native AI control** — start/stop the VM and manage profiles, not just the containers on it.
- **In-browser container shell** and **MCP over HTTP** for remote clients.

See the [full roadmap](ROADMAP.md) — and **shape it**: open an
[issue](https://github.com/ParadoxInfinite/oriel/issues) or
[discussion](https://github.com/ParadoxInfinite/oriel/discussions) to request a
feature. No promises, but good requests are taken seriously.

## License

[Apache-2.0](LICENSE) © The Oriel contributors. See [NOTICE](NOTICE).
