<div align="center">

<img src="docs/img/logo.svg" alt="Oriel" width="88" height="88" />

# Oriel

**A bay window onto your local containers.**

A fast, local, single-binary web GUI for [Colima](https://github.com/abiosoft/colima)
and Docker — manage containers, images, volumes, networks and Compose stacks from
a clean browser UI, with a swappable, themeable front end.

[![CI](https://github.com/ParadoxInfinite/oriel/actions/workflows/ci.yml/badge.svg)](https://github.com/ParadoxInfinite/oriel/actions/workflows/ci.yml)
[![Release](https://img.shields.io/github/v/release/ParadoxInfinite/oriel?sort=semver)](https://github.com/ParadoxInfinite/oriel/releases)
[![License: Apache-2.0](https://img.shields.io/badge/License-Apache_2.0-blue.svg)](LICENSE)

</div>

Oriel is built to run on a potato: the Go backend idles around **15–30 MB RAM**,
the UI ships **embedded in one binary**, and it binds to `127.0.0.1` only — it's
meant to run locally next to your Colima VM or Docker daemon, not to be hosted.

What makes Oriel different from other container GUIs is the **edition system**:
the data and control layer is fixed and validated, while the entire presentation
is a swappable plugin. Two editions ship in the box, you can recolor them, and
third parties can drop in their own — the same idea as  ↔ .
See [docs/THEMES.md](docs/THEMES.md).

## Screenshots

<p align="center">
  <img src="docs/img/studio-light.png" alt="Oriel — Studio edition (light)" width="48%" />
  <img src="docs/img/studio-dark.png" alt="Oriel — Studio edition (dark)" width="48%" />
</p>
<p align="center">
  <img src="docs/img/classic.png" alt="Oriel — Classic edition" width="72%" />
</p>

> **Studio** is the default edition (System light/dark, follows your OS); **Classic**
> is a calm dark-teal alternative. Switch and recolor from Settings.

## Features

- **Lifecycle** — start / stop / restart Colima with live progress, plus a
  dashboard (CPU history graph, memory, disk, runtime, docker socket).
- **Containers** — live state, CPU and memory, plus the **exit code/status** for
  stopped ones; start / stop / restart / remove; **multi-select bulk actions**;
  streaming **logs** (last 100 instantly, scroll up to lazy-load older, memory
  stays bounded) and a full **inspect** panel (env, mounts, networks, restart
  policy, health).
- **Images** — list, remove, per-tag untag, prune dangling, and a **pull dialog
  with live registry search and tag suggestions** across Docker Hub, Quay.io and
  AWS ECR Public (plus pull-by-ref for GHCR, GCR, registry.k8s.io, MCR).
  **Digest-pinned images** (e.g. Compose `image: …@sha256:…`) show their name
  instead of `<none>`, link their **"in use" count to the containers** actually
  holding them, and offer a one-click **Tag** (pre-filled from the using container).
- **Volumes & Networks** — list, sort, remove, prune.
- **Compose stacks** — running ones discovered from container labels; up / stop /
  restart / down. Plus **directory discovery**: point Oriel at folders (with
  optional recursive traversal, allow/deny filters, and Oriel-only renames) and
  deploy compose projects that have never been started, right from the UI.
- **Reclaim space** — pick exactly what to prune (stopped containers, dangling
  images, build cache, unused networks/volumes — volumes opt-in and warned; build
  cache dangling-only by default with an all-unused override). Prunes run
  **server-side as background jobs**: they survive a refresh, show a live progress
  bar in a **sidebar operations tray**, and can be **cancelled** mid-run.
- **Command palette** (`⌘K` / `Ctrl+K`) — fuzzy-run any action; optional
  natural-language mode.
- **Editions & themes** — switch the whole UI (Studio / Classic), light/dark/system
  appearance, custom accent themes, or load an external theme by URL.
- **Live updates** — everything streams over **one SSE connection** (stats, CPU
  history, status, outages) plus a filtered docker-event channel; the UI **never
  polls**.

Everything routes through one validated **tool registry**, which is also the seam
for optional natural-language control (see below).

## Requirements

- macOS or Linux with a Docker Engine API–compatible runtime and the `docker`
  CLI available. [Colima](https://github.com/abiosoft/colima) is first-class
  (and unlocks VM start/stop), but Docker Engine, OrbStack, Rancher Desktop,
  Docker Desktop, Lima, Podman, and remote daemons all work — see
  **[docs/DAEMONS.md](docs/DAEMONS.md)**.
- For building from source: Go 1.26+ and Node 24+ (current LTS).

## Install

**Download a release binary** (no toolchain required). Grab the latest build for
your platform from [Releases](https://github.com/ParadoxInfinite/oriel/releases) —
prebuilt for `linux/amd64`, `linux/arm64`, `darwin/amd64`, and `darwin/arm64`:

```sh
# Linux arm64 (e.g. a Raspberry Pi); swap the suffix for your platform.
curl -fL https://github.com/ParadoxInfinite/oriel/releases/latest/download/oriel-linux-arm64 -o oriel
chmod +x oriel
./oriel
```

**With Go** (Oriel is a single module with the frontend embedded):

```sh
go install github.com/ParadoxInfinite/oriel@latest
oriel
```

**From source** — see [Build & run](#build--run) below.

## Build & run

```sh
make build   # builds the frontend, embeds it, produces ./oriel
./oriel      # opens http://127.0.0.1:4321 in your browser
```

Flags: `--port <n>` (default 4321), `--no-open` (don't launch a browser). If
Colima is stopped, the UI shows a zero-state with a **Start** button.

**Behind a reverse-proxy subpath.** Set `ORIEL_BASE_PATH` to serve under a path
instead of the host root — the same binary works at either, no rebuild:

```sh
ORIEL_BASE_PATH=/oriel ./oriel --no-open      # served at https://host/oriel/
```

Then mount `/oriel` in your proxy (e.g. `tailscale serve --set-path /oriel 4321`,
or nginx `location /oriel/`). It works whether the proxy strips the prefix or
passes it through. (Hard-refresh after changing the base — assets are cached.)

### Run as a background service

```sh
./oriel service install     # launchd (macOS) / systemd user unit (Linux); starts on login
./oriel service status
./oriel service uninstall
```

Pick a port with `service install --port 4399`. Place the binary somewhere stable
(e.g. `/usr/local/bin/oriel`) before installing. Logs: `~/Library/Logs/oriel.log`
(macOS) or `journalctl --user -u oriel -f` (Linux).

### Development

```sh
make dev       # backend on :4321 (serves the last-built UI)
make dev-web   # Vite dev server on :5173 with hot reload, proxying /api → :4321
make test      # Go unit tests
```

## Editions & themes

Oriel mounts exactly one **edition** — a complete front end built on a stable
**platform SDK**. Built in: **Studio** (clean, light/dark, the default) and
**Classic** (calm dark teal). Switch and theme from **Settings**, or author your
own. The full guide — the SDK contract, custom accents, and dropping in external
theme bundles via `window.__orielThemes` — is in **[docs/THEMES.md](docs/THEMES.md)**.

## Natural-language control (optional plugin seam)

The base ships **no model**. It exposes a dormant seam so you can add whatever
fits your machine — a few hundred KB of rules, an embedding model, or a local LLM
— as a **separate process**. The base binary never links any ML code.

Point Oriel at a resolver, either at launch or live from **Settings → AI**:

```sh
ORIEL_PROVIDER_URL=http://127.0.0.1:8899 ./oriel
```

When set, the command palette gains a free-text **Interpret** mode. The base POSTs
the text plus the available tools and live entities to `‹url›/resolve`, which
returns a `{tool, args}` call:

```json
{ "tool": "container.stop", "args": { "id": "app-postgres-1" }, "confidence": 0.9 }
```

That call runs through the **same validated execution path** as every button, so
a provider can never invoke an unknown tool or a non-existent entity. Return an
empty `"tool"` for "no match". A minimal rule-based provider is ~40 lines of
Python; an embedding- or LLM-backed one implements the same one route.

## Architecture

```
Browser (Svelte + Tailwind, embedded)  ── one edition, built on the platform SDK
   │  REST (actions) + SSE (live/events/logs, push-only)   — 127.0.0.1 only
   ▼
Go single binary
   ├─ server/      net/http + SSE + embedded static files
   ├─ tools/       Tool Registry — the canonical, validated action layer
   ├─ actions/     wires Docker/Colima ops into the registry + entity resolver
   ├─ docker/      Docker Engine API over the Colima unix socket
   ├─ colima/      `colima` CLI wrapper (status, lifecycle)
   ├─ execstream/  shared streaming-exec helper (colima + compose)
   └─ provider/    dormant NL seam
```

**One execution path.** UI buttons, the command palette, and the optional NL
provider all produce a `{tool, args}` call that goes through `tools.Registry.Execute`,
which validates arguments against the tool's schema **and** checks that referenced
entities exist before running. Safety lives in the base, not in any plugin.

## Security & remote access

Oriel has **no authentication** — it binds to `127.0.0.1` and trusts whoever can
reach it as the local user. Because it drives the Docker daemon, **reaching Oriel
is effectively root on the host.** So:

- **Run it locally.** The default — bound to `127.0.0.1`, next to your Colima VM
  or Docker daemon.
- **For remote access, use a private network ONLY** — a peer-to-peer mesh / VPN
  such as **Tailscale**, **ZeroTier**, **WireGuard**, or a Nebula/Headscale-style
  overlay. Keep Oriel on `127.0.0.1` and reach it over the private interface
  (e.g. `tailscale serve`, tailnet-only). **Never expose it to the public
  internet** — no port-forwarding, no `tailscale funnel`, no public reverse proxy.

Putting an unauthenticated, root-equivalent endpoint on the open internet is a
host takeover waiting to happen. The full trust model and remote-access guidance
is in **[SECURITY.md](SECURITY.md)**.

## Contributing

Issues and PRs welcome — see [CONTRIBUTING.md](CONTRIBUTING.md). New **editions and
themes** are especially welcome; start with [docs/THEMES.md](docs/THEMES.md).

## License

[Apache-2.0](LICENSE) © The Oriel contributors. See [NOTICE](NOTICE).
