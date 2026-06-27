package server

import (
	"encoding/json"
	"net"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"sync"
)

// Oriel has no auth and is root-equivalent on the host, so the realistic remote
// attacker is a malicious web page the user visits: it can issue cross-site
// requests to http://127.0.0.1:4321, and DNS rebinding (evil.com → 127.0.0.1)
// defeats the loopback bind. The guard requires every /api request to (a) not be
// cross-site and (b) carry a loopback or explicitly-allowed Host header.
//
// Default is loopback-only. To reach Oriel over a private network (Tailscale,
// nginx, a domain), add the host(s) in Settings → Remote access, with
// `oriel remote allow <host>`, or in settings.json.

type hostGuard struct {
	mu    sync.RWMutex
	hosts map[string]bool // allowed non-loopback hosts (lowercased)
}

func normHost(h string) string { return strings.ToLower(strings.TrimSpace(h)) }

func buildHostSet(hosts []string) map[string]bool {
	m := map[string]bool{}
	for _, h := range hosts {
		if n := normHost(h); n != "" {
			m[n] = true
		}
	}
	return m
}

func newHostGuard() *hostGuard {
	return &hostGuard{hosts: buildHostSet(loadSettings().AllowedHosts)}
}

// set rebuilds the effective allow-set from the persisted host list.
func (g *hostGuard) set(persisted []string) {
	m := buildHostSet(persisted)
	g.mu.Lock()
	g.hosts = m
	g.mu.Unlock()
}

func (g *hostGuard) allows(host string) bool {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.hosts[normHost(host)]
}

// allowAPI decides whether an /api request may proceed.
func (s *Server) allowAPI(r *http.Request) bool {
	// Cross-site requests (CSRF / drive-by) are never allowed. same-origin,
	// same-site, and none (direct navigation) are fine. Fetch Metadata is the
	// primary signal, but a browser/context that omits Sec-Fetch-Site would
	// otherwise fall through to the Host check, so back it with an Origin check:
	// a present, cross-origin Origin is a CSRF attempt regardless of the header.
	if r.Header.Get("Sec-Fetch-Site") == "cross-site" || s.crossOrigin(r) {
		return false
	}
	host := r.Host
	if h, _, err := net.SplitHostPort(host); err == nil {
		host = h
	}
	// A loopback Host is the local UI, trusted, no token. But ONLY when the
	// request reached us directly: behind a reverse proxy the Host is whatever the
	// remote client sent, so a `Host: 127.0.0.1` would otherwise wave a remote
	// caller straight past the allow-list and the token. A proxied request always
	// carries a forwarding header the client can't strip, so we deny it the
	// loopback shortcut and make it clear the allow-list + token like anyone else.
	if isLoopbackHost(host) && !forwarded(r) {
		return true
	}
	// Non-loopback (or proxied): must be an allowed Host (anti-rebinding) AND, when
	// a token is configured, be authenticated, a bearer token (MCP/programmatic) or
	// a live session cookie (the browser GUI, after logging in with the token).
	if !s.guard.allows(host) {
		return false
	}
	// A couple of /api paths are reachable pre-auth (login, and the auth-status
	// the GUI polls to decide whether to show a login screen): still host-guarded
	// and cross-origin-checked above.
	if preAuthEndpoint(r) {
		return true
	}
	return s.authed(r)
}

// crossOrigin reports whether the request carries an Origin header from a
// different origin than the server itself. Browsers attach Origin to every
// state-changing cross-origin request (fetch, XHR, and form POSTs), so a present
// Origin whose host is neither loopback, nor this request's own Host, nor an
// allow-listed remote host is a cross-site (CSRF) attempt, even when the browser
// omitted Sec-Fetch-Site. An absent Origin (a non-browser client like curl, or a
// same-origin navigation) is not treated as cross-origin; those still face the
// Host check and, for remote callers, the token.
func (s *Server) crossOrigin(r *http.Request) bool {
	o := r.Header.Get("Origin")
	if o == "" {
		return false
	}
	u, err := url.Parse(o)
	if err != nil || u.Host == "" {
		return true // unparseable / opaque Origin: treat as hostile
	}
	oh := normHost(u.Hostname())
	if isLoopbackHost(oh) || oh == normHost(hostOnly(r.Host)) {
		return false // loopback UI or genuinely same-origin
	}
	return !s.guard.allows(oh) // an allow-listed remote origin is legitimate
}

// forwarded reports whether the request arrived through a reverse proxy, by the
// presence of a standard forwarding header the proxy adds on the hop to us. A
// remote client can send these, but it can't prevent the proxy from setting them,
// and a direct local request never has them, so this reliably distinguishes "the
// local UI" from "a remote caller wearing a forged loopback Host."
func forwarded(r *http.Request) bool {
	return r.Header.Get("X-Forwarded-For") != "" ||
		r.Header.Get("X-Forwarded-Host") != "" ||
		r.Header.Get("Forwarded") != ""
}

func isLoopbackHost(h string) bool {
	if strings.EqualFold(h, "localhost") {
		return true
	}
	ip := net.ParseIP(h)
	return ip != nil && ip.IsLoopback()
}

// localAdmin reports whether r may perform a local-only administrative change,
// setting the token or editing the allow-list. These are deliberately not
// delegatable to remote callers, even authenticated ones: who may reach Oriel is
// a local-machine decision. It requires a direct loopback request (loopback Host
// AND not proxied), so a forged loopback Host behind a proxy can't reach them.
func (s *Server) localAdmin(r *http.Request) bool {
	return isLoopbackHost(hostOnly(r.Host)) && !forwarded(r)
}

// normalizeHosts lowercases, trims, drops blanks, and dedupes a host list.
func normalizeHosts(in []string) []string {
	seen := map[string]bool{}
	out := []string{}
	for _, h := range in {
		if n := normHost(h); n != "" && !seen[n] {
			seen[n] = true
			out = append(out, n)
		}
	}
	sort.Strings(out)
	return out
}

// handleGetRemote returns the persisted allowed-host list (env-provided hosts are
// always in effect but not editable here).
func (s *Server) handleGetRemote(w http.ResponseWriter, r *http.Request) {
	hosts := loadSettings().AllowedHosts
	if hosts == nil {
		hosts = []string{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"hosts": hosts})
}

// handlePutRemote replaces the allowed-host list and updates the live guard.
// Local-only: a remote caller, even an authenticated one, must not be able to
// add its own host to the allow-list and entrench access.
func (s *Server) handlePutRemote(w http.ResponseWriter, r *http.Request) {
	if !s.localAdmin(r) {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "the allow-list can only be changed from the local machine"})
		return
	}
	var body struct {
		Hosts []string `json:"hosts"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid body"})
		return
	}
	hosts := normalizeHosts(body.Hosts)
	if err := updateSettings(func(st *settings) { st.AllowedHosts = hosts }); err != nil {
		writeJSON(w, http.StatusInternalServerError, errorBody(err))
		return
	}
	s.guard.set(hosts)
	writeJSON(w, http.StatusOK, map[string]any{"hosts": hosts})
}
