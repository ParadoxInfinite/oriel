# Roadmap

Where Oriel's headed. Not a promise. Priorities move with feedback, and the best
way to push something up the list is to open an
[issue](https://github.com/ParadoxInfinite/oriel/issues) or a
[discussion](https://github.com/ParadoxInfinite/oriel/discussions).

## Now

- **Optional authentication.** Today Oriel binds to `127.0.0.1` and only guards
  against DNS rebinding (host allow-list), and there's no login. Add an **opt-in**
  token/password gate so Oriel (and MCP-over-HTTP) can be exposed safely beyond
  loopback. Off by default; local use unchanged. (See [SECURITY.md](SECURITY.md).)
- **In-browser container shell.** An interactive `exec` terminal into a running
  container, straight from the UI, built on the existing exec-streaming seam.
- **MCP over HTTP (Streamable HTTP).** Reach Oriel's MCP server from remote clients
  and hosted-AI connectors. Gated on optional authentication landing first.
- **Read-only & scoped MCP access.** A `--read-only` mode and per-tool allow-lists,
  so you can hand an MCP client a deliberately limited surface: logs and inspect
  only, no lifecycle. Works alongside the destructive grant, so you
  decide exactly what an AI can touch.
- **Audit log of AI actions.** A durable record of every tool call an MCP client or
  assistant makes: what ran, with which arguments, and when, so you can always
  see what an AI did to your containers. Your own UI clicks aren't logged.

## Next

- **In-app natural-language assistant.** A built-in command-palette mode that
  drives the same validated registry, for users who'd rather not wire up an
  external MCP client. Provider-agnostic: the base bundles no model.
- **MCP resources & prompts.** Expose container logs, inspect output, and Compose
  files as readable MCP **resources**, plus canned diagnostic **prompts**, so an
  AI can read a crashing container's logs as context, not just call tools.
- **Colima as a first-class MCP target.** Today an AI can only read `colima.status`.
  Add gated tools to manage the VM itself: `colima.start` / `stop` / `restart`,
  plus profiles and resources (CPU / memory / disk), so it can control the machine
  your containers run on, not just the containers.

## Later

- **homebrew-core submission**: `brew install oriel` with no tap, once the
  project clears Homebrew's notability bar.

## Demand- or sponsor-gated

These aren't planned on their own. They happen only if the gate below is met.

- **Windows support.** Parked unless demand clearly flares up. If you want it,
  upvote/comment on the [Windows tracking issue](https://github.com/ParadoxInfinite/oriel/issues).
  That demand is what moves it off this list.
- **Signed & notarized macOS binaries.** Requires Apple's **paid Developer
  Program (~$99/yr, recurring)**. Oriel won't fund an Apple subscription out of
  pocket. This only happens if sponsorship covers the ongoing cost. Until then
  the Homebrew cask strips the Gatekeeper quarantine attribute on install, so it
  works fine without it; the binaries just aren't Apple-blessed.

## Recently shipped

See the [CHANGELOG](CHANGELOG.md). Highlights: a **second review-driven hardening
pass** (atomic settings writes, masked connection-string/command secrets, prune
failures surfaced, modal-Escape fixes) (v0.4.2); **reliability + safety hardening**
(size-capped external fetches, atomic self-update, stronger secret masking,
clamped grant window) (v0.4.1); the **MCP server (`oriel mcp`) + read tools,
secret masking in inspect, and the time-boxed destructive grant** (v0.4.0);
themeable swappable editions, Compose discovery & deploy, CLI self-update +
`doctor`, reverse-proxy hosting, and Homebrew install (macOS).
