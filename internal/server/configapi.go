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
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid body"})
		return
	}
	// Only a base-path change requires a restart (it's baked into the served
	// assets at boot); the masking settings hot-reload (read per inspect request).
	var cur settings
	baseChanged := false
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
		cur = *c
	}); err != nil {
		writeJSON(w, http.StatusInternalServerError, errorBody(err))
		return
	}
	// Self-restart only if the base path changed and we're a managed service;
	// otherwise the caller must restart by hand to apply a base-path change.
	restarting := baseChanged && service.IsManaged()
	writeJSON(w, http.StatusOK, map[string]any{
		"basePath":   normalizeBase(cur.BasePath),
		"maskEnv":    string(secrets.ParseMode(cur.MaskEnv)),
		"maskLogs":   string(secrets.ParseLogMode(cur.MaskLogs)),
		"envReveal":  normReveal(cur.EnvReveal),
		"restarting": restarting,
	})
	if restarting {
		go func() {
			time.Sleep(500 * time.Millisecond)
			_ = service.Restart()
		}()
	}
}
