package mcp

import (
	"context"
	"net"
	"net/http"
	"time"

	"github.com/ParadoxInfinite/oriel/internal/settings"
	"github.com/ParadoxInfinite/oriel/internal/tools"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// ServeHTTP serves the MCP server over Streamable HTTP at addr until ctx is
// cancelled. token gates non-loopback callers — a loopback client (the same
// machine) is exempt, and an empty token disables the gate. Remote/hosted MCP
// clients connect here with `Authorization: Bearer <token>`. The caller must
// ensure a token is set before binding a non-loopback address (see mcp_cmd.go).
func ServeHTTP(ctx context.Context, addr string, reg *tools.Registry, version string, include func(*tools.Tool) bool, token string) error {
	srv := newServer(reg, version, include)
	mcpHandler := mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server { return srv }, nil)

	httpSrv := &http.Server{
		Addr:              addr,
		Handler:           authMiddleware(mcpHandler, token),
		ReadHeaderTimeout: 10 * time.Second,
	}
	go func() {
		<-ctx.Done()
		sctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = httpSrv.Shutdown(sctx)
	}()
	if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}

// authMiddleware requires the bearer token for non-loopback clients. Loopback
// (the same machine) is exempt; an empty token disables the gate. It reuses the
// single constant-time compare in internal/settings.
func authMiddleware(next http.Handler, token string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !clientLoopback(r) && !settings.TokenOK(settings.Bearer(r.Header.Get("Authorization")), token) {
			http.Error(w, "unauthorized: missing or invalid bearer token", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// clientLoopback reports whether the request came from the local machine.
func clientLoopback(r *http.Request) bool {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}
