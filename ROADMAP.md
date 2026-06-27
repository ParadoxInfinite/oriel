# Roadmap

Where Oriel's headed. Not a promise. Priorities move with feedback, and the best
way to push something up the list is to open an
[issue](https://github.com/ParadoxInfinite/oriel/issues) or a
[discussion](https://github.com/ParadoxInfinite/oriel/discussions).

## Up next

- **In-browser container shell.** An interactive `exec` terminal into a running
  container, straight from the UI, built on the existing exec-streaming seam.

## After that

- **Translations (i18n).** The keyboard/screen-reader accessibility pass has
  shipped; localization is the remaining half, so the UI isn't English-only.

## Further out

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

Highlights by release; the full history is in the [CHANGELOG](CHANGELOG.md).

- **v0.9.0** · Browser login for remote access (a session cookie over your private
  overlay), stable/edge update channels, the GUI as a loopback-only Linux
  container, and an audit log of AI actions.
- **v0.8.0** · Docker networks (create/inspect/connect/disconnect, in the UI and
  over MCP), a keyboard/screen-reader accessibility pass, container-log secret
  redaction, CSRF + CSP hardening, and the official MCP registry listing plus a
  multi-arch GHCR image.
- **v0.7.0** · A responsive, mobile-friendly UI, and removal of the second
  (Classic) edition.
- **v0.6.0** · Scoped & read-only MCP, MCP over HTTP behind a token, MCP resources
  & prompts, Colima VM control, a Docker-env helper, optional authentication, and
  removal of the in-app NL resolver.
- **v0.5.1** · Compose stacks drivable from ⌘K and MCP
  (start/stop/restart/down/alias), and Oriel-side renaming on any stack.
- **v0.5.0** · The run-command palette across all resources, ⌘K navigation, a
  shared nav seam in the theme SDK, and deprecation of the in-app NL resolver.
- **v0.4.2** · A second review-driven hardening pass (atomic settings writes,
  masked connection-string/command secrets, prune failures surfaced, modal-Escape
  fixes).
- **v0.4.1** · Reliability & safety hardening (size-capped external fetches, atomic
  self-update, stronger secret masking, clamped grant window).
- **v0.4.0** · The MCP server (`oriel mcp`) and read tools, secret masking in
  inspect, and the time-boxed destructive grant.
- **Earlier** · Themeable swappable editions, Compose discovery & deploy, CLI
  self-update and `doctor`, reverse-proxy hosting, and Homebrew install (macOS).
