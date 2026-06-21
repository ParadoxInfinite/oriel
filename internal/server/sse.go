package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
)

// sseWriter is a minimal Server-Sent Events writer. SSE is used for all live
// data (lifecycle progress, docker events, stats, logs) — one channel each,
// cheap, and no websocket dependency. The mutex lets a keepalive goroutine ping
// alongside the data goroutine without racing the underlying writer.
type sseWriter struct {
	w  http.ResponseWriter
	f  http.Flusher
	mu sync.Mutex
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
	s := &sseWriter{w: w, f: f}
	// An immediate comment flushes the stream open so the client's onopen fires
	// even through a buffering proxy and even before any data arrives.
	s.ping()
	return s, true
}

// send writes a named event whose data is JSON-encoded v.
func (s *sseWriter) send(event string, v any) {
	b, err := json.Marshal(v)
	if err != nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if event != "" {
		fmt.Fprintf(s.w, "event: %s\n", event)
	}
	fmt.Fprintf(s.w, "data: %s\n\n", b)
	s.f.Flush()
}

// ping writes an SSE comment (ignored by EventSource) to flush the stream and
// keep an idle connection alive through proxy read timeouts.
func (s *sseWriter) ping() {
	s.mu.Lock()
	defer s.mu.Unlock()
	fmt.Fprint(s.w, ": ping\n\n")
	s.f.Flush()
}
