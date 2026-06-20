package server

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// sseWriter is a minimal Server-Sent Events writer. SSE is used for all live
// data (lifecycle progress, docker events, stats, logs) — one channel each,
// cheap, and no websocket dependency.
type sseWriter struct {
	w http.ResponseWriter
	f http.Flusher
}

// newSSE prepares the response for streaming. Returns ok=false if the
// ResponseWriter cannot flush (no streaming possible).
func newSSE(w http.ResponseWriter) (*sseWriter, bool) {
	f, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return nil, false
	}
	h := w.Header()
	h.Set("Content-Type", "text/event-stream")
	h.Set("Cache-Control", "no-cache")
	h.Set("Connection", "keep-alive")
	h.Set("X-Accel-Buffering", "no")
	w.WriteHeader(http.StatusOK)
	f.Flush()
	return &sseWriter{w: w, f: f}, true
}

// send writes a named event whose data is JSON-encoded v.
func (s *sseWriter) send(event string, v any) {
	b, err := json.Marshal(v)
	if err != nil {
		return
	}
	if event != "" {
		fmt.Fprintf(s.w, "event: %s\n", event)
	}
	fmt.Fprintf(s.w, "data: %s\n\n", b)
	s.f.Flush()
}
