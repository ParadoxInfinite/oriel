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

// TestAllowAPI_CrossOrigin covers the Origin fallback that backs Sec-Fetch-Site:
// a present cross-origin Origin is rejected even without the Fetch-Metadata
// header, while the loopback UI's own Origin and an allow-listed remote origin
// still pass.
func TestAllowAPI_CrossOrigin(t *testing.T) {
	mk := func(token string) *Server {
		return &Server{
			guard: &hostGuard{hosts: map[string]bool{"oriel.example": true}},
			auth:  &authGate{token: token},
		}
	}
	req := func(host, origin, authHdr string) *http.Request {
		r, _ := http.NewRequest("POST", "/api/x", nil)
		r.Host = host
		if origin != "" {
			r.Header.Set("Origin", origin)
		}
		if authHdr != "" {
			r.Header.Set("Authorization", authHdr)
		}
		return r // deliberately no Sec-Fetch-Site, to exercise the Origin fallback
	}
	for _, c := range []struct {
		name, token, host, origin, authHdr string
		want                               bool
	}{
		{"loopback, no origin (curl/native)", "", "127.0.0.1:4321", "", "", true},
		{"loopback, own loopback origin", "", "127.0.0.1:4321", "http://127.0.0.1:4321", "", true},
		{"loopback, localhost origin", "", "127.0.0.1:4321", "http://localhost:4321", "", true},
		{"loopback, evil cross-origin denied", "", "127.0.0.1:4321", "https://evil.com", "", false},
		{"loopback, malformed origin denied", "", "127.0.0.1:4321", "://nope", "", false},
		{"remote allowed, same origin + token", "secret", "oriel.example", "https://oriel.example", "Bearer secret", true},
		{"remote allowed, evil origin denied", "secret", "oriel.example", "https://evil.com", "Bearer secret", false},
	} {
		if got := mk(c.token).allowAPI(req(c.host, c.origin, c.authHdr)); got != c.want {
			t.Errorf("%s: allowAPI=%v want %v", c.name, got, c.want)
		}
	}
}

// TestAllowAPI_ForwardedLoopback closes the proxy bypass: behind a reverse proxy
// the Host is attacker-controlled, so a forged `Host: 127.0.0.1` must NOT inherit
// the local-UI exemption, the forwarding header the proxy adds gives it away.
func TestAllowAPI_ForwardedLoopback(t *testing.T) {
	mk := func(token string) *Server {
		return &Server{
			guard: &hostGuard{hosts: map[string]bool{"oriel.example": true}},
			auth:  &authGate{token: token},
		}
	}
	// Direct local request, no forwarding header, keeps the exemption.
	if !mk("secret").allowAPI(authReq("127.0.0.1:4321", "", "")) {
		t.Fatal("direct loopback request must stay exempt (local UI)")
	}
	// Each standard forwarding header revokes the loopback shortcut: a forged
	// loopback Host then fails the allow-list (127.0.0.1 isn't listed) → denied,
	// even carrying the token, because the Host isn't an allowed remote host.
	for _, hdr := range []string{"X-Forwarded-For", "X-Forwarded-Host", "Forwarded"} {
		r := authReq("127.0.0.1", "Bearer secret", "")
		r.Header.Set(hdr, "203.0.113.7")
		if mk("secret").allowAPI(r) {
			t.Errorf("%s: proxied request with forged loopback Host must be denied", hdr)
		}
	}
	// A proxied request with the real (allowed) Host + token still works, proxies
	// are a supported remote path, the forwarding header alone doesn't block them.
	r := authReq("oriel.example", "Bearer secret", "")
	r.Header.Set("X-Forwarded-For", "203.0.113.7")
	if !mk("secret").allowAPI(r) {
		t.Error("proxied request with allowed Host + token must be allowed")
	}
}

// TestLocalAdmin gates the local-only admin ops (set/clear token, edit allow-list):
// a direct loopback request may; a proxied one wearing a forged loopback Host, or
// any remote Host, may not, even though it might otherwise pass the /api gate.
func TestLocalAdmin(t *testing.T) {
	s := &Server{}
	if !s.localAdmin(authReq("127.0.0.1:4321", "", "")) {
		t.Error("direct loopback must be allowed to administer")
	}
	if !s.localAdmin(authReq("localhost", "", "")) {
		t.Error("direct localhost must be allowed to administer")
	}
	// Forged loopback Host behind a proxy, the escalation path, must be denied.
	for _, hdr := range []string{"X-Forwarded-For", "X-Forwarded-Host", "Forwarded"} {
		req := authReq("127.0.0.1", "", "")
		req.Header.Set(hdr, "203.0.113.7")
		if s.localAdmin(req) {
			t.Errorf("%s: proxied forged-loopback Host must NOT administer", hdr)
		}
	}
	if s.localAdmin(authReq("oriel.example", "", "")) {
		t.Error("a remote Host must never administer")
	}
}

// TestRandomToken: 256-bit hex, fresh each call, no error.
func TestRandomToken(t *testing.T) {
	a, err := randomToken()
	if err != nil {
		t.Fatalf("randomToken errored: %v", err)
	}
	if len(a) != 64 {
		t.Errorf("token len = %d, want 64 hex chars (256 bits)", len(a))
	}
	if b, _ := randomToken(); a == b {
		t.Error("two generated tokens must differ")
	}
}
