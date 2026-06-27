BINARY := oriel
WEB := web
# Version stamped into the binary (shown in the UI footer + used by the update
# check). Local builds use the package.json version; releases inject the git tag.
VERSION := $(shell node -p "require('./$(WEB)/package.json').version" 2>/dev/null || echo dev)
# -s -w strip the symbol/debug tables; matches the release build so a local
# `make build` is the same lean, ~13 MB binary.
LDFLAGS := -s -w -X main.version=$(VERSION)

.PHONY: all build web dev dev-web run test clean tidy

all: build

## build: build frontend, embed it, produce the single binary
build: web
	go build -trimpath -ldflags "$(LDFLAGS)" -o $(BINARY) .
	@echo "built ./$(BINARY) ($(VERSION))"

## web: build the Svelte frontend into web/dist (embedded by the binary)
web:
	cd $(WEB) && npm run build

## run: build everything and run the binary
run: build
	./$(BINARY)

## service: build and install as a background service (starts on login)
service: build
	./$(BINARY) service install

## unservice: stop and remove the background service
unservice:
	./$(BINARY) service uninstall

## dev: run the Go backend (serves last-built UI); pair with `make dev-web`
dev:
	go run . --no-open

## dev-web: run Vite dev server with hot reload, proxying /api to the backend
dev-web:
	cd $(WEB) && npm run dev

## test: run Go unit tests
test:
	go test ./...

## tidy: tidy go modules
tidy:
	go mod tidy

## clean: remove build artifacts
clean:
	rm -f $(BINARY)
	rm -rf $(WEB)/dist
