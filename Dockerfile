# syntax=docker/dockerfile:1
#
# Oriel as a container, primarily for running the MCP server:
#   docker run -i --rm -v /var/run/docker.sock:/var/run/docker.sock \
#     ghcr.io/paradoxinfinite/oriel
# That speaks MCP over stdio (the default command is `mcp`) against the host's
# Docker via the mounted socket. Colima-specific tools are inert in a container;
# everything else (containers, images, volumes, networks, compose) works.

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
RUN apk add --no-cache docker-cli docker-cli-compose ca-certificates
COPY --from=build /out/oriel /usr/local/bin/oriel
ENTRYPOINT ["oriel"]
CMD ["mcp"]
