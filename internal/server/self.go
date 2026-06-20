package server

import (
	"context"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
)

// selfStats reports this GUI's own resource footprint, so the dashboard can show
// what the tool itself costs.
type selfStats struct {
	RSS        int64 `json:"rss"` // resident set size, bytes
	Goroutines int   `json:"goroutines"`
	HeapAlloc  int64 `json:"heapAlloc"` // Go heap in use, bytes
}

func (s *Server) handleSelf(w http.ResponseWriter, r *http.Request) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	writeJSON(w, http.StatusOK, selfStats{
		RSS:        processRSS(r.Context()),
		Goroutines: runtime.NumGoroutine(),
		HeapAlloc:  int64(m.HeapAlloc),
	})
}

// processRSS reads our own resident memory via `ps` (portable across the
// macOS/Linux targets without a gopsutil-style dependency). Returns 0 on error.
func processRSS(ctx context.Context) int64 {
	out, err := exec.CommandContext(ctx, "ps", "-o", "rss=", "-p", strconv.Itoa(os.Getpid())).Output()
	if err != nil {
		return 0
	}
	kb, err := strconv.ParseInt(strings.TrimSpace(string(out)), 10, 64)
	if err != nil {
		return 0
	}
	return kb * 1024
}
