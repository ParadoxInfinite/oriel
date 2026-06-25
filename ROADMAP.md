# Roadmap

Where Oriel's headed. Not a promise. Priorities move with feedback, and the best
way to push something up the list is to open an
[issue](https://github.com/ParadoxInfinite/oriel/issues) or a
[discussion](https://github.com/ParadoxInfinite/oriel/discussions).

## Now

- **In-browser container shell.** An interactive `exec` terminal into a running
  container, straight from the UI, built on the existing exec-streaming seam.
- **Self-hostable on a server (single admin).** v0.6.0 added the opt-in bearer
  token. Next: let Oriel bind beyond loopback (refusing to start without a token)
  and add a simple admin **login plus session**, so one operator can run it on a
  home server or NAS and reach it from a browser over the LAN or a private network.
  Single-operator by design, **not multi-user: no per-user accounts, no RBAC**
  (that's a different product). An official Docker image follows after this lands.
  (See [SECURITY.md](SECURITY.md).)
- **Audit log of AI actions.** A durable record of every tool call an MCP client or
  assistant makes: what ran, with which arguments, and when, so you can always
  see what an AI did to your containers. Your own UI clicks aren't logged.

## Next

- **Mobile-friendly, responsive UI.** Make the dashboard and the resource views
  usable from a phone or tablet over the private network, not just a desktop browser.
- **Accessibility & translations.** A keyboard/screen-reader pass and i18n so the
  UI isn't English-only.

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

See the [CHANGELOG](CHANGELOG.md). Highlights: **scoped & read-only MCP, MCP over
HTTP behind a token, MCP resources & prompts, Colima VM control, a Docker-env
helper, optional authentication, and removal of the in-app NL resolver** (v0.6.0);
**compose stacks drivable from ⌘K +
MCP (start/stop/restart/down/alias) and Oriel-side renaming on any stack** (v0.5.1);
the **run-command palette across all resources + ⌘K navigation, a shared nav seam in
the theme SDK, and deprecation of the in-app NL resolver** (v0.5.0); a **second
review-driven hardening pass** (atomic
settings writes, masked connection-string/command secrets, prune failures surfaced,
modal-Escape fixes) (v0.4.2); **reliability + safety hardening**
(size-capped external fetches, atomic self-update, stronger secret masking,
clamped grant window) (v0.4.1); the **MCP server (`oriel mcp`) + read tools,
secret masking in inspect, and the time-boxed destructive grant** (v0.4.0);
themeable swappable editions, Compose discovery & deploy, CLI self-update +
`doctor`, reverse-proxy hosting, and Homebrew install (macOS).
