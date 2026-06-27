package server

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"

	settingspkg "github.com/ParadoxInfinite/oriel/internal/settings"
)

// minTokenLen is the floor for a hand-set token. The generated token is 64 hex
// chars (256 bits); a user who types their own must still clear a bar that a
// rate-limit-free online guess can't reasonably cross.
const minTokenLen = 16

// authGate is the opt-in access-control layer for NON-loopback requests. The host
// allow-list (hostGuard) is anti-rebinding, not authentication, anyone who can
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
	return settingspkg.TokenOK(settingspkg.Bearer(r.Header.Get("Authorization")), token)
}

// matches reports whether provided equals the configured token (constant-time).
// Used by the GUI login, where the secret arrives in the request body rather than
// an Authorization header. False when auth is off, there's no secret to log into.
func (a *authGate) matches(provided string) bool {
	a.mu.RLock()
	token := a.token
	a.mu.RUnlock()
	return token != "" && settingspkg.TokenOK(provided, token)
}

// randomToken returns a 256-bit hex token from the OS CSPRNG. It returns the
// error rather than emitting on failure: a security token must never be
// best-effort, a short read or RNG failure must abort, not install a weak token.
func randomToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func hostOnly(host string) string {
	if h, _, err := net.SplitHostPort(host); err == nil {
		return h
	}
	return host
}

// handleGetAuth reports whether the token gate is on and whether this request is
// already authenticated (so the GUI knows to show a login screen). Never returns
// the token.
func (s *Server) handleGetAuth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"enabled":       s.auth.enabled(),
		"authenticated": s.authed(r),
		"localAdmin":    s.localAdmin(r), // may this caller change the token here?
	})
}

// handlePutAuth sets, generates, or clears the token. Local-only: who may
// authenticate is a local-machine decision, even an already-authenticated
// remote client can't rotate or disable the gate (localAdmin also rejects a
// proxied request wearing a forged loopback Host).
func (s *Server) handlePutAuth(w http.ResponseWriter, r *http.Request) {
	if !s.localAdmin(r) {
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
		t, err := randomToken()
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "could not generate a token"})
			return
		}
		token = t
	default:
		// A hand-set token must clear the strength floor; there's no rate limit
		// on the gate, so a short token would be online-guessable.
		if len([]rune(token)) < minTokenLen {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": fmt.Sprintf("token too weak: use at least %d characters, or --generate for a strong one", minTokenLen)})
			return
		}
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
