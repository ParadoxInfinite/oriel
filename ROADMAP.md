# Roadmap

A living view of where Oriel is headed — not a contract. Priorities shift with
feedback; the best way to influence them is to open an
[issue](https://github.com/ParadoxInfinite/oriel/issues) or a
[discussion](https://github.com/ParadoxInfinite/oriel/discussions).

## Near-term

- **Optional authentication.** Today Oriel binds to `127.0.0.1` and only guards
  against DNS rebinding (host allow-list) — there is no login, so remote use is
  limited to trusted private networks. Add an **opt-in** token/password gate so
  Oriel can be exposed safely beyond loopback. Stays off by default; local use is
  unchanged. (See [SECURITY.md](SECURITY.md) for the current trust model.)
- **In-browser container shell.** An interactive `exec` terminal into a running
  container, straight from the UI — built on the existing exec-streaming seam.
  The feature container managers like lazydocker and Portainer are reached for.

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

See the [CHANGELOG](CHANGELOG.md). Highlights: themeable swappable editions,
Compose discovery & deploy, CLI self-update + `doctor`, reverse-proxy hosting,
and Homebrew install (macOS).
