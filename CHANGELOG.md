# Changelog

All notable changes to Oriel are documented here.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.5.0] - 2026-06-23

The run-command palette grows up, and the in-app natural-language resolver starts
its way out.

### Added

- **The palette covers everything now.** ⌘K opens empty and suggests as you type
  instead of dumping every action up front, and it spans containers, images,
  volumes, networks, and prune — not just containers. Type "stop postgres" and it
  lists each matching container so you pick the exact one.
- **Jump anywhere from ⌘K.** New entries to go to any view, or open a container's
  logs straight from the palette.
- **Navigation is part of the theme SDK.** A shared `nav` seam (`navigate` /
  `takeTarget`) lets the palette and deep-links move whichever edition is mounted.
  Both built-in editions use it, and it's documented for theme authors in
  [docs/THEMES.md](docs/THEMES.md).

### Deprecated

- **In-app natural-language resolver (Settings → AI).** The single-shot text→tool
  resolver and the palette's "Interpret" mode are deprecated. Driving Oriel with a
  model is the MCP server's job (local or hosted), and it does more. This still
  works in 0.5.x and will be removed in 0.6.0 — see
  [docs/DEPRECATIONS.md](docs/DEPRECATIONS.md), including how to keep using it or
  ask for it to stay.

## [0.4.2] - 2026-06-23

A second hardening pass from a full adversarial code review: every fix below was
independently verified before merging. No new features; recommended for everyone
on 0.4.1 (it carries a UI regression fixed here).

### Fixed

- **Classic Dashboard "Retry" button** worked again — it threw a reference error
  in the Docker-unreachable state (a 0.4.1 regression), leaving the recovery
  action dead.
- **Disk prune reports failures.** A prune step that failed (daemon gone,
  permission denied, cancelled) was silently dropped and the job still reported
  success with a reclaim total; failures now surface and a cancelled run stops.
- **Pull dialog spinner** no longer sticks on after a search is superseded
  mid-type, and a dialog closed while typing cancels its pending request.
- **Settings writes are atomic.** Concurrent changes to different settings (mask
  mode, allowed hosts, provider URL, discovery) can no longer clobber each other.
- **One Escape closes one overlay.** With dialogs stacked (e.g. a confirm over a
  drawer), Escape dismissed the whole stack; now only the top layer closes.
- **Stale scan state cleared.** A failed Compose rescan no longer shows the prior
  scan's directories as if current.
- **Sturdier internals:** metrics-recorder shutdown no longer races its own
  flush, the update check no longer serializes behind a slow GitHub call, long
  command-output lines aren't silently truncated, and `service uninstall`/`status`
  as root no longer target the wrong unit (or claim success when nothing was
  removed). Service plists with special characters in the path render correctly.

### Security

- **Connection-string credentials are masked.** `DATABASE_URL` / `REDIS_URL` and
  similar values with embedded `user:pass@` no longer pass through in plaintext.
- **The AI/MCP path can't be talked out of masking.** `container.inspect` on the
  non-interactive (assistant/provider) path keeps a hard masking floor even when
  the local UI is set to reveal, and now also masks secrets in the command line
  and labels, not just env.

## [0.4.1] - 2026-06-22

A hardening and cleanup release: reliability and safety fixes from a full code
review, plus an internal refactor that makes the two editions presentation-only.
No new features; recommended for everyone on 0.4.0.

### Fixed

- **External responses are size-capped.** Registry/provider JSON and self-update
  binary downloads are now bounded, so a malformed or hostile endpoint can't
  exhaust memory or fill the disk.
- **Self-update is atomic.** The new binary is swapped into place in one step, so
  an interrupted update can't leave a half-written executable behind.
- **Live stream is sturdier.** Dropped SSE connections surface in the UI, the
  stream is idempotent across reconnects, and container inspect resets when you
  switch containers (no stale data carrying over).
- **Background jobs and timers clean up.** Finished prune/op jobs are reaped, the
  operation auto-dismiss timer is cleared, and a pending confirm dialog is
  cancelled rather than leaked.
- **Smaller UI correctness fixes.** Enter on a focused **Cancel** cancels,
  bulk **Start** only targets stopped containers, log and tag lists key stably,
  and external theme imports are guarded.
- **Explicit Colima profile** is threaded through, and some internal errors (e.g.
  filesystem listing) are surfaced instead of swallowed.

### Security

- **Stronger guardrails.** The destructive-grant window is clamped to a maximum
  duration, grant writes are atomic, secret masking covers more key names and
  value shapes, and tool entity references are validated at registration.

### Changed

- **Editions are presentation-only.** Behavior shared between the Studio and
  Classic UIs (lifecycle/stack ops, image tag/used-by, log formatting, Settings
  forms, dashboard telemetry) moved into shared headless controllers behind the
  platform SDK. No user-visible change.
- **CLI polish.** Cleaner subcommand help and exit codes, synchronous bind with a
  clean shutdown path.
- **Docs.** Trimmed repeated content across the README, MCP, and editions guides.

## [0.4.0] - 2026-06-22

The AI/MCP release: drive Docker/Colima from any MCP client through Oriel's
validated, secret-masked tools, with destructive actions locked behind a
time-boxed grant.

### Added

- **Oriel as an MCP server.** `oriel mcp` serves the tool registry to any
  MCP-capable client (Claude Desktop, Claude Code, Cursor, a local LLM host) over
  stdio JSON-RPC. It resolves the same Docker/Colima connection as the GUI and
  exposes all 19 tools through the same argument + entity validation the UI uses.
  No model ships in the binary; your client brings the model. See
  [docs/MCP.md](docs/MCP.md) for a sample client config.
- **Read/query tools in the registry.** `container.list` / `inspect` / `logs`,
  `image.list`, `volume.list`, `network.list`, `stacks.list`, `system.df`,
  `colima.status`, so an AI can *see* state and target by description, not just
  mutate blind. `container.inspect` masks env values like the UI does.
- **Time-boxed destructive grant.** Destructive tools (remove / prune) are locked
  for non-interactive callers (MCP / a future assistant) until you open a window
  on purpose: `oriel ai allow-destructive --for 6h` (also `status` / `lock`), or
  the new **Settings → Automation access** panel. Auto-relocks. Your own UI clicks
  are never gated. Enforced once in the execution path, so it covers every caller.
- **Secret masking in container inspect.** Environment-variable values are masked
  by default (`••••••••`) so API keys don't leak from screenshots or screen-shares.
  A gated "Reveal values" action unmasks them; masking is enforced server-side.
  Configurable in Settings → Secrets: mask mode (all / sensitive / off) and reveal
  policy (local / local & remote / off).

### Fixed

- **Siderail update pill opens the confirm-update modal directly** when the
  install can self-update, instead of just navigating to Settings (falls back to
  the Updates panel for Homebrew / unmanaged installs).

## [0.3.3] - 2026-06-22

### Fixed

- **The live demo no longer ships inside the release binary.** The `VITE_DEMO`
  guard didn't fold to a constant, so the mock backend and synthetic seed data
  were bundled into the real binary (v0.3.0–0.3.2). Now gated on a build-time
  literal with the demo modules marked side-effect-free, so they tree-shake out
  completely (verified: zero demo strings in the prod bundle).
- **Container logs: proper connecting and empty states (Studio).** While the
  stream connects you get a spinner; a container that has written nothing now
  shows "No logs yet" when a container hasn't written anything to stdout/stderr.
  instead of a stuck "Waiting for logs…".

### Security

- Pinned all third-party GitHub Actions to full commit SHAs (`goreleaser-action`
  and the three GitHub Pages actions). Supply-chain hardening, especially for
  the release workflow which runs alongside a cross-repo Homebrew-tap token.

### Demo

- The dashboard CPU graph now renders (its seeded history is anchored to real
  time, so it falls inside the chart's last-30-min window) and is smooth (a
  single system-level sample per tick instead of a per-container sum that spiked).

## [0.3.2] - 2026-06-22

### Fixed

- **Logs: restored the per-line gutter (regression).** Container logs again show a
  per-line marker: a wall-clock **timestamp** in a bordered gutter, plus a
  stream-coloured left edge for stderr/error, so lines are easy to tell apart.
  This was lost when lazy-loading replaced the old line-number gutter (line
  numbers shift as older lines load, so the gutter now uses stable timestamps).
- **Demo: an untagged image showed a blank label.** The mock's dangling image now
  carries the `<none>` tag the real Docker API reports, so it renders correctly
  and counts as dangling in the disk view and prune.

## [0.3.1] - 2026-06-22

### Added

- **Containers: select a whole stack at once.** Each stack group header now has a
  checkbox that selects (or clears) all of its containers, so you can
  start / stop / restart / remove an entire Compose stack from the bulk bar.

### Fixed

- **Homebrew installs no longer fight the in-app updater.** Oriel detects a
  Homebrew-managed binary and routes you to `brew upgrade oriel` instead of
  self-updating in place (which would desync Homebrew's tracked files). And
  `oriel service install` on a brew binary now targets the stable Homebrew
  symlink, so the service survives `brew upgrade` (the Caskroom path is versioned).

### Docs

- Clarified that the supported-runtimes **Platform** column is where *Oriel*
  runs: macOS and Linux only. Dropped "Windows" from runtime rows where it
  implied Windows support Oriel doesn't have; a Windows-hosted daemon is still
  reachable as a remote daemon.

## [0.3.0] - 2026-06-21

### Added

- **Homebrew install (macOS):** `brew install ParadoxInfinite/oriel/oriel`, via a
  new tap. The cask is generated and updated automatically on each release.
- **Live demo**: a "try it live" build of the full UI backed by in-memory mock
  data, deployed to GitHub Pages. No backend, no real Docker; a refresh resets it.
- Published a [ROADMAP](ROADMAP.md).

### Changed

- Releases are now built and published by [GoReleaser](https://goreleaser.com).
  The release assets are unchanged: the same static `oriel-<os>-<arch>` binaries
  and `SHA256SUMS.txt`, so `install.sh`, the curl one-liners, and `go install`
  work exactly as before; the only addition is the Homebrew cask.

## [0.2.4] - 2026-06-21

### Fixed

- Settings layout is now stable. It used CSS-columns masonry, which rebalanced by
  content height, so the same page could lay out differently on two instances,
  and any card height change (e.g. clicking "Check for updates") repainted the
  whole page. Replaced with an explicit two-column layout: each card has a fixed
  position regardless of content. Both editions.
- "Check for updates" no longer flickers; "Checking…" shows for a brief minimum
  instead of a one-frame flash, since the backend usually answers instantly from
  cache.

## [0.2.3] - 2026-06-21

### Fixed

- Settings → Updates: a missing space after the period ("…service install).A new
  version…"). Svelte was trimming the leading space inside the conditional; now
  forced explicitly. Both editions.

## [0.2.2] - 2026-06-21

### Fixed

- In-app self-update on Linux: "Restart now" no longer fails with "restart is only
  available for service-managed installs". Applying an update renames the running
  binary to `.bak`, after which the live process reports that `.bak` path and
  `IsManaged()` stopped matching the unit's `ExecStart`; the path is now normalized.

### Changed

- The siderail "update" pill opens the in-app Updates panel (Settings) instead of
  redirecting to the GitHub releases page.

## [0.2.1] - 2026-06-21

### Fixed

- Container logs: an empty but live stream now shows a "No logs yet" message when the container
  hasn't written to stdout/stderr" instead of a blank panel that looked like it
  failed to load. Log streams also send an initial flush + a periodic keepalive,
  so they open promptly through a reverse proxy and idle (quiet-container) streams
  aren't dropped by proxy read-timeouts.

## [0.2.0] - 2026-06-21

### Changed

- Studio's Settings cards now flow in a packed masonry layout (CSS columns) instead
  of a rigid grid, so cards sit directly under each other with no wasted whitespace.
- The Stacks folder button adapts to context: "Reveal in Finder" (macOS) / "Open
  folder" (Linux) for a local instance, and "Copy path" when viewing Oriel
  remotely, where opening a folder on the *server* is meaningless.
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
  Docker reachability) and warns when a base path is set with no allowed hosts (the proxy-403 footgun), so it's visible in `journalctl` without guessing.
- `oriel update [--check]`: checksum-verified CLI self-update for service-managed
  installs (check → download + verify → restart), so a headless box can upgrade
  from the terminal without the UI.
- `oriel config base-path [<path>|--clear]`: show, set, or clear the reverse-proxy
  sub-path in settings.json from the CLI; restarts a managed service to apply.
- `oriel doctor`: a read-only health check that reports Docker reachability, the
  running instance's base path + allowed hosts, version skew, and service status,
  and prints the exact fix command for anything wrong (e.g. a sub-path set with no
  allowed hosts → the proxy 403).
- `oriel remote <list|allow|deny> <host>` manages the running instance's host
  allow-list from the CLI over loopback. Changes apply immediately (no restart)
  and persist; run on the box itself, and it's the way out of the bootstrap
  deadlock where the reverse-proxy host is 403'd before you can reach Settings →
  Remote access.

### Fixed

- UI showed the Oriel version with a doubled `v` (`vv0.1.3`): the build version
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

- `install.sh` no longer aborts under `set -u` on the first download; a trailing
  multibyte ellipsis was fusing onto the `$asset` variable name.

### Added

- The release workflow can be triggered from the Actions tab (`workflow_dispatch`)
  to build a release from any branch HEAD, not only a pushed tag.

## [0.1.0] - 2026-06-21

First public release: a fast, local, single-binary web GUI for Colima and Docker,
with a swappable, themeable front end.

### Added

- **Lifecycle**: start / stop / restart Colima with live progress, plus a
  dashboard (CPU history graph, memory, disk, runtime, docker socket). Works
  against a plain Docker daemon too.
- **Containers**: live state, CPU and memory, exit code/status for stopped ones;
  start / stop / restart / remove; multi-select bulk actions; streaming logs (last
  100 instantly, lazy-load older with bounded memory) and a full inspect panel
  (env, mounts, networks, restart policy, health).
- **Images**: list, remove, per-tag untag, prune dangling, and a pull dialog with
  live registry search and tag suggestions (Docker Hub, Quay.io, AWS ECR Public;
  pull-by-ref for GHCR, GCR, registry.k8s.io, MCR). Digest-pinned images show
  their name instead of `<none>`, link their "in use" count to the containers
  holding them, and offer a one-click tag prefilled from the using container.
- **Volumes & Networks**: list, sort, remove, prune.
- **Compose stacks**: running stacks discovered from container labels (up / stop /
  restart / down), plus directory discovery: point Oriel at folders (recursive
  traversal, allow/deny filters, Oriel-only renames) and deploy compose projects
  that have never been started.
- **Reclaim space**: selectable prune (stopped containers, dangling images, build
  cache, unused networks/volumes; volumes opt-in and warned). Prunes run
  server-side as cancelable background jobs that survive a refresh, with a live
  progress bar in a sidebar operations tray.
- **Editions & themes**: swap the whole UI between **Studio** (default;
  light/dark/system) and **Classic** (dark teal), recolor with custom accents, or
  load an external theme bundle by URL, all from Settings. Built on a stable
  platform SDK so third parties can ship their own front end. See
  [docs/THEMES.md](docs/THEMES.md).
- **Live updates**: stats, CPU history, status and outages stream over one SSE
  connection, plus a filtered docker-event channel; the UI never polls.
- **Command palette** (`⌘K` / `Ctrl+K`): fuzzy-run any action, with an optional
  natural-language mode.
- **Natural-language control seam**: a dormant `ORIEL_PROVIDER_URL` plugin point;
  the base binary links no ML code, and every resolved action is re-validated
  against the tool registry.
- **Subpath hosting**: serve under a reverse-proxy path via `ORIEL_BASE_PATH`,
  no rebuild required.
- **Background service**: install as a launchd agent (macOS) or systemd unit
  (Linux, user or `--system`/root) that starts on login.
- **Version + in-app updates**: the sidebar footer shows the build version and
  flags newer GitHub releases; service-managed installs can self-update from
  Settings (download + SHA-256 verification against the release `SHA256SUMS.txt`,
  atomic binary replace, then a prompted restart, auto by default).
- **Single validated execution path**: UI buttons, the command palette, and the
  NL provider all route through `tools.Registry.Execute`, which checks arguments
  against each tool's schema and verifies referenced entities exist.

[Unreleased]: https://github.com/ParadoxInfinite/oriel/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/ParadoxInfinite/oriel/releases/tag/v0.1.0
