package server

import (
	"context"
	"net/http"

	"github.com/ParadoxInfinite/oriel/internal/colima"
)

// statusResult wraps engine status for the live stream: ok=false carries the
// reason colima was unreachable (so the UI shows "offline", not "stopped").
type statusResult struct {
	OK     bool           `json:"ok"`
	Status *colima.Status `json:"status,omitempty"`
	Error  string         `json:"error,omitempty"`
}

// currentStatus computes engine status once. Shared by the REST handler and the
// live stream so both report identically.
func (s *Server) currentStatus(ctx context.Context) statusResult {
	if colima.Installed() {
		st, err := colima.GetStatus(ctx)
		if err != nil {
			return statusResult{OK: false, Error: err.Error()}
		}
		st.Engine = "colima"
		return statusResult{OK: true, Status: &st}
	}
	info := s.docker.EngineInfo(ctx)
	return statusResult{OK: true, Status: &colima.Status{
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
	}}
}

// handleColimaStatus returns engine status — the source of truth for the
// dashboard gauges and the running/stopped zero-state.
func (s *Server) handleColimaStatus(w http.ResponseWriter, r *http.Request) {
	res := s.currentStatus(r.Context())
	if !res.OK {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": res.Error})
		return
	}
	writeJSON(w, http.StatusOK, res.Status)
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
