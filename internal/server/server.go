// Package server wires the HTTP router for Oriel: a small JSON REST
// surface for actions, SSE channels for live data, and the embedded frontend.
package server

import (
	"context"
	"encoding/json"
	"io/fs"
	"net/http"
	"os"
	"strings"

	"github.com/ParadoxInfinite/oriel/internal/actions"
	"github.com/ParadoxInfinite/oriel/internal/docker"
	"github.com/ParadoxInfinite/oriel/internal/provider"
	"github.com/ParadoxInfinite/oriel/internal/tools"
)

// Server holds shared dependencies and implements http.Handler.
type Server struct {
	mux      *http.ServeMux
	web      fs.FS
	base     string
	version  string
	docker   *docker.Client
	tools    *tools.Registry
	provider *provider.Provider
	recorder *recorder
	jobs     *jobManager
	guard    *hostGuard
	cancel   context.CancelFunc
}

// New constructs the router. web is the embedded frontend filesystem; version is
// the build version ("dev" for local builds, the release tag otherwise).
func New(web fs.FS, version string) *Server {
	dc := docker.New()
	s := &Server{
		mux:      http.NewServeMux(),
		web:      web,
		base:     normalizeBase(os.Getenv("ORIEL_BASE_PATH")),
		version:  version,
		docker:   dc,
		tools:    actions.New(dc),
		provider: provider.New(),
		recorder: newRecorder(dc),
		jobs:     newJobManager(),
		guard:    newHostGuard(),
	}
	// The env var wins as an explicit override; otherwise restore a URL the user
	// configured at runtime (Settings → AI) so it survives restarts.
	if s.provider.URL() == "" {
		if u := loadSettings().ProviderURL; u != "" {
			s.provider.SetURL(u)
		}
	}
	// Always-on metrics recorder for the live stream + 30-min history buffer.
	ctx, cancel := context.WithCancel(context.Background())
	s.cancel = cancel
	go s.recorder.run(ctx)
	s.routes()
	return s
}

// Close stops the recorder and persists the history + outage logs. Call on shutdown.
func (s *Server) Close() {
	if s.cancel != nil {
		s.cancel()
	}
	s.recorder.closeOpenOutage()
	s.recorder.flush()
	s.recorder.flushOutages()
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// When served under a subpath, strip it before routing so the mux and the
	// static handler see root-relative paths. This is tolerant: if the reverse
	// proxy already strips the prefix, the request arrives root-relative and the
	// TrimPrefix is a no-op — so it works with proxies that strip and those that
	// don't.
	if s.base != "/" {
		prefix := strings.TrimSuffix(s.base, "/") // e.g. "/oriel"
		if r.URL.Path == prefix {
			// Bare "/oriel" → "/oriel/" so the base resolves consistently.
			http.Redirect(w, r, s.base, http.StatusMovedPermanently)
			return
		}
		if rest, ok := strings.CutPrefix(r.URL.Path, prefix+"/"); ok {
			r.URL.Path = "/" + rest
		}
	}
	// Anti-rebinding / CSRF: API requests must be same-origin and arrive on a
	// loopback or explicitly-allowed Host. Static assets (the SPA) are harmless.
	if strings.HasPrefix(r.URL.Path, "/api/") && !s.allowAPI(r) {
		http.Error(w, "forbidden: cross-site request or untrusted Host — add it in Settings → Remote access, or set ORIEL_ALLOWED_HOSTS", http.StatusForbidden)
		return
	}
	s.mux.ServeHTTP(w, r)
}

func (s *Server) routes() {
	s.mux.HandleFunc("GET /api/health", s.handleHealth)
	s.mux.HandleFunc("GET /api/self", s.handleSelf)
	s.mux.HandleFunc("GET /api/remote", s.handleGetRemote)
	s.mux.HandleFunc("PUT /api/remote", s.handlePutRemote)
	s.mux.HandleFunc("GET /api/themes", s.handleListThemes)
	s.mux.HandleFunc("GET /api/themes/{file}", s.handleServeTheme)
	s.mux.HandleFunc("GET /api/update", s.handleUpdateCheck)
	s.mux.HandleFunc("POST /api/update/apply", s.handleUpdateApply)
	s.mux.HandleFunc("POST /api/update/restart", s.handleUpdateRestart)

	// Colima lifecycle.
	s.mux.HandleFunc("GET /api/colima/status", s.handleColimaStatus)
	s.mux.HandleFunc("POST /api/colima/{action}", s.handleColimaLifecycle)

	// Tool registry — the canonical action layer.
	s.mux.HandleFunc("GET /api/tools", s.handleTools)
	s.mux.HandleFunc("POST /api/invoke", s.handleInvoke)

	// Provider seam (dormant unless ORIEL_PROVIDER_URL is set, or configured
	// at runtime via Settings → AI).
	s.mux.HandleFunc("GET /api/provider", s.handleProviderStatus)
	s.mux.HandleFunc("POST /api/provider", s.handleSetProvider)
	s.mux.HandleFunc("POST /api/resolve", s.handleResolve)

	// Containers.
	s.mux.HandleFunc("GET /api/containers", s.handleContainers)
	s.mux.HandleFunc("GET /api/containers/{id}/inspect", s.handleContainerInspect)
	s.mux.HandleFunc("GET /api/containers/{id}/logs", s.handleContainerLogs)
	s.mux.HandleFunc("GET /api/containers/{id}/logs/before", s.handleContainerLogsBefore)

	// Compose stacks + discovery.
	s.mux.HandleFunc("GET /api/stacks", s.handleStacks)
	s.mux.HandleFunc("POST /api/stacks/up", s.handleStackUp)
	s.mux.HandleFunc("POST /api/stacks/{project}/{action}", s.handleStackAction)
	s.mux.HandleFunc("GET /api/discovery", s.handleGetDiscovery)
	s.mux.HandleFunc("PUT /api/discovery", s.handlePutDiscovery)
	s.mux.HandleFunc("GET /api/discovery/scan", s.handleScanDiscovery)
	s.mux.HandleFunc("GET /api/fs/list", s.handleFsList)
	s.mux.HandleFunc("POST /api/fs/open", s.handleFsOpen)

	// Images, volumes, networks.
	s.mux.HandleFunc("GET /api/images", s.handleImages)
	s.mux.HandleFunc("GET /api/images/search", s.handleImageSearch)
	s.mux.HandleFunc("GET /api/images/tags", s.handleImageTags)
	s.mux.HandleFunc("POST /api/images/pull", s.handleImagePull)
	s.mux.HandleFunc("GET /api/volumes", s.handleVolumes)
	s.mux.HandleFunc("GET /api/volumes/prune/preview", s.handleVolumesPrunePreview)
	s.mux.HandleFunc("GET /api/networks", s.handleNetworks)

	// System-wide disk usage.
	s.mux.HandleFunc("GET /api/system/df", s.handleSystemUsage)

	// Background operations: prune jobs run server-side (survive client refresh),
	// stream progress, and can be cancelled. List + stream + cancel let a client
	// re-attach to whatever it left running.
	s.mux.HandleFunc("GET /api/ops", s.handleListOps)
	s.mux.HandleFunc("GET /api/ops/{id}/stream", s.handleOpStream)
	s.mux.HandleFunc("POST /api/ops/{id}/cancel", s.handleCancelOp)
	s.mux.HandleFunc("POST /api/ops/system-prune", s.handleStartSystemPrune)
	s.mux.HandleFunc("POST /api/ops/image-prune", s.handleStartImagePrune)
	s.mux.HandleFunc("POST /api/ops/volume-prune", s.handleStartVolumePrune)

	// Live data: docker events (push, change-triggered) + one consolidated stream
	// for everything periodic (stats, history, status, self, outages) so the
	// client never polls. The plain GETs remain for one-off/manual use.
	s.mux.HandleFunc("GET /api/events", s.handleEvents)
	s.mux.HandleFunc("GET /api/live", s.handleLive)
	s.mux.HandleFunc("GET /api/history", s.handleHistory)
	s.mux.HandleFunc("GET /api/outages", s.handleOutages)

	// Everything else falls through to the embedded SPA.
	s.mux.Handle("/", s.staticHandler())
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// writeJSON is the single JSON response helper used across handlers.
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
