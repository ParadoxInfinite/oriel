package server

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"net"
	"net/http"
	"strings"
	"sync"
)

// authGate is the opt-in access-control layer for NON-loopback requests. The host
// allow-list (hostGuard) is anti-rebinding, not authentication — anyone who can
// reach an allowed host can otherwise use the API unauthenticated. When a token
// is configured, remote callers (including MCP-over-HTTP) must present it as a
// bearer token. Loopback is always exempt: the local UI is trusted, and a local
// attacker is already root-equivalent. Empty token = off (loopback-only as before).
type authGate struct {
	mu    sync.RWMutex
	token string
}

func newAuthGate() *authGate { return &authGate{token: loadSettings().AuthToken} }

func (a *authGate) set(token string) {
	a.mu.Lock()
	a.token = token
	a.mu.Unlock()
}

func (a *authGate) enabled() bool {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.token != ""
}

// ok reports whether r carries the configured token. True when auth is off.
// Constant-time compare so a wrong token can't be guessed byte-by-byte via timing.
func (a *authGate) ok(r *http.Request) bool {
	a.mu.RLock()
	token := a.token
	a.mu.RUnlock()
	if token == "" {
		return true
	}
	got := bearerToken(r.Header.Get("Authorization"))
	return got != "" && subtle.ConstantTimeCompare([]byte(got), []byte(token)) == 1
}

func bearerToken(h string) string {
	const p = "Bearer "
	if len(h) > len(p) && strings.EqualFold(h[:len(p)], p) {
		return strings.TrimSpace(h[len(p):])
	}
	return ""
}

func randomToken() string {
	b := make([]byte, 32)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func hostOnly(host string) string {
	if h, _, err := net.SplitHostPort(host); err == nil {
		return h
	}
	return host
}

// handleGetAuth reports whether the token gate is on. Never returns the token.
func (s *Server) handleGetAuth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"enabled": s.auth.enabled()})
}

// handlePutAuth sets, generates, or clears the token. Loopback-only: who may
// authenticate is a local decision — even an already-authenticated remote client
// can't rotate or disable the gate.
func (s *Server) handlePutAuth(w http.ResponseWriter, r *http.Request) {
	if !isLoopbackHost(hostOnly(r.Host)) {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "auth can only be changed from the local machine"})
		return
	}
	var body struct {
		Token    string `json:"token"`
		Generate bool   `json:"generate"`
		Clear    bool   `json:"clear"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid body"})
		return
	}
	token := strings.TrimSpace(body.Token)
	switch {
	case body.Clear:
		token = ""
	case body.Generate:
		token = randomToken()
	}
	if err := updateSettings(func(st *settings) { st.AuthToken = token }); err != nil {
		writeJSON(w, http.StatusInternalServerError, errorBody(err))
		return
	}
	s.auth.set(token)
	resp := map[string]any{"enabled": token != ""}
	if body.Generate {
		resp["token"] = token // returned once, only on generation
	}
	writeJSON(w, http.StatusOK, resp)
}
