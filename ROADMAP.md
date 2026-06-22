# Roadmap

A living view of where Oriel is headed — not a contract. Priorities shift with
feedback; the best way to influence them is to open an
[issue](https://github.com/ParadoxInfinite/oriel/issues) or a
[discussion](https://github.com/ParadoxInfinite/oriel/discussions).

## Now

- **Secret masking in inspect.** Container env vars often hold API keys and
  tokens; the inspect panel shows them in full today — easy to leak in a
  screenshot or screen-share. Mask sensitive values **server-side** by default
  (detected by key name or value shape), with a gated "Reveal values" action.
  Because it's server-side, the same masking keeps raw keys out of the MCP
  `inspect` tool below — so it lands first. See [docs/MCP.md](docs/MCP.md).
- **Oriel as an MCP server.** Expose Oriel's validated tool layer over the
  [Model Context Protocol](https://modelcontextprotocol.io) so any MCP-capable AI
  client — Claude Desktop, Claude Code, Cursor, a local Ollama-backed host, … —
  can inspect and manage your Docker/Colima **through Oriel's safety layer**.
  Every call still routes through the same argument + entity validation the UI
  uses. Starts with the **local stdio transport** (no network, no auth needed).
  This brings natural-language control **without bundling any model**: your client
  brings the model, so Oriel stays vendor-neutral and works with cloud or local
  LLMs. Includes adding **read/query tools** (list / inspect / logs / stats) so an
  AI can see state and target by description, not just mutate blind.
- **Time-boxed destructive grant.** AI-driven actions are read-only and reversible
  by default; destructive ones (remove / prune) stay **locked** until you grant
  them for a window you choose (`oriel ai allow-destructive --for 6h`, or a
  Settings toggle), then **auto-relock**. One control covering the MCP server and
  the in-app assistant alike — so even a headless AI can only do damage inside a
  window you opened on purpose.

## Next

- **Optional authentication.** Today Oriel binds to `127.0.0.1` and only guards
  against DNS rebinding (host allow-list) — there is no login. Add an **opt-in**
  token/password gate so Oriel (and MCP-over-HTTP) can be exposed safely beyond
  loopback. Off by default; local use unchanged. (See [SECURITY.md](SECURITY.md).)
- **In-browser container shell.** An interactive `exec` terminal into a running
  container, straight from the UI — built on the existing exec-streaming seam.
- **MCP over HTTP (Streamable HTTP).** Reach Oriel's MCP server from remote clients
  and hosted-AI connectors. Gated on optional authentication landing first.

## Later

- **In-app natural-language assistant.** A built-in command-palette mode that
  drives the same validated registry. **Provider-agnostic by design:** the base
  binary bundles no model — it talks to a resolver you point it at (a documented
  HTTP contract, plus an out-of-tree reference resolver you can run against Claude,
  an OpenAI-compatible endpoint, or a local model). For most users the MCP server
  above already delivers NL control through their own AI client, so this is a
  convenience layer, not the main path.
- **MCP resources & prompts.** Expose container logs, inspect output, and Compose
  files as readable MCP **resources**, plus canned diagnostic **prompts** — so an
  AI can read a crashing container's logs as context, not just call tools.
- **homebrew-core submission** — `brew install oriel` with no tap, once the
  project clears Homebrew's notability bar.

### Research (not committed work)

- **A small model to drive Oriel's tools.** Smallest reliable local model for our
  ~19 tools, and whether to train a standalone or "handoff" model. Findings +
  recommendation (TL;DR: ship MCP for existing models, add constrained decoding
  for the local path, don't train yet) in
  [docs/ai-model-research.md](docs/ai-model-research.md).

## Demand- or sponsor-gated

These aren't planned on their own. They happen only if the gate below is met.

- **Windows support.** Parked unless demand clearly flares up. If you want it,
  upvote/comment on the [Windows tracking issue](https://github.com/ParadoxInfinite/oriel/issues)
  — that demand is what moves it off this list.
- **Signed & notarized macOS binaries.** Requires Apple's **paid Developer
  Program (~$99/yr, recurring)**. Oriel won't fund an Apple subscription out of
  pocket — this only happens if sponsorship covers the ongoing cost. Until then
  the Homebrew cask strips the Gatekeeper quarantine attribute on install, so it
  works fine without it; the binaries just aren't Apple-blessed.

## Recently shipped

See the [CHANGELOG](CHANGELOG.md). Highlights: themeable swappable editions,
Compose discovery & deploy, CLI self-update + `doctor`, reverse-proxy hosting,
and Homebrew install (macOS).
