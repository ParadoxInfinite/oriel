package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/ParadoxInfinite/oriel/internal/actions"
	"github.com/ParadoxInfinite/oriel/internal/docker"
	"github.com/ParadoxInfinite/oriel/internal/grant"
	"github.com/ParadoxInfinite/oriel/internal/mcp"
	"github.com/ParadoxInfinite/oriel/internal/secrets"
	"github.com/ParadoxInfinite/oriel/internal/settings"
	"github.com/ParadoxInfinite/oriel/internal/tools"
)

// runMCP serves the validated tool registry to an MCP client over stdio. It
// speaks JSON-RPC on stdin/stdout, so nothing may be logged to stdout, the SDK
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
	httpAddr := fs.String("http", "", "serve over Streamable HTTP at this address (e.g. 127.0.0.1:8080) instead of stdio")
	if err := fs.Parse(args); err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	reg := actions.New(docker.New(), func() secrets.Mode { return secrets.MaskAll })
	reg.SetDestructiveWindow(grant.New().Active)
	include := toolFilter(*readOnly, *allow, *deny)

	if *httpAddr != "" {
		// Never expose MCP beyond loopback without a token, that's an open door
		// to the user's Docker. Force them to set one first.
		if exposedAddr(*httpAddr) && settings.Load().AuthToken == "" {
			return fmt.Errorf("refusing to serve MCP over HTTP on a non-loopback address (%s) without auth, set a token first:\n  oriel config auth-token --generate\n\nNote: this server speaks plain HTTP. The bearer token is sent in cleartext, so on any untrusted network put it behind a TLS-terminating reverse proxy (one that sets X-Forwarded-For) rather than binding it to the open address directly.", *httpAddr)
		}
		// Read the token fresh per request so a rotation/clear in the other
		// process (UI or `oriel config auth-token`) takes effect without a restart.
		return mcp.ServeHTTP(ctx, *httpAddr, reg, version, include, func() string { return settings.Load().AuthToken })
	}

	if err := mcp.Serve(ctx, reg, version, include); err != nil && !cleanShutdown(err) {
		return err
	}
	return nil
}

// exposedAddr reports whether addr binds beyond loopback (so it needs a token).
// A wildcard bind (":8080", "0.0.0.0", "::") or any non-loopback IP/host counts;
// 127.0.0.1 / ::1 / localhost do not.
func exposedAddr(addr string) bool {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		host = addr
	}
	if host == "" || host == "0.0.0.0" || host == "::" {
		return true
	}
	if ip := net.ParseIP(host); ip != nil {
		return !ip.IsLoopback()
	}
	return !strings.EqualFold(host, "localhost")
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
// signal, the normal way a stdio MCP server ends. The SDK wraps stdin EOF in
// an unexported jsonrpc2 "server is closing" error that errors.Is can't match,
// so fall back to the message for that one.
func cleanShutdown(err error) bool {
	if errors.Is(err, io.EOF) || errors.Is(err, context.Canceled) {
		return true
	}
	msg := err.Error()
	return strings.Contains(msg, "server is closing") || strings.Contains(msg, "EOF")
}
