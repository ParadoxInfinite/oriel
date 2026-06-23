package main

import (
	"context"
	"errors"
	"flag"
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
	"github.com/ParadoxInfinite/oriel/internal/tools"
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
func runMCP(args []string) error {
	fs := flag.NewFlagSet("mcp", flag.ContinueOnError)
	readOnly := fs.Bool("read-only", false, "expose only read-only tools (no start/stop/remove/prune/...)")
	allow := fs.String("allow-tools", "", "expose only these tools (comma-separated names)")
	deny := fs.String("deny-tools", "", "exclude these tools (comma-separated names)")
	if err := fs.Parse(args); err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	reg := actions.New(docker.New(), func() secrets.Mode { return secrets.MaskAll })
	reg.SetDestructiveWindow(grant.New().Active)

	include := toolFilter(*readOnly, *allow, *deny)
	if err := mcp.Serve(ctx, reg, version, include); err != nil && !cleanShutdown(err) {
		return err
	}
	return nil
}

// toolFilter builds the predicate that scopes which tools the MCP server exposes:
// --read-only keeps only pure reads, --allow-tools is an exclusive whitelist, and
// --deny-tools removes names. A nil-equivalent (all flags empty) admits everything.
func toolFilter(readOnly bool, allow, deny string) func(*tools.Tool) bool {
	allowSet := csvSet(allow)
	denySet := csvSet(deny)
	return func(t *tools.Tool) bool {
		if readOnly && !t.ReadOnly {
			return false
		}
		if len(allowSet) > 0 && !allowSet[t.Name] {
			return false
		}
		return !denySet[t.Name]
	}
}

func csvSet(s string) map[string]bool {
	m := map[string]bool{}
	for _, x := range strings.Split(s, ",") {
		if x = strings.TrimSpace(x); x != "" {
			m[x] = true
		}
	}
	return m
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
