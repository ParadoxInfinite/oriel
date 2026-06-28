package server

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/ParadoxInfinite/oriel/internal/secrets"
	"github.com/ParadoxInfinite/oriel/internal/service"
)

func oneOf(v string, allowed ...string) bool {
	for _, a := range allowed {
		if v == a {
			return true
		}
	}
	return false
}

// handlePutConfig updates settings.json config that cannot hot-reload. Right now
// that's the reverse-proxy base path: it's baked into the served assets at boot,
// so we persist the new value and restart the (managed) service to apply it.
// Read it back from /api/self ("basePath").
func (s *Server) handlePutConfig(w http.ResponseWriter, r *http.Request) {
	var body struct {
		BasePath  *string `json:"basePath"`
		MaskEnv   *string `json:"maskEnv"`   // "all" | "sensitive" | "off"
		MaskLogs  *string `json:"maskLogs"`  // "sensitive" | "off" (UI only; MCP is always >= sensitive)
		EnvReveal *string `json:"envReveal"` // "off" | "local" | "remote"
		// Auth behavior knobs. Settable by any authenticated session (not local-only
		// like the token), since holding a session already implies full control; a
		// 0 means "use the default". The server clamps the effective values.
		SessionTTLMinutes *int    `json:"sessionTTLMinutes"`
		LoginFreeAttempts *int    `json:"loginFreeAttempts"`
		UpdateChannel     *string `json:"updateChannel"` // "stable" | "edge"
		ShellDisabled     *bool   `json:"shellDisabled"` // turn the container shell off/on
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid body"})
		return
	}
	// Only a base-path change requires a restart (it's baked into the served
	// assets at boot); the masking settings hot-reload (read per inspect request).
	var cur settings
	baseChanged := false
	channelChanged := false
	if err := updateSettings(func(c *settings) {
		if body.BasePath != nil {
			nb := strings.TrimSpace(*body.BasePath)
			baseChanged = nb != c.BasePath
			c.BasePath = nb
		}
		if body.MaskEnv != nil && oneOf(*body.MaskEnv, "all", "sensitive", "off") {
			c.MaskEnv = *body.MaskEnv
		}
		if body.MaskLogs != nil && oneOf(*body.MaskLogs, "sensitive", "off") {
			c.MaskLogs = *body.MaskLogs
		}
		if body.EnvReveal != nil && oneOf(*body.EnvReveal, "off", "local", "remote") {
			c.EnvReveal = *body.EnvReveal
		}
		if body.SessionTTLMinutes != nil && *body.SessionTTLMinutes >= 0 {
			c.SessionTTLMinutes = *body.SessionTTLMinutes // 0 = default; effective value clamped server-side
		}
		if body.LoginFreeAttempts != nil && *body.LoginFreeAttempts >= 0 {
			c.LoginFreeAttempts = *body.LoginFreeAttempts // 0 = default
		}
		if body.UpdateChannel != nil && oneOf(*body.UpdateChannel, "stable", "edge") {
			channelChanged = *body.UpdateChannel != c.UpdateChannel
			c.UpdateChannel = *body.UpdateChannel
		}
		if body.ShellDisabled != nil {
			c.ShellDisabled = *body.ShellDisabled
		}
		cur = *c
	}); err != nil {
		writeJSON(w, http.StatusInternalServerError, errorBody(err))
		return
	}
	// A channel switch invalidates the cached update check (it was for the old
	// channel), so the next check re-fetches against the new one.
	if channelChanged {
		updateMu.Lock()
		updateCache = nil
		updateMu.Unlock()
	}
	// Self-restart only if the base path changed and we're a managed service;
	// otherwise the caller must restart by hand to apply a base-path change.
	restarting := baseChanged && service.IsManaged()
	writeJSON(w, http.StatusOK, map[string]any{
		"basePath":          normalizeBase(cur.BasePath),
		"maskEnv":           string(secrets.ParseMode(cur.MaskEnv)),
		"maskLogs":          string(secrets.ParseLogMode(cur.MaskLogs)),
		"envReveal":         normReveal(cur.EnvReveal),
		"sessionTTLMinutes": cur.SessionTTLMinutes,
		"loginFreeAttempts": cur.LoginFreeAttempts,
		"updateChannel":     normChannel(cur.UpdateChannel),
		"shell":             !cur.ShellDisabled,
		"restarting":        restarting,
	})
	if restarting {
		go func() {
			time.Sleep(500 * time.Millisecond)
			_ = service.Restart()
		}()
	}
}
