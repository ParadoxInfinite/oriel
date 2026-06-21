# Reverse proxy & remote access

Oriel binds to `127.0.0.1` and has **no authentication** — its security model is
the network boundary (see [SECURITY.md](../SECURITY.md)). To reach it from another
machine you put a reverse proxy in front of it and tell Oriel to trust that path.
Two settings do this, both read from the environment:

| Env var | Flag | Purpose |
|---|---|---|
| `ORIEL_BASE_PATH` | `--base-path` | Serve every asset + API under a sub-path, e.g. `/oriel`, so one proxy host can mount Oriel alongside other apps. |
| `ORIEL_ALLOWED_HOSTS` | `--allowed-hosts` | Comma-separated `Host` headers allowed to reach `/api` over the network. Without this, non-loopback requests get **403** (anti-rebinding / CSRF guard). |

> [!CAUTION]
> Allowing a host exposes a root-equivalent, **unauthenticated** Docker UI to
> anyone who can send that `Host` header. Only allow hosts on a network you trust
> (a private VPN/tailnet, or a proxy that adds TLS **and** a login). **Never expose
> Oriel directly to the public internet.**

## Why you can't just use the in-app toggle

Settings → Remote access can add allowed hosts at runtime — but only once the UI
can talk to `/api`. Behind a proxy on a non-loopback hostname, the UI's *own* first
API call already carries that hostname and is rejected with 403, so the page can
never load enough to fix itself. Bootstrap the **first** host via the env var /
installer below; after that the in-app toggle works for further changes.

## Setting it up

### New install — let the installer ask

`install.sh` prompts for the sub-path and allowed host (with a security warning)
when you opt into the background service. Unattended:

```sh
ORIEL_SERVICE=1 ORIEL_BASE_PATH=/oriel ORIEL_ALLOWED_HOSTS=oriel.example.com \
  curl -fsSL https://raw.githubusercontent.com/ParadoxInfinite/oriel/main/install.sh | sh
```

### Existing service — bake it into the unit

```sh
oriel service install --base-path /oriel --allowed-hosts oriel.example.com
```

This embeds the env vars into the systemd unit (or launchd plist) so they survive
restarts, reinstalls, and self-updates.

### Existing service — wipe-proof drop-in (systemd)

If you'd rather not re-run `service install`, a drop-in survives it (Oriel only
rewrites the main unit, never `*.service.d/`):

```sh
sudo mkdir -p /etc/systemd/system/oriel.service.d
printf '[Service]\nEnvironment=ORIEL_BASE_PATH=/oriel\nEnvironment=ORIEL_ALLOWED_HOSTS=oriel.example.com\n' \
  | sudo tee /etc/systemd/system/oriel.service.d/override.conf
sudo systemctl daemon-reload
sudo systemctl restart oriel
```

Verify: `systemctl show oriel -p Environment`.

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

    # Keep the public Host so it matches ORIEL_ALLOWED_HOSTS; forward the scheme.
    proxy_set_header Host $host;
    proxy_set_header X-Forwarded-Proto $scheme;
    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
}
```

Set `ORIEL_BASE_PATH=/oriel` to match the `location`, and add the proxy's public
hostname to `ORIEL_ALLOWED_HOSTS`.

## Any other proxy (Caddy, Traefik, …)

The same three requirements apply:

1. **Forward the public `Host`** (must match `ORIEL_ALLOWED_HOSTS`).
2. **Disable response buffering** for the SSE streams under `/api`.
3. If mounting under a sub-path, set `ORIEL_BASE_PATH` to match (Oriel works whether
   or not the proxy strips the prefix).
