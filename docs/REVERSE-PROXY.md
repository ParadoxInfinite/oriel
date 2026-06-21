# Reverse proxy & remote access

Oriel binds to `127.0.0.1` and has **no authentication** — its security model is
the network boundary (see [SECURITY.md](../SECURITY.md)). To reach it from another
machine you put a reverse proxy in front of it and tell Oriel to trust that path.

All config lives in **`settings.json`** (in your OS config dir, e.g.
`~/.config/oriel/settings.json`). You set it from the running instance — the UI,
or the CLI over loopback — so it persists across restarts, reinstalls, and
self-updates. Two settings matter for a proxy:

| settings.json key | Set it with | Purpose |
|---|---|---|
| `allowedHosts` | `oriel remote allow <host>` (or Settings → Remote access) | `Host` headers allowed to reach `/api`. Without this, non-loopback requests get **403** (anti-rebinding / CSRF guard). Applies immediately, no restart. |
| `basePath` | `oriel config base-path <path>` | Serve every asset + API under a sub-path, e.g. `/oriel`. Restarts a managed service to apply. |

> [!CAUTION]
> Allowing a host exposes a root-equivalent, **unauthenticated** Docker UI to
> anyone who can send that `Host` header. Only allow hosts on a network you trust
> (a private VPN/tailnet, or a proxy that adds TLS **and** a login). **Never expose
> Oriel directly to the public internet.**

## Set it up

After installing the service, configure the running instance:

```sh
oriel remote allow oriel.example.com     # allow your proxy's hostname (no restart)
oriel config base-path /oriel            # if mounting under a sub-path (restarts)
oriel doctor                             # verify Docker, base path, hosts, service
```

`oriel doctor` will tell you exactly what's missing — e.g. a sub-path set with no
allowed host, which is the most common cause of a 403 over the proxy.

### Why not just the in-app toggle?

Settings → Remote access edits `allowedHosts` at runtime — but only once the UI
can talk to `/api`. Behind a proxy on a non-loopback hostname, the UI's *own* first
API call carries that hostname and is 403'd, so the page can't load enough to fix
itself. Run `oriel remote allow <host>` **on the box** (loopback is always trusted)
to break the deadlock; after that the in-app toggle works for further changes.

### Migrating from env vars (pre-0.2)

Older versions configured this via `ORIEL_BASE_PATH` / `ORIEL_ALLOWED_HOSTS` /
`ORIEL_PROVIDER_URL`. These are deprecated: on first start of 0.2+, any that are
set are migrated into `settings.json` (and logged), then ignored. Remove them from
your service unit/environment once migrated — `settings.json` is the source now.

## nginx

Oriel streams all live data (logs, stats, events) over **SSE**, so the proxy must
not buffer. It tolerates the prefix being stripped or passed through, so either
`proxy_pass` style works.

```nginx
location /oriel/ {
    proxy_pass http://127.0.0.1:4321;

    # Required for SSE — without these the log/stat streams hang at 0 bytes:
    proxy_http_version 1.1;            # default 1.0 does not stream
    proxy_set_header Connection "";
    proxy_buffering off;
    proxy_cache off;
    gzip off;
    proxy_read_timeout 3600s;

    # Keep the public Host so it matches allowedHosts; forward the scheme.
    proxy_set_header Host $host;
    proxy_set_header X-Forwarded-Proto $scheme;
    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
}
```

Set `oriel config base-path /oriel` to match the `location`, and
`oriel remote allow <the public hostname>`.

## Any other proxy (Caddy, Traefik, …)

The same three requirements apply:

1. **Forward the public `Host`** (must match `allowedHosts`).
2. **Disable response buffering** for the SSE streams under `/api`.
3. If mounting under a sub-path, `oriel config base-path` to match (Oriel works
   whether or not the proxy strips the prefix).
