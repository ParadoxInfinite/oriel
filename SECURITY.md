# Security

## Reporting a vulnerability

Please **do not** open a public issue for security problems. Report them
privately via GitHub Security Advisories on
[ParadoxInfinite/oriel](https://github.com/ParadoxInfinite/oriel/security/advisories/new),
or email the maintainer. We'll acknowledge within a few days.

## Trust model: read before exposing Oriel

By default Oriel has **no login**, and its baseline security model is the network
boundary: it binds to **`127.0.0.1` only** and trusts anyone who can reach that
port as the local user. There's an **optional bearer token** that gates
non-loopback and MCP-over-HTTP access (off by default; see
[Optional authentication](#optional-authentication)), but it's a single shared
secret, not a multi-user login. Unless noted, the model below assumes the
default: token off, loopback only.

This is deliberate: it's a local tool meant to sit next to your container
runtime. But it means **reaching Oriel is equivalent to root on the Docker
host.** Through the API you can:

- start/stop/remove any container, pull/remove images, prune the system;
- run `docker compose up` against any compose file on disk (`POST /api/stacks/up`);
- **browse and open any directory** the user can read (`GET /api/fs/list`,
  `POST /api/fs/open`);
- and, because the Docker daemon can mount the host filesystem and run
  privileged containers, escalate to full host control.

Two more powers worth calling out:

- **External themes** (`Settings → Themes → Load external theme`) dynamically
  `import()` a URL and run it as part of the app. Only load themes you trust.
  it's third-party JavaScript executing in your browser session.

- **Shared / multi-user hosts:** because anything that can reach the loopback
  port is trusted as the local user, **do not run Oriel on a shared, multi-user,
  or CI host.** On such a machine any other local user or job can reach
  `127.0.0.1` and gain the same root-equivalent Docker control, which is a local
  privilege-escalation path. Run it only where you trust every local user.

### What Oriel is not

It is **not a high-assurance, audited, or multi-tenant system, and must not be
treated as one.** There is no per-user identity, no audit log, no rate limiting,
and no least-privilege scoping for remote callers. Reaching the API at all means
root-equivalent host control. It has had no independent security assessment.
Suitable for a single trusted operator on a private network; **never** rely on it
as a security boundary for untrusted users, regulated/government workloads, or
public exposure without an independent audit and a hardened deployment.

## Optional authentication

Since v0.6.0 there's an opt-in bearer token that gates **non-loopback** `/api`
and **MCP-over-HTTP** access:

```sh
oriel config auth-token --generate   # generate a 256-bit token (printed once)
oriel config auth-token --clear      # turn the gate back off
```

What it does and doesn't do:

- **Loopback is always exempt.** The token applies only to remote or proxied
  callers; the local UI never needs it.
- **It's one shared secret**, constant-time compared. There's no per-user
  identity, login page, session, or RBAC (see [What Oriel is not](#what-oriel-is-not)).
- **Plain HTTP.** The token rides in cleartext, so keep Oriel behind a
  TLS-terminating reverse proxy on a private network. The token hardens that
  setup; it doesn't replace it.
- Changing or clearing the token takes effect on the next request, including on a
  running `oriel mcp --http`.

The network boundary is still the primary control. The token is defense-in-depth
on top of it, not a license to expose Oriel to the public internet.

## Remote access (private networks only)

The optional token helps, but **exposing Oriel safely still means putting a
trusted network boundary in front of it.** Reach it over a **private overlay
network / VPN ONLY**: **Tailscale**, **ZeroTier**, **WireGuard**, or a
Nebula/Headscale-style mesh. The example below uses Tailscale, but the rule is
the same for any of them: Oriel stays on `127.0.0.1` and is reachable only over
the private interface, **never the public internet.**

**Example: Tailscale `serve` (tailnet-only).** Keep Oriel on `127.0.0.1` and
let Tailscale proxy to it, so it's reachable only by devices in *your* tailnet:

```sh
./oriel --no-open                 # stays bound to 127.0.0.1:4321
tailscale serve --bg 4321         # serve to your tailnet over HTTPS
# Allow your tailnet hostname; the anti-rebinding guard blocks non-loopback
# Host headers by default, and `tailscale serve` forwards the .ts.net Host:
oriel remote allow <your-machine>.<tailnet>.ts.net
# reachable at https://<your-machine>.<tailnet>.ts.net/ from your devices
```

The allowed host is stored in `settings.json`, so it persists across restarts and
updates. For sub-path mounts and nginx/Caddy/Traefik, see
[docs/REVERSE-PROXY.md](docs/REVERSE-PROXY.md).

This is reasonably safe **if and only if**:

- you use `tailscale serve` (tailnet-scoped), **never `tailscale funnel`**
  (Funnel publishes to the public internet; with no auth that is an instant,
  total compromise of your Docker host; do not do it);
- your tailnet ACLs restrict that port to devices/users you trust. Remember
  that *anyone who can reach Oriel has full host control*, so a shared node or a
  compromised tailnet device inherits that power;
- you treat **the network (Tailscale) as the primary authentication.** Oriel's
  optional token can add a second factor for non-loopback callers, but don't lean
  on it alone.

**Avoid:** binding to `0.0.0.0` or a LAN IP, port-forwarding through a router, or
any reverse proxy without its own authentication. If you need strong auth in
front of Oriel, terminate it at the proxy (e.g. Tailscale, or an authenticating
reverse proxy on the same host). Oriel's own token is a single shared secret, not
a substitute for that.

When in doubt, run it locally and reach it over SSH port-forwarding
(`ssh -L 4321:127.0.0.1:4321 host`), which keeps the same `127.0.0.1` trust model.
