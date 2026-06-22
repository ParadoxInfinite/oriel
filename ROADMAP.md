# Roadmap

A living view of where Oriel is headed — not a contract. Priorities shift with
feedback; the best way to influence them is to open an
[issue](https://github.com/ParadoxInfinite/oriel/issues) or a
[discussion](https://github.com/ParadoxInfinite/oriel/discussions).

## Now

- **Optional authentication.** Today Oriel binds to `127.0.0.1` and only guards
  against DNS rebinding (host allow-list) — there is no login. Add an **opt-in**
  token/password gate so Oriel (and MCP-over-HTTP) can be exposed safely beyond
  loopback. Off by default; local use unchanged. (See [SECURITY.md](SECURITY.md).)
- **In-browser container shell.** An interactive `exec` terminal into a running
  container, straight from the UI — built on the existing exec-streaming seam.
- **MCP over HTTP (Streamable HTTP).** Reach Oriel's MCP server from remote clients
  and hosted-AI connectors. Gated on optional authentication landing first.
- **Read-only & scoped MCP access.** A `--read-only` mode plus per-tool /
  per-namespace allow-lists, so you can hand an MCP client a deliberately limited
  surface (e.g. logs + inspect only, no lifecycle). Mirrors the proven flags from
  the Kubernetes MCP server and layers with the destructive grant for
  defense-in-depth. Among Docker-management MCP servers, gating like this is the
  exception, not the norm.
- **Audit log of AI-initiated actions.** Record who / what / when for every tool
  call that arrives through the MCP server or assistant (human UI clicks excluded)
  — a durable trail of what an AI did to your environment. No other
  Docker-management MCP server offers this; it's a trust anchor for AI-driven ops
  and a natural complement to the grant.

## Next

- **In-app natural-language assistant.** A built-in command-palette mode that
  drives the same validated registry, for users who'd rather not wire up an
  external MCP client. Provider-agnostic — the base bundles no model.
- **MCP resources & prompts.** Expose container logs, inspect output, and Compose
  files as readable MCP **resources**, plus canned diagnostic **prompts** — so an
  AI can read a crashing container's logs as context, not just call tools.
- **Colima as a first-class MCP target.** Today only `colima.status` is exposed
  (read-only). Add gated lifecycle + config tools — `colima.start` / `stop` /
  `restart`, plus profile and resource (CPU / memory / disk) management — so an AI
  can manage the **VM**, not just the containers on it. The wedge no competitor
  can copy: every other Docker MCP server just repoints the socket and is blind to
  Colima.

## Later

- **homebrew-core submission** — `brew install oriel` with no tap, once the
  project clears Homebrew's notability bar.



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

See the [CHANGELOG](CHANGELOG.md). Highlights: **MCP server (`oriel mcp`) + read
tools, secret masking in inspect, and the time-boxed destructive grant** (v0.4.0);
themeable swappable editions, Compose discovery & deploy, CLI self-update +
`doctor`, reverse-proxy hosting, and Homebrew install (macOS).
