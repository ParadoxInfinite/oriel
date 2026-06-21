package server

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/ParadoxInfinite/oriel/internal/service"
)

// handlePutConfig updates settings.json config that cannot hot-reload. Right now
// that's the reverse-proxy base path: it's baked into the served assets at boot,
// so we persist the new value and restart the (managed) service to apply it.
// Read it back from /api/self ("basePath").
func (s *Server) handlePutConfig(w http.ResponseWriter, r *http.Request) {
	var body struct {
		BasePath *string `json:"basePath"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid body"})
		return
	}
	cur := loadSettings()
	if body.BasePath != nil {
		cur.BasePath = strings.TrimSpace(*body.BasePath)
	}
	if err := saveSettings(cur); err != nil {
		writeJSON(w, http.StatusInternalServerError, errorBody(err))
		return
	}
	// A base-path change only takes effect on restart. Self-restart if we're a
	// managed service; otherwise the caller must restart the process by hand.
	managed := service.IsManaged()
	writeJSON(w, http.StatusOK, map[string]any{"basePath": normalizeBase(cur.BasePath), "restarting": managed})
	if managed {
		go func() {
			time.Sleep(500 * time.Millisecond)
			_ = service.Restart()
		}()
	}
}
