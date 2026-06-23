<div align="center">

<img src="docs/img/logo.svg" alt="Oriel" width="88" height="88" />

# Oriel

**A small, local GUI for Colima and Docker that's also an MCP server. Manage containers from a browser, or hand the job to your AI. An open-source Docker Desktop alternative.**

Manage containers, images, volumes, networks, and Compose stacks from a clean
browser UI. Or skip the UI entirely and point an MCP client at it (Claude Desktop,
Claude Code, Cursor, a local LLM). The AI works through the same validation,
secret-masking, and destructive-action grant the UI does.

It's the GUI Colima never shipped: ~15–30 MB of RAM, no Electron, Apache-2.0.
Light enough to leave running as a service, or just start it when you need it.
Everything stays on your machine. No account, no central server, and if you want
a login you run it yourself. Binds to `127.0.0.1`; works on macOS and Linux.

[![CI](https://github.com/ParadoxInfinite/oriel/actions/workflows/ci.yml/badge.svg)](https://github.com/ParadoxInfinite/oriel/actions/workflows/ci.yml)
[![Release](https://img.shields.io/github/v/release/ParadoxInfinite/oriel?sort=semver)](https://github.com/ParadoxInfinite/oriel/releases)
[![License: Apache-2.0](https://img.shields.io/badge/License-Apache_2.0-blue.svg)](LICENSE)

</div>

<p align="center">
  <img src="docs/img/studio-light.png" alt="Oriel Studio (light)" width="49%" />
  <img src="docs/img/studio-dark.png" alt="Oriel Studio (dark)" width="49%" />
</p>

## Install

**Homebrew** (macOS & Linux):

```sh
brew install ParadoxInfinite/oriel/oriel
```

**Quick install**: detects your platform, verifies the checksum, installs to your PATH:

```sh
curl -fsSL https://raw.githubusercontent.com/ParadoxInfinite/oriel/main/install.sh | sh
```

> [!CAUTION]
> This pipes a script into your shell, which runs as you. **[Read `install.sh`](https://github.com/ParadoxInfinite/oriel/blob/main/install.sh) first**, or use the explicit per-platform commands below, which do the same thing, one intentional step at a time.

**Manual**: copy the one-liner for your platform:

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
`oriel config base-path /oriel`. Config is stored in `settings.json`. Run
`oriel doctor` to check it. See [docs/REVERSE-PROXY.md](docs/REVERSE-PROXY.md).
Oriel has no auth, so read the security note there first.

Needs a Docker Engine–compatible runtime + the `docker` CLI. [Colima](https://github.com/abiosoft/colima)
is first-class (adds VM start/stop); Docker Engine, OrbStack, Rancher/Docker Desktop,
Podman, and remote daemons also work. See [docs/DAEMONS.md](docs/DAEMONS.md).

## Features

- **Containers**: live CPU/mem, exit codes, bulk actions, streaming logs, full inspect.
- **AI control (MCP)**: `oriel mcp` lets any MCP client (Claude, Cursor, a local LLM) drive Docker/Colima through the same validated, secret-masked tools, with destructive actions locked behind a grant. [More ↓](#ai-control-mcp)
- **Images**: pull with registry search, prune, digest-pinned naming, one-click tag.
- **Compose**: manage running stacks, plus discover & deploy projects from disk.
- **Reclaim space**: selectable prune as cancelable background jobs that survive a refresh.
- **Dashboard**: CPU history, memory, disk, and uptime/outage tracking.
- **Command palette** (`⌘K`): fuzzy-run any action or jump to any view. (An optional natural-language mode is [deprecated in v0.5.0, removed in v0.6.0](docs/DEPRECATIONS.md); use the MCP server.)
- **Editions & themes**: swap the whole UI (Studio / Classic), light/dark/system, custom accents.
- **Live**: everything streams over one SSE connection; the UI never polls.
- **Self-update**: service installs update in-app via checksum-verified downloads.

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

When to reach for it:

- **Over Docker Desktop or OrbStack**: you want a graphical UI without a paid license, a bundled VM, or a desktop app sitting in your menu bar. Oriel is ~13 MB and drives the engine you already run.
- **Over lazydocker**: you want a real browser UI with dashboards and graphs, not a terminal one.
- **Over Portainer**: you want a binary you run for yourself, not a server to deploy and lock down.

And it's the only one here an AI can drive directly, through the same checks the UI gives you.

> **Coming:** read-only and audited MCP access, Colima VM control (start/stop, profiles), an in-browser shell, and MCP over HTTP. More on the [roadmap](ROADMAP.md).

## AI control (MCP)

`oriel mcp` runs Oriel as a [Model Context Protocol](https://modelcontextprotocol.io)
server, so an MCP client (Claude Desktop, Claude Code, Cursor, a local LLM) can
manage your Docker/Colima in plain English. You don't have to open the GUI at
all. It works headless. Either way the AI uses the same tools the UI does, with
the same guardrails:

- **Secrets stay masked**: `container.inspect` never hands raw env values to a model.
- **Destructive actions are locked** until you open a window on purpose (`oriel ai allow-destructive --for 6h`). Reads always work; remove/prune don't, until you say so.
- **No model in the binary**: your client brings the model; Oriel stays vendor-neutral.

```json
{ "mcpServers": { "oriel": { "command": "oriel", "args": ["mcp"] } } }
```

Full setup and the tool list are in [docs/MCP.md](docs/MCP.md).

## Editions & themes

The presentation is a swappable plugin on a stable platform SDK. Ships with
**Studio** (default; light/dark/system) and **Classic** (dark teal); recolor
either, or drop in your own. See [docs/THEMES.md](docs/THEMES.md).

<p align="center">
  <img src="docs/img/classic.png" alt="Oriel Classic edition" width="70%" />
</p>

## Security

Oriel has **no authentication**, and driving Docker is effectively **root on the host**.
Run it locally, or reach it remotely over a **private network only** (Tailscale,
ZeroTier, WireGuard, …). **Never put it on the public internet.** Full trust model:
[SECURITY.md](SECURITY.md).

## Develop

`make dev` + `make dev-web` (Vite hot reload), `make test`. See [CONTRIBUTING.md](CONTRIBUTING.md).

## FAQ

**Is there a GUI for Colima?**
Yes. Oriel is a graphical web UI built for Colima. Colima ships CLI-only by design, and Oriel fills that gap. It also drives Docker Engine, OrbStack, Podman, and remote daemons.

**Is Oriel a free Docker Desktop alternative?**
Yes. It's Apache-2.0, with no license fees and no account. Point it at any Docker-compatible engine and manage everything from the browser.

**Does Oriel need Docker Desktop?**
No. It needs any Docker Engine–compatible runtime plus the `docker` CLI. [Colima](https://github.com/abiosoft/colima) is first-class; see [docs/DAEMONS.md](docs/DAEMONS.md).

**Can an AI manage my containers?**
Yes. Run `oriel mcp` and point any MCP client (Claude Desktop, Claude Code, Cursor, a local LLM) at it. The AI uses the same validated, secret-masked tools the UI does, and destructive actions stay locked until you open a window (`oriel ai allow-destructive --for 6h`). See [AI control](#ai-control-mcp).

**How is it different from lazydocker?**
lazydocker is a terminal UI. Oriel is a real graphical browser UI, with dashboards, streaming logs, registry search, Compose discovery, and themeable editions.

**How much memory does it use?**
~15–30 MB. The binary is ~13 MB and there's no Electron.

**Which platforms are supported?**
macOS (Apple Silicon + Intel) and Linux (amd64 + arm64).

**Is it safe to run?**
Oriel has **no authentication** and driving Docker is root-equivalent on the host. Run it on `127.0.0.1`, or reach it over a private network only. Never the public internet. See [SECURITY.md](SECURITY.md).

**Where is it headed?**
Actively developed. Optional auth, read-only & audited MCP access, Colima-native VM control, an in-browser shell, and MCP over HTTP are next. The [roadmap](ROADMAP.md) has the full list, and the best way to move something up it is to open an [issue](https://github.com/ParadoxInfinite/oriel/issues) or [discussion](https://github.com/ParadoxInfinite/oriel/discussions).

## License

[Apache-2.0](LICENSE) © The Oriel contributors. See [NOTICE](NOTICE).
