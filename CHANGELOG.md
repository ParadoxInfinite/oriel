# Changelog

All notable changes to Oriel are documented here.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Changed

- Studio's Settings cards now flow in a packed masonry layout (CSS columns) instead
  of a rigid grid, so cards sit directly under each other with no wasted whitespace.
- The Stacks folder button adapts to context: "Reveal in Finder" (macOS) / "Open
  folder" (Linux) for a local instance, and "Copy path" when viewing Oriel
  remotely — where opening a folder on the *server* is meaningless.
- **Config is a single JSON file now, not environment variables.** `ORIEL_BASE_PATH`,
  `ORIEL_ALLOWED_HOSTS`, and `ORIEL_PROVIDER_URL` are deprecated: on first start
  they're migrated into `settings.json` automatically (and logged), then ignored.
  Config is now visible in the UI, editable from the CLI, and durable across
  reinstalls and self-updates. (`ORIEL_OUTAGE_RETENTION_DAYS` remains an advanced
  env-only tuning knob.)
- `service install` no longer bakes config into the unit; the `--base-path` /
  `--allowed-hosts` flags are removed. Configure the running instance instead with
  `oriel config base-path` / `oriel remote allow` (stored in settings.json).

### Added

- On startup the server logs a config summary (version, base path, allowed hosts,
  Docker reachability) and warns when a base path is set with no allowed hosts —
  the proxy-403 footgun — so it's visible in `journalctl` without guessing.
- `oriel update [--check]` — checksum-verified CLI self-update for service-managed
  installs (check → download + verify → restart), so a headless box can upgrade
  from the terminal without the UI.
- `oriel config base-path [<path>|--clear]` — show, set, or clear the reverse-proxy
  sub-path in settings.json from the CLI; restarts a managed service to apply.
- `oriel doctor` — a read-only health check that reports Docker reachability, the
  running instance's base path + allowed hosts, version skew, and service status,
  and prints the exact fix command for anything wrong (e.g. a sub-path set with no
  allowed hosts → the proxy 403).
- `oriel remote <list|allow|deny> <host>` manages the running instance's host
  allow-list from the CLI over loopback. Changes apply immediately (no restart)
  and persist — and run on the box itself, it's the way out of the bootstrap
  deadlock where the reverse-proxy host is 403'd before you can reach Settings →
  Remote access.

### Fixed

- UI showed the Oriel version with a doubled `v` (`vv0.1.3`) — the build version
  already includes the `v`, so the UI no longer prepends its own.

## [0.1.3] - 2026-06-21

### Added

- `oriel version` (also `--version` / `-v`) prints the build version. `install.sh`
  now reports the installed version after downloading.

### Fixed

- Re-running the installer on Linux now upgrades a running service in place:
  `service install` restarts the unit instead of `enable --now` (which left the
  old process running until a manual restart).

## [0.1.2] - 2026-06-21

### Added

- **Reverse-proxy setup as a first-class option.** `oriel service install` and
  `install.sh` now take `--base-path` (`ORIEL_BASE_PATH`) and `--allowed-hosts`
  (`ORIEL_ALLOWED_HOSTS`), baking them into the service unit so they survive
  restarts, reinstalls, and self-updates. The installer prompts for both and
  warns about the risk of allowing non-loopback hosts (no auth, root-equivalent).
  New guide: [docs/REVERSE-PROXY.md](docs/REVERSE-PROXY.md).

### Fixed

- SECURITY.md's Tailscale `serve` example now allows the tailnet host, which the
  anti-rebinding guard otherwise blocks with a 403.

## [0.1.1] - 2026-06-21

### Changed

- Build toolchain upgraded to Vite 8 and `@sveltejs/vite-plugin-svelte` 7.

### Fixed

- `install.sh` no longer aborts under `set -u` on the first download — a trailing
  multibyte ellipsis was fusing onto the `$asset` variable name.

### Added

- The release workflow can be triggered from the Actions tab (`workflow_dispatch`)
  to build a release from any branch HEAD, not only a pushed tag.

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
