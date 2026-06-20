package server

import (
	"encoding/json"
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

// handleSystemPrune runs a system prune; `volumes:true` also reclaims unused volumes.
func (s *Server) handleSystemPrune(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Volumes bool `json:"volumes"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	res, err := s.docker.SystemPrune(r.Context(), body.Volumes)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, errorBody(err))
		return
	}
	writeJSON(w, http.StatusOK, res)
}
