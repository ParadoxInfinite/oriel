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
// cancelled. tokenFn supplies the gate token, read fresh per request, so a
// rotation or clear (via the UI or `oriel config auth-token`, in the other
// process) takes effect immediately, including revoking a leaked token without a
// restart. A loopback, non-proxied client is exempt; an empty token disables the
// gate. Remote/hosted clients connect with `Authorization: Bearer <token>`. The
// caller must ensure a token is set before binding a non-loopback address (see
// mcp_cmd.go).
func ServeHTTP(ctx context.Context, addr string, reg *tools.Registry, version string, include func(*tools.Tool) bool, tokenFn func() string) error {
	srv := newServer(reg, version, include)
	// nil options on purpose: it keeps the SDK's default browser protections ON,
	// which we rely on alongside the token gate. Do NOT set DisableLocalhostProtection
	//, that default 403s any request whose Host isn't loopback when we're bound to
	// loopback (DNS-rebinding defense, keyed off the real socket address), and the
	// default Content-Type: application/json requirement forces a CORS preflight a
	// cross-origin page can't satisfy. Together with authMiddleware, that covers the
	// remote-caller, rebinding, and cross-origin-POST vectors.
	mcpHandler := mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server { return srv }, nil)

	httpSrv := &http.Server{
		Addr:              addr,
		Handler:           authMiddleware(mcpHandler, tokenFn),
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
// local, direct one. The token is read via tokenFn on each request, so it always
// reflects the current setting. An empty token disables the gate. Reuses the
// single constant-time compare in internal/settings.
func authMiddleware(next http.Handler, tokenFn func() string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !localDirect(r) && !settings.TokenOK(settings.Bearer(r.Header.Get("Authorization")), tokenFn()) {
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
// caller past the token. A forwarding header, which the proxy adds and the real
// client can't strip, means the caller is remote and must present the token.
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
