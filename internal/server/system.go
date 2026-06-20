package server

import (
	"context"
	"net/http"
)

// handleSystemUsage reports what a system prune would reclaim (a confirm preview).
func (s *Server) handleSystemUsage(w http.ResponseWriter, r *http.Request) {
	u, err := s.docker.SystemUsage(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, errorBody(err))
		return
	}
	writeJSON(w, http.StatusOK, u)
}

// handleSystemPrune streams a system prune as SSE (one line per step). `?volumes=true`
// also reclaims unused volumes. The prune runs on a background context, so a client
// refresh/disconnect never aborts it mid-way — it finishes server-side regardless.
func (s *Server) handleSystemPrune(w http.ResponseWriter, r *http.Request) {
	includeVolumes := r.URL.Query().Get("volumes") == "true"
	sse, ok := newSSE(w)
	if !ok {
		return
	}
	res, err := s.docker.SystemPrune(context.Background(), includeVolumes, func(line string) {
		sse.send("line", map[string]string{"line": line})
	})
	result := map[string]any{"ok": err == nil}
	if err != nil {
		result["error"] = err.Error()
	} else {
		result["reclaimed"] = res.Reclaimed
	}
	sse.send("done", result)
}
