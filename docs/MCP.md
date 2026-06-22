# MCP server — design

> Status: design / not yet implemented. See [ROADMAP.md](../ROADMAP.md).

Expose Oriel's validated tool layer over the
[Model Context Protocol](https://modelcontextprotocol.io) so **any** MCP-capable
AI client (Claude Desktop, Claude Code, Cursor, a local Ollama-backed host, …)
can inspect and manage Docker/Colima **through Oriel's safety checks**.

## Principles

- **No model in the base binary.** Oriel ships no LLM, no LLM client, no API-key
  handling. The user's MCP client brings the model — so this stays vendor-neutral
  and works with cloud or local LLMs.
- **One validated path.** Every MCP call routes through
  `internal/tools/Registry.Execute` — the same argument + entity-existence
  validation the UI uses. The MCP server is a thin adapter, not a second system.
- **Safe by default.** Read-only and reversible actions are allowed; destructive
  ones are locked behind a user-granted, time-boxed window (below).
- **Secrets never reach the model.** Inspect output is masked server-side.

## Architecture

```
MCP client (Claude Desktop / Code / Cursor / Ollama host)
        │  JSON-RPC 2.0 over stdio
        ▼
  `oriel mcp`  ──► internal/tools.Registry.Execute ──► internal/docker (Colima/Docker)
        (adapter)        (validation + safety)              (same client as the GUI)
```

`oriel mcp` is a subcommand that resolves the same Docker/Colima connection the
server does and serves MCP over stdio.

### Registry → MCP mapping (mechanical)

| `tools.Tool` field | MCP tool field |
|---|---|
| `Name` | tool `name` |
| `Title` / `Description` | `description` |
| `Schema` | `inputSchema` (JSON Schema) |
| `Destructive` (true) | `annotations.destructiveHint: true` |
| read tools | `annotations.readOnlyHint: true` |
| start/restart | `annotations.idempotentHint: true` |
| `Entity *EntityRef` | enforced server-side (existence check before the handler) |

## Tool surface

**Existing (mutations, already in the registry):** `container.start` / `stop` /
`restart` / `remove`, `image.remove` / `tag` / `prune`, `volume.remove` / `prune`,
`network.remove`.

**New read/query tools — added as first-class registry tools.** These are the
unlock: an AI can *see* state and target by description, not just mutate blind.
They become available to MCP **and** the future in-app assistant for free.

- `container.list`, `container.inspect`, `container.logs` (tail N), `container.stats`
- `image.list`, `volume.list`, `network.list`
- `system.df`, `colima.status`, `stacks.list`

> Logs and inspect may **also** be exposed as MCP **resources** later (so a client
> can attach "this container's logs" as context). That's additive — they are
> tools first; resources never replace them.

## Safety: time-boxed destructive grant

- **Default:** read + reversible actions (`start`/`restart`/`stop`) allowed;
  `Destructive`-flagged tools (`remove`/`prune`/compose `down`) are **locked**.
- **Grant:** `oriel ai allow-destructive --for 6h` (or a Settings toggle). Stored
  with an `ExpiresAt`; **auto-relocks** when the window lapses.
- **Enforced once** on `tool.Destructive` inside the execution path, so it covers
  the MCP server and the in-app assistant identically — even a headless AI can
  only do damage inside a window the user opened on purpose.
- A locked call returns a structured error explaining how to grant, which an MCP
  client surfaces to the user.

## Secret masking (shared with the inspect UI)

Lands **before** the `container.inspect` tool — otherwise MCP would feed raw API
keys to a model.

- `container.inspect` returns env with sensitive **values masked server-side** by
  default (e.g. `OPENAI_API_KEY = sk-ant-••••••••3f2a`).
- **Detection:** sensitive if the key name matches (`KEY` / `SECRET` / `TOKEN` /
  `PASSWORD` / `AUTH` / `CREDENTIAL` / `PRIVATE` / …) **or** the value matches a
  secret shape (`sk-`, `ghp_`, `AKIA`, `-----BEGIN … PRIVATE KEY`, JWT, long
  high-entropy string).
- **Over MCP:** raw secret values are **never** returned to the model (no reveal,
  or behind a separate explicit grant).
- **In the UI:** masked by default (setting: off / sensitive-only / all); a
  "Reveal values" action requests raw values explicitly and is gated to loopback.

## Transports

- **v1 — stdio.** `oriel mcp`. Local, no network, no auth needed; the client
  spawns it as a subprocess.
- **Later — Streamable HTTP.** Reach the server from remote/hosted clients and the
  hosted-AI MCP connectors. **Gated on the optional-auth tier landing first.**

## Implementation spike (decide before building)

MCP is JSON-RPC 2.0. Compare for **leanest + most spec-correct**, then pick one:

- the official `modelcontextprotocol/go-sdk`
- `mark3labs/mcp-go`
- a hand-rolled JSON-RPC handler

Spike = expose 1–2 tools over stdio, drive it from a real client (Claude
Desktop/Code), and measure the **binary-size delta** and added LOC. Smallest
that's correct wins.

## Phasing

1. Read tools in the registry + server-side secret masking.
2. MCP stdio server (tools) + the time-boxed destructive grant.
3. MCP resources (logs/inspect) + prompts.
4. MCP over HTTP (after optional auth).

## Non-goals

- No model bundled in the binary.
- No MCP **client** (Oriel consuming other servers) for now — the value is
  exposing Docker, not consuming external tools.

## Open questions

- Logs as tool vs. additionally a resource (tool first; resource later).
- Grant scope — global window in v1; per-client/session later.
- Reveal mechanism for masked secrets (gated server endpoint vs. never over MCP).
