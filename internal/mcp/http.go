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

// authMiddleware requires the bearer token for every caller except a genuinely
// local, direct one. An empty token disables the gate. Reuses the single
// constant-time compare in internal/settings.
func authMiddleware(next http.Handler, token string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !localDirect(r) && !settings.TokenOK(settings.Bearer(r.Header.Get("Authorization")), token) {
			http.Error(w, "unauthorized: missing or invalid bearer token", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// localDirect reports a request from the local machine that did NOT pass through
// a reverse proxy. RemoteAddr alone is not enough: a same-host proxy (the normal
// way this endpoint is exposed for TLS) makes RemoteAddr loopback for EVERY
// request it forwards, so trusting loopback would wave every proxied remote
// caller past the token. A forwarding header — which the proxy adds and the real
// client can't strip — means the caller is remote and must present the token.
func localDirect(r *http.Request) bool {
	return clientLoopback(r) && !forwarded(r)
}

// clientLoopback reports whether the TCP peer is the local machine.
func clientLoopback(r *http.Request) bool {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

// forwarded reports whether the request arrived through a reverse proxy, by the
// presence of a standard forwarding header the proxy adds on the hop to us.
func forwarded(r *http.Request) bool {
	return r.Header.Get("X-Forwarded-For") != "" ||
		r.Header.Get("X-Forwarded-Host") != "" ||
		r.Header.Get("Forwarded") != ""
}
