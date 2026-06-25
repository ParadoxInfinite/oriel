package mcp

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAuthMiddleware(t *testing.T) {
	ok := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })

	call := func(token, remoteAddr, authHeader string) int {
		r := httptest.NewRequest("POST", "/", nil)
		r.RemoteAddr = remoteAddr
		if authHeader != "" {
			r.Header.Set("Authorization", authHeader)
		}
		w := httptest.NewRecorder()
		authMiddleware(ok, func() string { return token }).ServeHTTP(w, r)
		return w.Code
	}

	for _, c := range []struct {
		name, token, remote, auth string
		want                      int
	}{
		{"loopback client exempt even with token set", "secret", "127.0.0.1:5555", "", http.StatusOK},
		{"remote + correct token", "secret", "10.0.0.2:5555", "Bearer secret", http.StatusOK},
		{"remote + missing token", "secret", "10.0.0.2:5555", "", http.StatusUnauthorized},
		{"remote + wrong token", "secret", "10.0.0.2:5555", "Bearer nope", http.StatusUnauthorized},
		{"no token configured: remote allowed", "", "10.0.0.2:5555", "", http.StatusOK},
	} {
		if got := call(c.token, c.remote, c.auth); got != c.want {
			t.Errorf("%s: status=%d want %d", c.name, got, c.want)
		}
	}
}

// TestAuthMiddleware_ProxyBypass closes the critical hole: when this endpoint is
// fronted by a same-host reverse proxy, every forwarded request arrives with a
// loopback RemoteAddr. Trusting that would skip the configured token for every
// remote caller. A forwarding header must force the token regardless of RemoteAddr.
func TestAuthMiddleware_ProxyBypass(t *testing.T) {
	ok := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })
	call := func(fwdHeader, auth string) int {
		r := httptest.NewRequest("POST", "/", nil)
		r.RemoteAddr = "127.0.0.1:5555" // the same-host proxy, always loopback
		r.Header.Set(fwdHeader, "203.0.113.7")
		if auth != "" {
			r.Header.Set("Authorization", auth)
		}
		w := httptest.NewRecorder()
		authMiddleware(ok, func() string { return "secret" }).ServeHTTP(w, r)
		return w.Code
	}
	for _, h := range []string{"X-Forwarded-For", "X-Forwarded-Host", "Forwarded"} {
		if got := call(h, ""); got != http.StatusUnauthorized {
			t.Errorf("%s + no token: status=%d want 401 (proxied request must not inherit loopback trust)", h, got)
		}
		if got := call(h, "Bearer secret"); got != http.StatusOK {
			t.Errorf("%s + correct token: status=%d want 200", h, got)
		}
		if got := call(h, "Bearer nope"); got != http.StatusUnauthorized {
			t.Errorf("%s + wrong token: status=%d want 401", h, got)
		}
	}
	// A genuinely direct loopback request (no forwarding header) stays exempt.
	r := httptest.NewRequest("POST", "/", nil)
	r.RemoteAddr = "127.0.0.1:5555"
	w := httptest.NewRecorder()
	authMiddleware(ok, func() string { return "secret" }).ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Errorf("direct loopback (no proxy): status=%d want 200", w.Code)
	}
}

// TestAuthMiddleware_LiveTokenRotation: the token is read per request, so a
// rotation (and especially a revocation by clearing) takes effect immediately on
// a running server, no restart, no window where the old token still works.
func TestAuthMiddleware_LiveTokenRotation(t *testing.T) {
	ok := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })
	current := "old-secret"
	h := authMiddleware(ok, func() string { return current })
	call := func(auth string) int {
		r := httptest.NewRequest("POST", "/", nil)
		r.RemoteAddr = "10.0.0.2:5555" // remote, token required
		if auth != "" {
			r.Header.Set("Authorization", auth)
		}
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		return w.Code
	}
	if call("Bearer old-secret") != http.StatusOK {
		t.Fatal("old token should work before rotation")
	}
	current = "new-secret" // rotate
	if call("Bearer old-secret") != http.StatusUnauthorized {
		t.Error("old token must stop working immediately after rotation")
	}
	if call("Bearer new-secret") != http.StatusOK {
		t.Error("new token should work immediately after rotation")
	}
	current = "" // revoke (clear), but a remote caller is still gated by exposedAddr at bind time
	if call("Bearer new-secret") != http.StatusOK {
		t.Error("clearing the token disables the gate (loopback-only model); documented behavior")
	}
}
