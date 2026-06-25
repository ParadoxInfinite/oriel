package server

import (
	"net/http"
	"strconv"
)

func (s *Server) handleImages(w http.ResponseWriter, r *http.Request) {
	list, err := s.docker.ListImages(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, errorBody(err))
		return
	}
	writeJSON(w, http.StatusOK, list)
}

func (s *Server) handleVolumes(w http.ResponseWriter, r *http.Request) {
	list, err := s.docker.ListVolumes(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, errorBody(err))
		return
	}
	writeJSON(w, http.StatusOK, list)
}

// handleVolumesPrunePreview lists the unused volumes a prune would remove, with
// sizes, so the UI can show and let the user deselect before pruning.
func (s *Server) handleVolumesPrunePreview(w http.ResponseWriter, r *http.Request) {
	list, err := s.docker.PruneableVolumes(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, errorBody(err))
		return
	}
	writeJSON(w, http.StatusOK, list)
}

func (s *Server) handleNetworks(w http.ResponseWriter, r *http.Request) {
	list, err := s.docker.ListNetworks(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, errorBody(err))
		return
	}
	writeJSON(w, http.StatusOK, list)
}

// handleNetworkInspect returns the curated network detail (addressing + attached
// containers) for the detail panel. Mutations (create, connect, disconnect) go
// through /api/invoke like every other tool; this is the only read that needs its
// own route beyond the list.
func (s *Server) handleNetworkInspect(w http.ResponseWriter, r *http.Request) {
	d, err := s.docker.InspectNetwork(r.Context(), r.PathValue("id"))
	if err != nil {
		writeJSON(w, http.StatusNotFound, errorBody(err))
		return
	}
	writeJSON(w, http.StatusOK, d)
}

// handleImageSearch proxies a registry search so the pull dialog can offer live
// suggestions without hitting CORS. `source` selects the registry: Docker Hub
// (via the daemon) or Quay.io (via its public API). Registries without a public
// search API are pull-only and never reach here.
func (s *Server) handleImageSearch(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	if len(q) < 2 {
		writeJSON(w, http.StatusOK, []any{})
		return
	}
	limit := 25
	if v, err := strconv.Atoi(r.URL.Query().Get("limit")); err == nil && v > 0 && v <= 100 {
		limit = v
	}

	var (
		list any
		err  error
	)
	switch r.URL.Query().Get("source") {
	case "quay":
		list, err = searchQuay(r.Context(), q, limit)
	case "ecr":
		list, err = searchECR(r.Context(), q, limit)
	default:
		list, err = s.docker.SearchImages(r.Context(), q, limit)
	}
	if err != nil {
		writeJSON(w, http.StatusBadGateway, errorBody(err))
		return
	}
	writeJSON(w, http.StatusOK, list)
}

// handleImageTags lists recent tags for a repo, for the pull dialog's tag picker.
func (s *Server) handleImageTags(w http.ResponseWriter, r *http.Request) {
	repo := r.URL.Query().Get("repo")
	if repo == "" {
		writeJSON(w, http.StatusOK, []string{})
		return
	}
	limit := 30
	if v, err := strconv.Atoi(r.URL.Query().Get("limit")); err == nil && v > 0 && v <= 100 {
		limit = v
	}
	tags, err := listTags(r.Context(), r.URL.Query().Get("source"), repo, limit)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, errorBody(err))
		return
	}
	writeJSON(w, http.StatusOK, tags)
}

// handleImagePull streams docker pull progress as SSE. POST since it mutates;
// the frontend reads the streamed body.
func (s *Server) handleImagePull(w http.ResponseWriter, r *http.Request) {
	ref := r.URL.Query().Get("ref")
	if ref == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "missing ref"})
		return
	}
	sse, ok := newSSE(w)
	if !ok {
		return
	}
	err := s.docker.PullImage(r.Context(), ref, func(msg map[string]any) {
		sse.send("progress", msg)
	})
	result := map[string]any{"ok": true}
	if err != nil && r.Context().Err() == nil {
		result["ok"] = false
		result["error"] = err.Error()
	}
	sse.send("done", result)
}
