package server

import (
	"net/http"
	"strconv"
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
// handleLive is the single push channel for all periodic UI data, so the client
// never polls. On connect it sends a full snapshot (history buffer, stats,
// status, self, outages); then every second it pushes the latest stats + history
// point, and every few seconds the slower-changing status/self/outages.
//
//	event "history" → []HistoryPoint  (full ~30-min buffer, once on connect)
//	event "stats"   → []docker.Stat   (per-container, 1s)
//	event "point"   → HistoryPoint    (newest aggregate sample, 1s)
//	event "status"  → statusResult    (engine status, ~5s)
//	event "self"    → selfStats       (gui footprint, ~5s)
//	event "outages" → []Outage        (downtime log, ~5s)
func (s *Server) handleLive(w http.ResponseWriter, r *http.Request) {
	sse, ok := newSSE(w)
	if !ok {
		return
	}
	ctx := r.Context()

	fast := func() {
		sse.send("stats", s.recorder.latestSnapshot())
		if p, ok := s.recorder.latestPoint(); ok {
			sse.send("point", p)
		}
	}
	slow := func() {
		sse.send("status", s.currentStatus(ctx))
		sse.send("self", s.currentSelf(ctx))
		sse.send("outages", s.recorder.outagesCopy())
	}

	sse.send("history", s.recorder.historyCopy())
	fast()
	slow()

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()
	const slowEvery = 5 // status/self/outages cadence, in fast ticks
	n := 0
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			fast()
			if n++; n%slowEvery == 0 {
				slow()
			}
		}
	}
}

// handleHistory returns the rolling ~30-minute aggregate CPU/memory series so
// the dashboard can render (and the browser can re-seed) the system pulse.
func (s *Server) handleHistory(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.recorder.historyCopy())
}

// handleOutages returns the persisted downtime log (retained ~30 days), each
// entry kind "down" (colima) or "offline" (Oriel).
func (s *Server) handleOutages(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.recorder.outagesCopy())
}

// handleContainerLogs follows one container's logs as SSE, seeded with the last
// 100 lines, then live. Older lines load on demand via handleContainerLogsBefore.
func (s *Server) handleContainerLogs(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	sse, ok := newSSE(w)
	if !ok {
		return
	}
	err := s.docker.StreamLogs(r.Context(), id, 100, true, "", func(stream, ts, line string) {
		sse.send("log", map[string]string{"stream": stream, "ts": ts, "line": line})
	})
	if err != nil && r.Context().Err() == nil {
		sse.send("error", errorBody(err))
	}
}

// handleContainerLogsBefore returns one historical batch of log lines ending just
// before ?before=<RFC3339Nano> (the cursor), newest-first window. ?limit caps the
// batch (default 100). Used to lazy-load older lines when scrolling back.
func (s *Server) handleContainerLogsBefore(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	until := r.URL.Query().Get("before")
	limit := 100
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 1000 {
			limit = n
		}
	}
	type logLine struct {
		Stream string `json:"stream"`
		TS     string `json:"ts"`
		Line   string `json:"line"`
	}
	lines := []logLine{}
	err := s.docker.StreamLogs(r.Context(), id, limit, false, until, func(stream, ts, line string) {
		lines = append(lines, logLine{Stream: stream, TS: ts, Line: line})
	})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, errorBody(err))
		return
	}
	writeJSON(w, http.StatusOK, lines)
}
