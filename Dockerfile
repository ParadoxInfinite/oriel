# syntax=docker/dockerfile:1
#
# Oriel as a container. Two uses:
#
#   MCP server (default command, stdio):
#     docker run -i --rm -v /var/run/docker.sock:/var/run/docker.sock \
#       ghcr.io/paradoxinfinite/oriel
#
#   GUI (Linux hosts) — share the host network so Oriel stays loopback-only and is
#   reached on the host's 127.0.0.1, then expose it over a private overlay
#   (Tailscale serve / a reverse proxy) exactly like the binary. NOT published with
#   -p; Oriel never binds beyond loopback. See docs/DAEMONS.md.
#     docker run -d --network host --name oriel \
#       -v /var/run/docker.sock:/var/run/docker.sock \
#       ghcr.io/paradoxinfinite/oriel --no-open
#
# Colima-specific tools are inert in a container; everything else (containers,
# images, volumes, networks, compose) works against the mounted socket.

# Build: cross-compile the static binary for the target arch on the native build
# arch. Go cross-compiles, so this stage needs no emulation.
FROM --platform=$BUILDPLATFORM golang:1.26-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
ARG TARGETARCH
ARG VERSION=dev
RUN CGO_ENABLED=0 GOOS=linux GOARCH=$TARGETARCH \
    go build -trimpath -ldflags "-s -w -X main.version=${VERSION}" -o /out/oriel .

# Runtime: the docker CLI + compose plugin (Oriel shells out to `docker compose`;
# everything else talks to the mounted socket through the Docker API).
FROM alpine:3.21
# Proves to the official MCP registry that this image belongs to the Oriel server
# entry (io.github.ParadoxInfinite/oriel); required to list the oci package.
LABEL io.modelcontextprotocol.server.name="io.github.ParadoxInfinite/oriel"
RUN apk add --no-cache docker-cli docker-cli-compose ca-certificates
COPY --from=build /out/oriel /usr/local/bin/oriel
# Lets the binary report it's containerized, so the GUI's update panel points at
# `docker pull` instead of an in-place self-update.
ENV ORIEL_CONTAINER=1
ENTRYPOINT ["oriel"]
CMD ["mcp"]
