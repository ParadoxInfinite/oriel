# Supported container runtimes

Oriel is a GUI for the **Docker Engine API**. It doesn't bundle a runtime. It
connects to one already on your machine. How it picks the connection
(`internal/docker/client.go`):

1. **If Colima is running**, Oriel talks to Colima's Docker socket (reported by
   `colima status`). This is the first-class path and also unlocks the VM
   **lifecycle controls** (start / stop / restart) on the dashboard.
2. **If Colima is installed but not started**, Oriel waits and retries rather
   than connecting to the wrong socket.
3. **Otherwise**, Oriel uses the standard Docker environment: `DOCKER_HOST` if
   set, else the platform's default socket (`/var/run/docker.sock`).

So anything that exposes the Docker Engine API over a unix socket (or
`DOCKER_HOST`) works. When the runtime isn't Colima, Oriel runs in **"Docker
engine" mode**: everything works except the VM lifecycle buttons (there's no VM
for Oriel to manage).

## Compatibility

| Runtime | Platform | Status | Notes |
|---|---|---|---|
| **Colima** | macOS, Linux | ✅ First-class | Auto-detected; adds VM start/stop/restart |
| **Docker Engine** (`dockerd`) | Linux | ✅ Supported | The native daemon at `/var/run/docker.sock` |
| **OrbStack** | macOS | ✅ Supported | Exposes a Docker-compatible socket |
| **Rancher Desktop** | macOS, Linux | ✅ Supported | Use the **dockerd (moby)** backend, not containerd |
| **Docker Desktop** | macOS, Linux | ✅ Supported | Works fine; just not the default on this project's dev machine |
| **Lima** (docker template) | macOS, Linux | ✅ Supported | Colima is built on Lima |
| **Podman** | macOS, Linux | ⚠️ Mostly | Via its Docker-compatible API (`podman system service` + `DOCKER_HOST`); a few endpoints differ |
| **Remote daemon** | any | ✅ Supported | Point `DOCKER_HOST=tcp://…` or `ssh://…` at it |
| **containerd / nerdctl** (directly) | n/a | ❌ Not supported | Not the Docker Engine API; needs a Docker-API shim |

The **Platform** column is where **Oriel itself** runs. It ships macOS and Linux
binaries only. Several of these runtimes also run on Windows; Oriel does not. A
Windows-hosted Docker daemon is still reachable as a **remote daemon**: point
`DOCKER_HOST` at it from a macOS or Linux machine (see below).

### Pointing Oriel at a non-Colima daemon

If Colima isn't installed, Oriel honors the usual Docker environment. For a
runtime that exposes a non-default socket, set `DOCKER_HOST` before launching:

```sh
# OrbStack example
DOCKER_HOST=unix://$HOME/.orbstack/run/docker.sock ./oriel

# Podman example
podman system service --time=0 &
DOCKER_HOST=unix://$(podman info -f '{{.Host.RemoteSocket.Path}}') ./oriel

# Remote daemon over SSH
DOCKER_HOST=ssh://user@host ./oriel
```
