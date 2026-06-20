package server

import "net/http"

// handleSystemUsage reports what a system prune would reclaim (a confirm preview).
func (s *Server) handleSystemUsage(w http.ResponseWriter, r *http.Request) {
	u, err := s.docker.SystemUsage(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, errorBody(err))
		return
	}
	writeJSON(w, http.StatusOK, u)
}

// System prune itself now runs as a background job — see handleStartSystemPrune
// in ops.go, which survives client refresh and is cancellable.
