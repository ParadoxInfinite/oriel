package main

import (
	"context"
	"errors"
	"io"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/ParadoxInfinite/oriel/internal/actions"
	"github.com/ParadoxInfinite/oriel/internal/docker"
	"github.com/ParadoxInfinite/oriel/internal/grant"
	"github.com/ParadoxInfinite/oriel/internal/mcp"
	"github.com/ParadoxInfinite/oriel/internal/secrets"
)

// runMCP serves the validated tool registry to an MCP client over stdio. It
// speaks JSON-RPC on stdin/stdout, so nothing may be logged to stdout — the SDK
// owns it. The process lives for the client session and exits on EOF.
//
// Two safety choices, because the client is an LLM with no human in the loop:
//   - Env values are masked with MaskAll regardless of the UI setting.
//   - Destructive tools are locked unless the user opened a grant window
//     (`oriel ai allow-destructive`); the MCP path never carries consent, so a
//     locked remove/prune returns a structured "how to unlock" error.
func runMCP(_ []string) error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	reg := actions.New(docker.New(), func() secrets.Mode { return secrets.MaskAll })
	reg.SetDestructiveWindow(grant.New().Active)

	if err := mcp.Serve(ctx, reg, version); err != nil && !cleanShutdown(err) {
		return err
	}
	return nil
}

// cleanShutdown reports whether err is just the client disconnecting or a
// signal — the normal way a stdio MCP server ends. The SDK wraps stdin EOF in
// an unexported jsonrpc2 "server is closing" error that errors.Is can't match,
// so fall back to the message for that one.
func cleanShutdown(err error) bool {
	if errors.Is(err, io.EOF) || errors.Is(err, context.Canceled) {
		return true
	}
	msg := err.Error()
	return strings.Contains(msg, "server is closing") || strings.Contains(msg, "EOF")
}
