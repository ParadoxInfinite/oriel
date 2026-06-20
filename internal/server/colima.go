package server

import (
	"net/http"

	"github.com/ParadoxInfinite/oriel/internal/colima"
)

// handleColimaStatus returns engine status — the source of truth for the
// dashboard gauges and the running/stopped zero-state. With colima present it
// reports the VM; otherwise it reports the generic Docker engine.
func (s *Server) handleColimaStatus(w http.ResponseWriter, r *http.Request) {
	if colima.Installed() {
		st, err := colima.GetStatus(r.Context())
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, errorBody(err))
			return
		}
		st.Engine = "colima"
		writeJSON(w, http.StatusOK, st)
		return
	}

	info := s.docker.EngineInfo(r.Context())
	writeJSON(w, http.StatusOK, colima.Status{
		Engine:       "docker",
		Profile:      "docker",
		Running:      info.Reachable,
		Runtime:      "docker",
		Arch:         info.Architecture,
		CPU:          info.NCPU,
		Memory:       info.MemTotal,
		Driver:       info.Driver,
		DockerSocket: info.Host,
		Version:      info.ServerVersion,
	})
}

// handleColimaLifecycle streams the output of `colima start|stop|restart` as SSE
// so the UI can show live progress. Uses POST (it mutates); the frontend reads
// the streamed body rather than EventSource.
func (s *Server) handleColimaLifecycle(w http.ResponseWriter, r *http.Request) {
	action := r.PathValue("action")

	lines, errc, err := colima.Stream(r.Context(), action)
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

// errorBody is the standard error envelope for JSON responses.
func errorBody(err error) map[string]string {
	return map[string]string{"error": err.Error()}
}
