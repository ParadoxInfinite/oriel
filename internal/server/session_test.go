package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestSessionStore(t *testing.T) {
	st := newSessionStore()
	id, err := st.create()
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if len(id) != 64 {
		t.Errorf("session id = %d chars, want 64 (256-bit hex)", len(id))
	}
	if !st.valid(id) {
		t.Error("fresh session should be valid")
	}
	if st.valid("") || st.valid("nope") {
		t.Error("empty / unknown id must not be valid")
	}
	// Expiry: force the stored expiry into the past, then it must read invalid
	// and be dropped.
	st.mu.Lock()
	st.sessions[id] = time.Now().Add(-time.Minute)
	st.mu.Unlock()
	if st.valid(id) {
		t.Error("expired session should be invalid")
	}
	st.mu.Lock()
	_, present := st.sessions[id]
	st.mu.Unlock()
	if present {
		t.Error("expired session should be pruned on access")
	}
	// Delete.
	id2, _ := st.create()
	st.delete(id2)
	if st.valid(id2) {
		t.Error("deleted session should be invalid")
	}
}

func TestSessionSlidesExpiry(t *testing.T) {
	st := newSessionStore()
	id, _ := st.create()
	st.mu.Lock()
	st.sessions[id] = time.Now().Add(time.Minute) // about to expire
	st.mu.Unlock()
	if !st.valid(id) {
		t.Fatal("should still be valid")
	}
	st.mu.Lock()
	exp := st.sessions[id]
	st.mu.Unlock()
	if time.Until(exp) < sessionTTL-time.Hour {
		t.Errorf("valid() should have slid the expiry to ~now+TTL, got %v out", time.Until(exp))
	}
}

func TestLoginThrottle(t *testing.T) {
	tr := &loginThrottle{}
	// The first loginFreeAttempts are allowed even as they fail.
	for i := 0; i < loginFreeAttempts; i++ {
		if !tr.allowed() {
			t.Fatalf("attempt %d should be allowed (within the free window)", i+1)
		}
		tr.fail()
	}
	if tr.allowed() {
		t.Error("after the free window, a fresh attempt should be blocked by the cooldown")
	}
	tr.reset()
	if !tr.allowed() {
		t.Error("reset (a success) should clear the lockout")
	}
}

func TestAuthed(t *testing.T) {
	// Auth off: always authed, regardless of cookie/header.
	off := &Server{auth: &authGate{token: ""}, sessions: newSessionStore()}
	if !off.authed(httptest.NewRequest("GET", "/api/x", nil)) {
		t.Error("auth off must be authed")
	}

	st := newSessionStore()
	s := &Server{auth: &authGate{token: "s3cr3t-token-1234"}, sessions: st}

	if s.authed(httptest.NewRequest("GET", "/api/x", nil)) {
		t.Error("no credential must not be authed when token is on")
	}

	bearer := httptest.NewRequest("GET", "/api/x", nil)
	bearer.Header.Set("Authorization", "Bearer s3cr3t-token-1234")
	if !s.authed(bearer) {
		t.Error("valid bearer token must be authed")
	}

	id, _ := st.create()
	cookied := httptest.NewRequest("GET", "/api/x", nil)
	cookied.AddCookie(&http.Cookie{Name: sessionCookieName, Value: id})
	if !s.authed(cookied) {
		t.Error("valid session cookie must be authed")
	}

	bad := httptest.NewRequest("GET", "/api/x", nil)
	bad.AddCookie(&http.Cookie{Name: sessionCookieName, Value: "forged"})
	if s.authed(bad) {
		t.Error("forged session cookie must not be authed")
	}
}

func TestAllowAPI_LoginCarveout(t *testing.T) {
	st := newSessionStore()
	s := &Server{
		guard:    &hostGuard{hosts: map[string]bool{"oriel.example": true}},
		auth:     &authGate{token: "s3cr3t-token-1234"},
		sessions: st,
		loginRL:  &loginThrottle{},
	}
	req := func(method, path, host string) *http.Request {
		r := httptest.NewRequest(method, path, nil)
		r.Host = host
		return r
	}
	// Login is reachable on an allowed host without being authenticated.
	if !s.allowAPI(req("POST", "/api/login", "oriel.example")) {
		t.Error("POST /api/login must be reachable pre-auth on an allowed host")
	}
	// But it's still host-guarded: an un-allowed host is denied.
	if s.allowAPI(req("POST", "/api/login", "evil.example")) {
		t.Error("login on a non-allowed host must be denied")
	}
	// Any other endpoint without auth is denied.
	if s.allowAPI(req("GET", "/api/containers", "oriel.example")) {
		t.Error("non-login endpoint without auth must be denied")
	}
	// With a live session cookie, a normal endpoint is allowed.
	id, _ := st.create()
	r := req("GET", "/api/containers", "oriel.example")
	r.AddCookie(&http.Cookie{Name: sessionCookieName, Value: id})
	if !s.allowAPI(r) {
		t.Error("a valid session cookie must pass the guard")
	}
}

func TestHandleLogin(t *testing.T) {
	st := newSessionStore()
	s := &Server{auth: &authGate{token: "s3cr3t-token-1234"}, sessions: st, loginRL: &loginThrottle{}, base: "/"}

	// Wrong token → 401, no cookie, and a recorded failure.
	rec := httptest.NewRecorder()
	s.handleLogin(rec, httptest.NewRequest("POST", "/api/login", strings.NewReader(`{"token":"wrong"}`)))
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("wrong token: code = %d, want 401", rec.Code)
	}
	if len(rec.Result().Cookies()) != 0 {
		t.Error("wrong token must not set a session cookie")
	}

	// Correct token → 200 and an HttpOnly session cookie that validates.
	rec = httptest.NewRecorder()
	s.handleLogin(rec, httptest.NewRequest("POST", "/api/login", strings.NewReader(`{"token":"s3cr3t-token-1234"}`)))
	if rec.Code != http.StatusOK {
		t.Fatalf("correct token: code = %d, want 200", rec.Code)
	}
	var sc *http.Cookie
	for _, c := range rec.Result().Cookies() {
		if c.Name == sessionCookieName {
			sc = c
		}
	}
	if sc == nil {
		t.Fatal("correct token must set a session cookie")
	}
	if !sc.HttpOnly || sc.SameSite != http.SameSiteLaxMode {
		t.Errorf("cookie must be HttpOnly + SameSite=Lax, got HttpOnly=%v SameSite=%v", sc.HttpOnly, sc.SameSite)
	}
	if !st.valid(sc.Value) {
		t.Error("the issued session must be valid in the store")
	}
}

func TestHandleLogin_AuthOff(t *testing.T) {
	s := &Server{auth: &authGate{token: ""}, sessions: newSessionStore(), loginRL: &loginThrottle{}, base: "/"}
	rec := httptest.NewRecorder()
	s.handleLogin(rec, httptest.NewRequest("POST", "/api/login", strings.NewReader(`{}`)))
	if rec.Code != http.StatusOK {
		t.Errorf("auth-off login: code = %d, want 200 (nothing to log into)", rec.Code)
	}
}
