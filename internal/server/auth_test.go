package server

import (
	"net/http"
	"testing"
)

func authReq(host, authHeader, fetchSite string) *http.Request {
	r, _ := http.NewRequest("GET", "/api/x", nil)
	r.Host = host
	if authHeader != "" {
		r.Header.Set("Authorization", authHeader)
	}
	if fetchSite != "" {
		r.Header.Set("Sec-Fetch-Site", fetchSite)
	}
	return r
}

func TestBearerToken(t *testing.T) {
	cases := map[string]string{
		"Bearer abc":   "abc",
		"bearer abc":   "abc", // scheme is case-insensitive
		"Bearer  abc ": "abc", // trimmed
		"Basic abc":    "",
		"abc":          "",
		"":             "",
		"Bearer":       "",
	}
	for in, want := range cases {
		if got := bearerToken(in); got != want {
			t.Errorf("bearerToken(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestAuthGateOK(t *testing.T) {
	if !(&authGate{}).ok(authReq("h", "", "")) {
		t.Error("auth off should allow everything")
	}
	on := &authGate{token: "secret"}
	for _, c := range []struct {
		name, header string
		want         bool
	}{
		{"correct token", "Bearer secret", true},
		{"wrong token", "Bearer nope", false},
		{"missing header", "", false},
		{"wrong scheme", "Basic secret", false},
		{"case-insensitive scheme", "bearer secret", true},
	} {
		if got := on.ok(authReq("h", c.header, "")); got != c.want {
			t.Errorf("%s: ok=%v want %v", c.name, got, c.want)
		}
	}
}

// TestAllowAPI_Auth is the real boundary: loopback exempt, remote needs an
// allowed Host AND (when on) the token, and auth-off preserves prior behavior.
func TestAllowAPI_Auth(t *testing.T) {
	mk := func(token string) *Server {
		return &Server{
			guard: &hostGuard{hosts: map[string]bool{"oriel.example": true}},
			auth:  &authGate{token: token},
		}
	}
	for _, c := range []struct {
		name, token, host, authHdr, fetchSite string
		want                                  bool
	}{
		{"loopback exempt with auth on", "secret", "127.0.0.1:4321", "", "", true},
		{"localhost exempt", "secret", "localhost", "", "", true},
		{"remote allowed + correct token", "secret", "oriel.example", "Bearer secret", "", true},
		{"remote allowed + no token denied", "secret", "oriel.example", "", "", false},
		{"remote allowed + wrong token denied", "secret", "oriel.example", "Bearer nope", "", false},
		{"remote not in allow-list denied", "secret", "evil.example", "Bearer secret", "", false},
		{"auth off: remote allowed + no token allowed", "", "oriel.example", "", "", true},
		{"cross-site always denied", "", "127.0.0.1", "", "cross-site", false},
	} {
		if got := mk(c.token).allowAPI(authReq(c.host, c.authHdr, c.fetchSite)); got != c.want {
			t.Errorf("%s: allowAPI=%v want %v", c.name, got, c.want)
		}
	}
}
