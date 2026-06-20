package server

import (
	"net/http"
	"time"
)

// handleEvents streams docker events as SSE so the UI can refresh lists live.
func (s *Server) handleEvents(w http.ResponseWriter, r *http.Request) {
	events, errc, err := s.docker.StreamEvents(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, errorBody(err))
		return
	}
	sse, ok := newSSE(w)
	if !ok {
		return
	}
	for {
		select {
		case <-r.Context().Done():
			return
		case ev, open := <-events:
			if !open {
				return
			}
			sse.send("event", ev)
		case e := <-errc:
			if e != nil {
				sse.send("error", errorBody(e))
			}
			return
		}
	}
}

// handleStats broadcasts the recorder's latest per-container snapshot every
// second. Sampling happens once, centrally, in the recorder — this just relays.
func (s *Server) handleStats(w http.ResponseWriter, r *http.Request) {
	sse, ok := newSSE(w)
	if !ok {
		return
	}
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	sse.send("stats", s.recorder.latestSnapshot())
	for {
		select {
		case <-r.Context().Done():
			return
		case <-ticker.C:
			sse.send("stats", s.recorder.latestSnapshot())
		}
	}
}

// handleHistory returns the rolling ~30-minute aggregate CPU/memory series so
// the dashboard can render (and the browser can re-seed) the system pulse.
func (s *Server) handleHistory(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.recorder.historyCopy())
}

// handleOutages returns the persisted downtime log (retained ~30 days), each
// entry kind "down" (colima) or "offline" (colima-gui).
func (s *Server) handleOutages(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.recorder.outagesCopy())
}

// handleContainerLogs follows one container's logs as SSE.
func (s *Server) handleContainerLogs(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	sse, ok := newSSE(w)
	if !ok {
		return
	}
	err := s.docker.StreamLogs(r.Context(), id, 200, func(stream, line string) {
		sse.send("log", map[string]string{"stream": stream, "line": line})
	})
	if err != nil && r.Context().Err() == nil {
		sse.send("error", errorBody(err))
	}
}
