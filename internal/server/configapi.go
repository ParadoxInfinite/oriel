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
		EnvReveal *string `json:"envReveal"` // "off" | "local" | "remote"
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid body"})
		return
	}
	cur := loadSettings()
	// Only a base-path change requires a restart (it's baked into the served
	// assets at boot); the masking settings hot-reload (read per inspect request).
	baseChanged := false
	if body.BasePath != nil {
		nb := strings.TrimSpace(*body.BasePath)
		baseChanged = nb != cur.BasePath
		cur.BasePath = nb
	}
	if body.MaskEnv != nil && oneOf(*body.MaskEnv, "all", "sensitive", "off") {
		cur.MaskEnv = *body.MaskEnv
	}
	if body.EnvReveal != nil && oneOf(*body.EnvReveal, "off", "local", "remote") {
		cur.EnvReveal = *body.EnvReveal
	}
	if err := saveSettings(cur); err != nil {
		writeJSON(w, http.StatusInternalServerError, errorBody(err))
		return
	}
	// Self-restart only if the base path changed and we're a managed service;
	// otherwise the caller must restart by hand to apply a base-path change.
	restarting := baseChanged && service.IsManaged()
	writeJSON(w, http.StatusOK, map[string]any{
		"basePath":   normalizeBase(cur.BasePath),
		"maskEnv":    string(secrets.ParseMode(cur.MaskEnv)),
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
