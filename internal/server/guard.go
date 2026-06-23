package server

import (
	"encoding/json"
	"net"
	"net/http"
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
	// same-site, and none (direct navigation) are fine. A missing header (older
	// browsers, curl) falls through to the Host check.
	if r.Header.Get("Sec-Fetch-Site") == "cross-site" {
		return false
	}
	host := r.Host
	if h, _, err := net.SplitHostPort(host); err == nil {
		host = h
	}
	if isLoopbackHost(host) {
		return true // loopback is trusted; no token needed for the local UI
	}
	// Non-loopback: must be an allowed Host (anti-rebinding) AND, when a token is
	// configured, carry it (real access control for remote / proxied access).
	if !s.guard.allows(host) {
		return false
	}
	return s.auth.ok(r)
}

func isLoopbackHost(h string) bool {
	if strings.EqualFold(h, "localhost") {
		return true
	}
	ip := net.ParseIP(h)
	return ip != nil && ip.IsLoopback()
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
func (s *Server) handlePutRemote(w http.ResponseWriter, r *http.Request) {
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
