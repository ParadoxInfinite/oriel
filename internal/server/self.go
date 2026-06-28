package server

import (
	"context"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"

	"github.com/ParadoxInfinite/oriel/internal/secrets"
)

// selfStats reports this GUI's own resource footprint, so the dashboard can show
// what the tool itself costs.
type selfStats struct {
	Version    string `json:"version"`  // build version ("dev" for local builds)
	BasePath   string `json:"basePath"` // configured reverse-proxy base, "/" at root
	OS         string `json:"os"`       // server GOOS, clients label host actions by it
	RSS        int64  `json:"rss"`      // resident set size, bytes
	Goroutines int    `json:"goroutines"`
	HeapAlloc  int64  `json:"heapAlloc"` // Go heap in use, bytes
	MaskEnv    string `json:"maskEnv"`   // inspect env masking: "all" | "sensitive" | "off"
	MaskLogs   string `json:"maskLogs"`  // UI log masking: "sensitive" | "off"
	EnvReveal  string `json:"envReveal"` // reveal policy: "off" | "local" | "remote"

	SessionTTLMinutes int    `json:"sessionTTLMinutes"` // GUI session idle timeout (0 = default); raw setting, for the UI input
	LoginFreeAttempts int    `json:"loginFreeAttempts"` // login tries before backoff (0 = default)
	UpdateChannel     string `json:"updateChannel"`     // "stable" | "edge"
	Shell             bool   `json:"shell"`             // is the in-browser container shell available (UI gates the feature on this)
}

// currentSelf samples this process's footprint. Shared by the REST handler and
// the live stream.
func (s *Server) currentSelf(ctx context.Context) selfStats {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	cfg := loadSettings()
	return selfStats{
		Version:           s.version,
		BasePath:          s.base,
		OS:                runtime.GOOS,
		RSS:               processRSS(ctx),
		Goroutines:        runtime.NumGoroutine(),
		HeapAlloc:         int64(m.HeapAlloc),
		MaskEnv:           string(secrets.ParseMode(cfg.MaskEnv)),
		MaskLogs:          string(secrets.ParseLogMode(cfg.MaskLogs)),
		EnvReveal:         normReveal(cfg.EnvReveal),
		SessionTTLMinutes: cfg.SessionTTLMinutes,
		LoginFreeAttempts: cfg.LoginFreeAttempts,
		UpdateChannel:     normChannel(cfg.UpdateChannel),
		Shell:             !cfg.ShellDisabled,
	}
}

func (s *Server) handleSelf(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.currentSelf(r.Context()))
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
