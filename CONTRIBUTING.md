# Contributing to Oriel

Thanks for helping out! Oriel is a small, fast, single-binary container GUI. The
guiding principles: **stay lean** (tiny memory, instant start), **keep one
validated execution path**, and **keep presentation swappable** from the data
layer.

## Getting set up

Requirements: Go 1.26+, Node 24+ (current LTS), and a working `docker` CLI (via
Colima or any daemon).

```sh
make dev       # Go backend on :4321 (serves the last-built UI)
make dev-web   # Vite dev server on :5173, hot reload, proxies /api → :4321
make build     # full build → ./oriel  (frontend embedded)
make test      # Go unit tests
```

Work against **http://localhost:5173** for live reload. For anything that needs
the embedded UI or a real server response, run `./oriel` directly.

## Project layout

```
main.go, embed.go        entrypoint + embedded frontend
internal/
  server/                HTTP routes, SSE, config, system/registry helpers
  tools/                 the validated tool Registry (the one execution path)
  actions/               wires Docker/Colima ops into the registry
  docker/                Docker Engine API DTOs + calls
  colima/                colima CLI wrapper
  mcp/                   the MCP server (stdio + HTTP) over the tool registry
  secrets/               env/log secret masking
  grant/                 the time-boxed destructive-action grant
  service/               launchd / systemd install
web/src/
  platform/index.js      the platform SDK, the stable contract for editions
  lib/                   shared stores, helpers (sort, format, registry, api)
  components/            host-level global overlays (palette, confirm, toasts…)
  editions/              the swappable UIs (studio) + registry
```

## Where changes go

- **New backend capability** → add a tool in `internal/actions`/`tools` (so it's
  validated and reachable from buttons, the palette, and the MCP server alike), or
  a new HTTP handler in `internal/server` for read/stream endpoints.
- **Frontend logic used by more than one edition** → put it behind the platform
  SDK (`web/src/platform` / `web/src/lib`), not inside an edition. Editions should
  be presentation, not behavior.
- **A new look** → that's an edition or theme. See
  [docs/THEMES.md](docs/THEMES.md). New editions and themes are very welcome.

## Style

- **Go**: standard `gofmt`; small packages; DTOs in `internal/docker` so the API
  contract is stable across SDK upgrades. Don't leak SDK types past that boundary.
- **Svelte 5 + runes**: `$state` / `$derived` / `$effect`; Tailwind for layout.
  Scope edition CSS under a unique root class.
- **Comments**: explain the non-obvious *why*, not the *what*. Keep it terse.

## Pull requests

- Keep PRs focused: one coherent change. Flag unrelated fixes for a separate PR.
- `make build` and `make test` should pass.
- Note any new env vars, endpoints, or platform-SDK exports in your description.

## Security

Oriel binds to `127.0.0.1` only and ships no model. If you find a security issue,
please open a private report rather than a public issue.

## License

By contributing, you agree your contributions are licensed under the
[Apache License 2.0](LICENSE), per its Section 5 (no separate CLA required).
