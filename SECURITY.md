# Security

## Reporting a vulnerability

Please **do not** open a public issue for security problems. Report them
privately via GitHub Security Advisories on
[ParadoxInfinite/oriel](https://github.com/ParadoxInfinite/oriel/security/advisories/new),
or email the maintainer. We'll acknowledge within a few days.

## Trust model — read this before exposing Oriel

Oriel has **no authentication**. Its entire security model is the network
boundary: by default it binds to **`127.0.0.1` only**, and it trusts anyone who
can reach that port as the local user.

This is deliberate — it's a local tool meant to sit next to your container
runtime — but it means **reaching Oriel is equivalent to root on the Docker
host.** Through the API you can:

- start/stop/remove any container, pull/remove images, prune the system;
- run `docker compose up` against any compose file on disk (`POST /api/stacks/up`);
- **browse and open any directory** the user can read (`GET /api/fs/list`,
  `POST /api/fs/open`);
- and, because the Docker daemon can mount the host filesystem and run
  privileged containers, escalate to full host control.

Two more powers worth calling out:

- **External themes** (`Settings → Themes → Load external theme`) dynamically
  `import()` a URL and run it as part of the app. Only load themes you trust —
  it's third-party JavaScript executing in your browser session.
- **The AI provider** (`ORIEL_PROVIDER_URL` / `Settings → AI`) POSTs your command
  text, the tool list, and live entity names to a URL you configure. Point it
  only at a resolver you trust. Every returned action is still re-validated
  against the tool registry before running, so a provider can't invoke an
  unknown tool or a non-existent entity — but it does receive that context.

## Remote access (private networks only)

Because there's no app-level password, **exposing Oriel safely means putting a
trusted network boundary in front of it.** Reach it over a **private overlay
network / VPN ONLY** — **Tailscale**, **ZeroTier**, **WireGuard**, or a
Nebula/Headscale-style mesh. The example below uses Tailscale, but the rule is
the same for any of them: Oriel stays on `127.0.0.1` and is reachable only over
the private interface, **never the public internet.**

**Example: Tailscale `serve` (tailnet-only).** Keep Oriel on `127.0.0.1` and
let Tailscale proxy to it, so it's reachable only by devices in *your* tailnet:

```sh
# Allow your tailnet hostname — the anti-rebinding guard blocks non-loopback
# Host headers by default, and `tailscale serve` forwards the .ts.net Host:
ORIEL_ALLOWED_HOSTS=<your-machine>.<tailnet>.ts.net ./oriel --no-open
tailscale serve --bg 4321         # serve to your tailnet over HTTPS
# reachable at https://<your-machine>.<tailnet>.ts.net/ from your devices
```

For a persistent install, bake the host in instead of exporting it each run:
`oriel service install --allowed-hosts <your-machine>.<tailnet>.ts.net`. For
sub-path mounts, nginx/Caddy/Traefik, and the full env-var reference, see
[docs/REVERSE-PROXY.md](docs/REVERSE-PROXY.md).

This is reasonably safe **if and only if**:

- you use `tailscale serve` (tailnet-scoped), **never `tailscale funnel`**
  (Funnel publishes to the public internet — with no auth that is an instant,
  total compromise of your Docker host; do not do it);
- your tailnet ACLs restrict that port to devices/users you trust — remember
  that *anyone who can reach Oriel has full host control*, so a shared node or a
  compromised tailnet device inherits that power;
- you accept that **Tailscale is the authentication.** There is no second factor
  inside Oriel.

**Avoid:** binding to `0.0.0.0` or a LAN IP, port-forwarding through a router, or
any reverse proxy without its own authentication. If you need auth in front of
Oriel, terminate it at the proxy (e.g. Tailscale, or an authenticating reverse
proxy on the same host) — Oriel itself won't ask for a password.

When in doubt, run it locally and reach it over SSH port-forwarding
(`ssh -L 4321:127.0.0.1:4321 host`), which keeps the same `127.0.0.1` trust model.
