# MCP server

> Status: **shipped** (stdio). `oriel mcp` serves the tool registry; destructive
> tools are gated behind the grant window. See [ROADMAP.md](../ROADMAP.md).

Oriel's tool layer, exposed over the
[Model Context Protocol](https://modelcontextprotocol.io). Any MCP client (Claude
Desktop, Claude Code, Cursor, a local Ollama host) can inspect and manage
Docker/Colima, and every call goes through the same checks the UI does.

> The in-app natural-language resolver (Settings → AI) was **removed in v0.6.0**;
> this MCP server replaced it. See [DEPRECATIONS.md](./DEPRECATIONS.md). MCP is the
> supported way to drive Oriel with a model, local or hosted.

## Set it up

The server is the `oriel mcp` subcommand: it speaks JSON-RPC over stdio, talks to
the same Docker/Colima the GUI does, and exposes every tool. Point any MCP client
at it.

**Before you start:** make sure `oriel` is on your `PATH` (`which oriel` should
print a path) and Colima/Docker is running. If `oriel` isn't on your `PATH`, use
its full path (e.g. `~/.local/bin/oriel`) wherever the steps below say `oriel`.

### Claude Code

One command, no files to edit:

```bash
claude mcp add oriel -- oriel mcp
```

That adds it for the current project. To use it in every project instead, add
`--scope user`:

```bash
claude mcp add oriel --scope user -- oriel mcp
```

Check it connected, then restart Claude Code (or open a new session) so it loads
the tools:

```bash
claude mcp get oriel      # look for: Status: ✔ Connected
```

Now ask it something like *"list my running containers"* — it'll call
`container.list`. To remove it later: `claude mcp remove oriel` (add `-s user` if
you used user scope).

### Claude Desktop, Cursor, and other clients

Add this to the client's MCP config (its `mcpServers` block), then restart the
client:

```json
{
  "mcpServers": {
    "oriel": { "command": "oriel", "args": ["mcp"] }
  }
}
```

Use the full path to `oriel` for `"command"` if it isn't on the client's `PATH`.

### Destructive actions need a grant

Read tools (`*.list`, `logs`, `inspect`, `status`) and reversible ones (`start` /
`stop` / `restart`) work right away. Destructive tools (`*.remove`, `*.prune`,
`stack.down`) return a "locked" error until you open a time-boxed window:

```bash
oriel ai allow-destructive --for 6h   # open the window
oriel ai status                       # check what's open
oriel ai lock                         # close it now
```

`--for` accepts a Go duration — `30s`, `90m`, `6h`, `1h30m`, `1.5h` (units `s` /
`m` / `h`, combinable) — or a days form like `2d` / `0.5d`. Anything from a few
seconds up to a **30-day** max; the days form doesn't combine with hours (use
`36h`, not `1d12h`).

Env values in `container.inspect` are always masked on this path, so secrets never
reach the model.

## Principles

- **No model in the base binary.** Oriel ships no LLM, no LLM client, no API-key
  handling. The user's MCP client brings the model, so this stays vendor-neutral
  and works with cloud or local LLMs.
- **One validated path.** Every MCP call routes through
  `internal/tools/Registry.Execute`, the same argument + entity-existence
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

### Registry → MCP mapping

| `tools.Tool` field | MCP tool field |
|---|---|
| `Name` | tool `name` |
| `Title` / `Description` | `description` |
| `Schema` | `inputSchema` (JSON Schema) |
| `Destructive` (true) | `annotations.destructiveHint: true` |
| read tools | `annotations.readOnlyHint: true` |
| start/restart | `annotations.idempotentHint: true` |
| `Entity *EntityRef` | enforced server-side (existence check before the handler) |

## Tools

Mutations: `container.start` / `stop` / `restart` / `remove`, `image.remove` /
`tag` / `prune`, `volume.remove` / `prune`, `network.remove`,
`stack.start` / `stop` / `restart` / `down`, `stack.alias`. Stack actions run
compose synchronously and return the collected output (the UI streams the same
actions for live progress); `stack.down` is destructive. `stack.alias` is a
display-only rename (sets the Oriel label for a project; the real compose name is
unchanged) — handy for an agent to organize stacks for the user.

Reads: `container.list` / `inspect` / `logs`, `image.list`, `volume.list`,
`network.list`, `stacks.list`, `system.df`, `colima.status`. These let an AI see
the current state and target things by description, instead of acting blind.

> Logs and inspect may also show up as MCP **resources** later (so a client can
> attach "this container's logs" as context). That's additive; they stay tools
> first.

## Safety: time-boxed destructive grant

- **Default:** read + reversible actions (`start`/`restart`/`stop`) allowed;
  `Destructive`-flagged tools (`remove`/`prune`/compose `down`) are **locked**.
- **Grant:** `oriel ai allow-destructive --for 6h` (or a Settings toggle). Stored
  with an `ExpiresAt`; **auto-relocks** when the window lapses.
- **Enforced once** on `tool.Destructive` inside the execution path, so it covers
  the MCP server and any non-interactive caller identically. Even a headless AI can
  only do damage inside a window the user opened on purpose.
- A locked call returns a structured error explaining how to grant, which an MCP
  client surfaces to the user.

## Secret masking (shared with the inspect UI)

`container.inspect` masks env values server-side, so MCP never feeds raw API
keys to a model. The placeholder is fixed (`OPENAI_API_KEY=••••••••`) — no value
or length is leaked.

- **Detection (sensitive mode):** by key name (`KEY` / `SECRET` / `TOKEN` /
  `PASSWORD` / `AUTH` / `CREDENTIAL` / `PRIVATE` / …) or by value shape (`sk-`,
  `ghp_`, `AKIA`, `-----BEGIN … PRIVATE KEY`, JWT, long high-entropy token).
- **Over MCP:** every env value is masked; there is no reveal on this path.
- **In the UI:** masked by default (setting: off / sensitive-only / all); a
  "Reveal values" action requests raw values explicitly and is gated to loopback.

## Transports

- **v1 (stdio).** `oriel mcp`. Local, no network, no auth needed; the client
  spawns it as a subprocess.
- **Later (Streamable HTTP).** Reach the server from remote/hosted clients and the
  hosted-AI MCP connectors. **Gated on the optional-auth tier landing first.**

## How it's built

Built on the official [`modelcontextprotocol/go-sdk`](https://github.com/modelcontextprotocol/go-sdk).
`oriel mcp` maps each registry tool to an MCP tool one-to-one via the SDK's
`Server.AddTool` (our JSON Schema in, a thin handler over `Registry.Execute`
out). Validation, the destructive-grant gate, and secret masking are reused from
that one path, not reimplemented; the adapter adds about 2 MB to the binary.

## Non-goals

Oriel exposes Docker and Colima over MCP. It is not an MCP *client* (it doesn't
consume other servers). Planned work (resources/prompts, MCP over HTTP) is on the
[roadmap](../ROADMAP.md).
