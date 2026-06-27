package server

import (
	"net/http"
	"strconv"

	"github.com/ParadoxInfinite/oriel/internal/audit"
)

// handleAudit returns the recorded agent tool calls, newest first. `?limit=`
// caps the count (default 200, hard max in the audit package).
func (s *Server) handleAudit(w http.ResponseWriter, r *http.Request) {
	limit := 200
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			limit = n
		}
	}
	writeJSON(w, http.StatusOK, audit.Read(limit))
}
