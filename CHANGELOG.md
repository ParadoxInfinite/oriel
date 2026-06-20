# Changelog

All notable changes to Oriel are documented here.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.1.0] - 2026-06-21

First public release: a fast, local, single-binary web GUI for Colima and Docker,
with a swappable, themeable front end.

### Added

- **Lifecycle** — start / stop / restart Colima with live progress, plus a
  dashboard (CPU history graph, memory, disk, runtime, docker socket). Works
  against a plain Docker daemon too.
- **Containers** — live state, CPU and memory, exit code/status for stopped ones;
  start / stop / restart / remove; multi-select bulk actions; streaming logs (last
  100 instantly, lazy-load older with bounded memory) and a full inspect panel
  (env, mounts, networks, restart policy, health).
- **Images** — list, remove, per-tag untag, prune dangling, and a pull dialog with
  live registry search and tag suggestions (Docker Hub, Quay.io, AWS ECR Public;
  pull-by-ref for GHCR, GCR, registry.k8s.io, MCR). Digest-pinned images show
  their name instead of `<none>`, link their "in use" count to the containers
  holding them, and offer a one-click tag prefilled from the using container.
- **Volumes & Networks** — list, sort, remove, prune.
- **Compose stacks** — running stacks discovered from container labels (up / stop /
  restart / down), plus directory discovery: point Oriel at folders (recursive
  traversal, allow/deny filters, Oriel-only renames) and deploy compose projects
  that have never been started.
- **Reclaim space** — selectable prune (stopped containers, dangling images, build
  cache, unused networks/volumes; volumes opt-in and warned). Prunes run
  server-side as cancelable background jobs that survive a refresh, with a live
  progress bar in a sidebar operations tray.
- **Editions & themes** — swap the whole UI between **Studio** (default;
  light/dark/system) and **Classic** (dark teal), recolor with custom accents, or
  load an external theme bundle by URL, all from Settings. Built on a stable
  platform SDK so third parties can ship their own front end. See
  [docs/THEMES.md](docs/THEMES.md).
- **Live updates** — stats, CPU history, status and outages stream over one SSE
  connection, plus a filtered docker-event channel; the UI never polls.
- **Command palette** (`⌘K` / `Ctrl+K`) — fuzzy-run any action, with an optional
  natural-language mode.
- **Natural-language control seam** — a dormant `ORIEL_PROVIDER_URL` plugin point;
  the base binary links no ML code, and every resolved action is re-validated
  against the tool registry.
- **Subpath hosting** — serve under a reverse-proxy path via `ORIEL_BASE_PATH`,
  no rebuild required.
- **Background service** — install as a launchd agent (macOS) or systemd unit
  (Linux, user or `--system`/root) that starts on login.
- **Version + in-app updates** — the sidebar footer shows the build version and
  flags newer GitHub releases; service-managed installs can self-update from
  Settings (download + SHA-256 verification against the release `SHA256SUMS.txt`,
  atomic binary replace, then a prompted restart — auto by default).
- **Single validated execution path** — UI buttons, the command palette, and the
  NL provider all route through `tools.Registry.Execute`, which checks arguments
  against each tool's schema and verifies referenced entities exist.

[Unreleased]: https://github.com/ParadoxInfinite/oriel/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/ParadoxInfinite/oriel/releases/tag/v0.1.0
