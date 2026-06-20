package server

import "net/http"

// handleStacks lists discovered compose stacks.
func (s *Server) handleStacks(w http.ResponseWriter, r *http.Request) {
	list, err := s.docker.ListStacks(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, errorBody(err))
		return
	}
	writeJSON(w, http.StatusOK, list)
}

// handleStackAction streams a compose action (up|down|start|stop|restart) as SSE.
func (s *Server) handleStackAction(w http.ResponseWriter, r *http.Request) {
	project := r.PathValue("project")
	action := r.PathValue("action")

	lines, errc, err := s.docker.StreamCompose(r.Context(), project, action)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorBody(err))
		return
	}
	sse, ok := newSSE(w)
	if !ok {
		return
	}
	for line := range lines {
		sse.send("line", map[string]string{"line": line})
	}
	result := map[string]any{"ok": true}
	if err := <-errc; err != nil {
		result["ok"] = false
		result["error"] = err.Error()
	}
	sse.send("done", result)
}
