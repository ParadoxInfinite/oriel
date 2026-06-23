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
		authMiddleware(ok, token).ServeHTTP(w, r)
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
