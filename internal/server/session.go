package server

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"time"
)

// Browser login (session cookies)
//
// The bearer token gates non-loopback callers, but a browser can't present it:
// there's no way to attach an Authorization header to a top-level navigation, and
// EventSource (which drives the whole live UI) can't send headers at all. A
// cookie is the only credential both fetch and EventSource carry automatically.
//
// So the GUI authenticates by logging in once with the same token (POST
// /api/login), which mints a server-side session and sets an HttpOnly cookie;
// every later request carries it. MCP/programmatic clients keep using the token
// as a bearer header. One secret, two transports. This applies only on the
// non-loopback (overlay) path; local access stays exempt and the bind is
// unchanged. It is defense-in-depth on top of the network boundary, not a
// replacement for it, and remains a single shared secret (no per-user identity).

const (
	sessionCookieName = "oriel_session"
	// defaultSessionTTL is the idle timeout when settings.SessionTTLMinutes is
	// unset. A session slides forward on each use, so an active operator stays
	// logged in and an abandoned one lapses. In-memory by design (single operator,
	// low stakes), so a restart just asks for a fresh login.
	defaultSessionTTL = 7 * 24 * time.Hour
	minSessionTTL     = time.Minute
	// maxSessionTTL caps the configured idle timeout. A long session is a
	// convenience; an unbounded one would let a stolen cookie persist
	// indefinitely, so even an authenticated caller (who can set this knob) can't
	// turn a session into a permanent foothold.
	maxSessionTTL = 30 * 24 * time.Hour
)

// effectiveSessionTTL is the sliding idle timeout: settings.SessionTTLMinutes
// (hot-reloaded) with a 7-day default, clamped to [1 minute, 30 days].
func effectiveSessionTTL(cfg settings) time.Duration {
	if cfg.SessionTTLMinutes <= 0 {
		return defaultSessionTTL
	}
	d := time.Duration(cfg.SessionTTLMinutes) * time.Minute
	switch {
	case d < minSessionTTL:
		return minSessionTTL
	case d > maxSessionTTL:
		return maxSessionTTL
	default:
		return d
	}
}

// settingsSessionTTL / settingsFreeAttempts read the live settings; passed to the
// store and throttle so a settings.json edit takes effect without a restart.
func settingsSessionTTL() time.Duration { return effectiveSessionTTL(loadSettings()) }
func settingsFreeAttempts() int         { return effectiveLoginFreeAttempts(loadSettings()) }

// sessionStore is the in-memory set of live browser sessions: random id → expiry.
type sessionStore struct {
	mu       sync.Mutex
	sessions map[string]time.Time
	ttl      func() time.Duration // sliding idle timeout, read per use so settings hot-reload
}

func newSessionStore(ttl func() time.Duration) *sessionStore {
	return &sessionStore{sessions: map[string]time.Time{}, ttl: ttl}
}

// create mints a session and returns its id. crypto/rand, 256-bit; it returns the
// error rather than a weak id, a session token must never be best-effort.
func (s *sessionStore) create() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	id := hex.EncodeToString(b)
	now := time.Now()
	s.mu.Lock()
	s.prune(now)
	s.sessions[id] = now.Add(s.ttl())
	s.mu.Unlock()
	return id, nil
}

// valid reports whether id is a live session, sliding its expiry forward on a hit
// so an active operator isn't logged out mid-use. An expired id is dropped.
func (s *sessionStore) valid(id string) bool {
	if id == "" {
		return false
	}
	now := time.Now()
	s.mu.Lock()
	defer s.mu.Unlock()
	exp, ok := s.sessions[id]
	if !ok || !now.Before(exp) {
		delete(s.sessions, id) // no-op if absent
		return false
	}
	s.sessions[id] = now.Add(s.ttl())
	return true
}

func (s *sessionStore) delete(id string) {
	if id == "" {
		return
	}
	s.mu.Lock()
	delete(s.sessions, id)
	s.mu.Unlock()
}

// prune drops expired sessions. Caller holds the lock.
func (s *sessionStore) prune(now time.Time) {
	for id, exp := range s.sessions {
		if !now.Before(exp) {
			delete(s.sessions, id)
		}
	}
}

const (
	defaultLoginFreeAttempts = 5               // failures before backoff starts
	loginBackoffBase         = 2 * time.Second // doubles per failure past the free window
	loginBackoffMax          = 15 * time.Minute
)

// effectiveLoginFreeAttempts is settings.LoginFreeAttempts (hot-reloaded) with a
// default of 5 and a floor of 1.
func effectiveLoginFreeAttempts(cfg settings) int {
	if cfg.LoginFreeAttempts < 1 {
		return defaultLoginFreeAttempts
	}
	return cfg.LoginFreeAttempts
}

// loginThrottle is a single global brute-force guard on /api/login. One counter
// suffices for a single-admin tool (a per-IP map would key on the proxy anyway).
// The first `free()` failures are unthrottled; after that each imposes an
// exponentially growing cooldown, reset on any success. The backoff base and cap
// stay fixed; the free-attempt count comes from settings.
type loginThrottle struct {
	mu    sync.Mutex
	fails int
	last  time.Time
	free  func() int // unthrottled attempts before backoff, read per use
}

func newLoginThrottle(free func() int) *loginThrottle { return &loginThrottle{free: free} }

// allowed reports whether an attempt may proceed now (the cooldown has elapsed).
func (t *loginThrottle) allowed() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.fails < t.free() {
		return true
	}
	return time.Since(t.last) >= t.cooldown()
}

// cooldown is the wait imposed by the current failure count. Caller holds the lock.
func (t *loginThrottle) cooldown() time.Duration {
	d := loginBackoffBase << (t.fails - t.free())
	if d <= 0 || d > loginBackoffMax { // <=0 catches the shift overflowing int64
		return loginBackoffMax
	}
	return d
}

func (t *loginThrottle) fail() {
	t.mu.Lock()
	t.fails++
	t.last = time.Now()
	t.mu.Unlock()
}

func (t *loginThrottle) reset() {
	t.mu.Lock()
	t.fails = 0
	t.mu.Unlock()
}

// authed reports whether r is authenticated for the API. Auth-off is always
// authed; direct loopback is exempt (same rule as the API guard, so the local UI
// is never asked to log in); otherwise a valid bearer token (programmatic / MCP)
// or a live session cookie (the browser GUI, post-login) passes.
func (s *Server) authed(r *http.Request) bool {
	if !s.auth.enabled() {
		return true
	}
	if isLoopbackHost(hostOnly(r.Host)) && !forwarded(r) {
		return true
	}
	if s.auth.ok(r) { // bearer token in the Authorization header
		return true
	}
	if c, err := r.Cookie(sessionCookieName); err == nil {
		return s.sessions.valid(c.Value)
	}
	return false
}

// preAuthEndpoint lists the /api paths reachable before authentication: the login
// POST, and the GET /api/auth the GUI polls to learn whether it must log in. Both
// are still host-guarded and cross-origin-checked by allowAPI.
func preAuthEndpoint(r *http.Request) bool {
	if r.URL.Path == "/api/login" && r.Method == http.MethodPost {
		return true
	}
	if r.URL.Path == "/api/auth" && r.Method == http.MethodGet {
		return true
	}
	return false
}

// handleLogin validates the configured token from the request body and, on a
// match, starts a browser session (an HttpOnly cookie). The host guard and
// cross-origin check still gate it (see allowAPI); brute force is throttled.
func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	// Auth off: nothing to log into, the GUI already works.
	if !s.auth.enabled() {
		writeJSON(w, http.StatusOK, map[string]any{"authenticated": true})
		return
	}
	if !s.loginRL.allowed() {
		writeJSON(w, http.StatusTooManyRequests, map[string]string{"error": "too many attempts; wait a bit and try again"})
		return
	}
	var body struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid body"})
		return
	}
	if !s.auth.matches(strings.TrimSpace(body.Token)) {
		s.loginRL.fail()
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "incorrect token"})
		return
	}
	s.loginRL.reset()
	id, err := s.sessions.create()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "could not start a session"})
		return
	}
	http.SetCookie(w, s.sessionCookie(r, id, effectiveSessionTTL(loadSettings())))
	writeJSON(w, http.StatusOK, map[string]any{"authenticated": true})
}

// handleLogout ends the current session and clears the cookie.
func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie(sessionCookieName); err == nil {
		s.sessions.delete(c.Value)
	}
	http.SetCookie(w, s.sessionCookie(r, "", -1))
	writeJSON(w, http.StatusOK, map[string]any{"authenticated": false})
}

// sessionCookie builds the session cookie. HttpOnly + SameSite=Lax always;
// Secure only when the request reached us over HTTPS, a plain-HTTP reverse proxy
// on a private mesh can't send a Secure cookie, and forcing it would break login
// there. Path is the configured base so a subpath mount scopes correctly. A
// non-positive ttl produces a deletion cookie.
func (s *Server) sessionCookie(r *http.Request, value string, ttl time.Duration) *http.Cookie {
	path := s.base
	if path == "" {
		path = "/"
	}
	maxAge := int(ttl / time.Second)
	if ttl <= 0 {
		maxAge = -1
	}
	return &http.Cookie{
		Name:     sessionCookieName,
		Value:    value,
		Path:     path,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   requestIsHTTPS(r),
		MaxAge:   maxAge,
	}
}

// requestIsHTTPS reports whether the request reached us over TLS, directly or via
// a terminating proxy that set X-Forwarded-Proto.
func requestIsHTTPS(r *http.Request) bool {
	return r.TLS != nil || strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https")
}
