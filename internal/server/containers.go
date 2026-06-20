package server

import "net/http"

// handleContainers lists all containers (running and stopped).
func (s *Server) handleContainers(w http.ResponseWriter, r *http.Request) {
	list, err := s.docker.ListContainers(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, errorBody(err))
		return
	}
	writeJSON(w, http.StatusOK, list)
}

// handleContainerInspect returns the curated inspect payload for a detail panel.
func (s *Server) handleContainerInspect(w http.ResponseWriter, r *http.Request) {
	d, err := s.docker.InspectContainer(r.Context(), r.PathValue("id"))
	if err != nil {
		writeJSON(w, http.StatusNotFound, errorBody(err))
		return
	}
	writeJSON(w, http.StatusOK, d)
}
