package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/ParadoxInfinite/oriel/internal/docker"
)

// Background operations ("ops"): long-running, cancellable jobs whose progress
// can be streamed and re-attached to after a client refresh. See jobs.go.

// pruneItem is one thing to remove, with its size so the server can report how
// much was reclaimed (the client already knows sizes from the prune preview).
type pruneItem struct {
	ID   string `json:"id"`
	Size int64  `json:"size"`
}

// handleStartSystemPrune launches a system prune as a background job and returns
// its id. Each category is opted in via a query flag: containers, images, networks,
// cache, volumes (e.g. ?containers=true&cache=true&volumes=false).
func (s *Server) handleStartSystemPrune(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	opts := docker.PruneOptions{
		Containers:    q.Get("containers") == "true",
		Images:        q.Get("images") == "true",
		Networks:      q.Get("networks") == "true",
		BuildCache:    q.Get("cache") == "true",
		BuildCacheAll: q.Get("cacheall") == "true",
		Volumes:       q.Get("volumes") == "true",
	}
	job := s.jobs.start("system-prune", "Reclaiming disk space", func(ctx context.Context, emit func(string)) error {
		_, err := s.docker.SystemPrune(ctx, opts, emit)
		return err
	})
	writeJSON(w, http.StatusOK, map[string]string{"id": job.ID})
}

// handleStartImagePrune removes the given images (by id) in the background.
func (s *Server) handleStartImagePrune(w http.ResponseWriter, r *http.Request) {
	s.startItemPrune(w, r, "image-prune", "Pruning images", func(ctx context.Context, id string, force bool) error {
		return s.docker.RemoveImage(ctx, id, true)
	})
}

// handleStartVolumePrune removes the given volumes (by name) in the background.
func (s *Server) handleStartVolumePrune(w http.ResponseWriter, r *http.Request) {
	s.startItemPrune(w, r, "volume-prune", "Pruning volumes", func(ctx context.Context, name string, force bool) error {
		return s.docker.RemoveVolume(ctx, name, false)
	})
}

// startItemPrune decodes the chosen items and launches a job that removes them
// one by one, emitting progress and honouring cancellation between items.
func (s *Server) startItemPrune(w http.ResponseWriter, r *http.Request, kind, title string, remove func(ctx context.Context, id string, force bool) error) {
	var body struct {
		Items []pruneItem `json:"items"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, errorBody(err))
		return
	}
	if len(body.Items) == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "nothing to prune"})
		return
	}
	items := body.Items
	job := s.jobs.start(kind, title, func(ctx context.Context, emit func(string)) error {
		var removed int
		var reclaimed int64
		for i, it := range items {
			if ctx.Err() != nil {
				emit(fmt.Sprintf("Cancelled — %d of %d removed (%s)", removed, len(items), humanBytes(reclaimed)))
				return ctx.Err()
			}
			emit(fmt.Sprintf("Removing %d of %d…", i+1, len(items)))
			if err := remove(ctx, it.ID, true); err != nil {
				emit(fmt.Sprintf("  skipped %s: %s", shortID(it.ID), err.Error()))
				continue
			}
			removed++
			reclaimed += it.Size
		}
		emit(fmt.Sprintf("Done — removed %d, reclaimed %s", removed, humanBytes(reclaimed)))
		return nil
	})
	writeJSON(w, http.StatusOK, map[string]string{"id": job.ID})
}

// handleListOps returns the still-running jobs so a refreshed client can
// re-attach to whatever it left running.
func (s *Server) handleListOps(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.jobs.active())
}

// handleOpStream streams a job's progress as SSE: a one-shot "snapshot" event
// replays everything so far (so a late/reconnecting client catches up without
// duplicates), then live "line" events, then a final "done".
func (s *Server) handleOpStream(w http.ResponseWriter, r *http.Request) {
	job := s.jobs.get(r.PathValue("id"))
	if job == nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "no such operation"})
		return
	}
	sse, ok := newSSE(w)
	if !ok {
		return
	}
	snapshot, ch, done, jok, jerr, unsub := job.subscribe()
	defer unsub()

	sse.send("snapshot", map[string]any{"lines": snapshot})
	if done {
		sse.send("done", doneBody(jok, jerr))
		return
	}
	for {
		select {
		case line, more := <-ch:
			if !more {
				ok, errMsg := job.finalState()
				sse.send("done", doneBody(ok, errMsg))
				return
			}
			sse.send("line", map[string]string{"line": line})
		case <-r.Context().Done():
			return // client gone; job keeps running, unsub via defer
		}
	}
}

// handleCancelOp requests cancellation of a running job.
func (s *Server) handleCancelOp(w http.ResponseWriter, r *http.Request) {
	job := s.jobs.get(r.PathValue("id"))
	if job == nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "no such operation"})
		return
	}
	if job.cancel != nil {
		job.cancel()
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func doneBody(ok bool, errMsg string) map[string]any {
	b := map[string]any{"ok": ok}
	if errMsg != "" {
		b["error"] = errMsg
	}
	return b
}

func shortID(id string) string {
	id = strings.TrimPrefix(id, "sha256:")
	if len(id) > 12 {
		return id[:12]
	}
	return id
}

// humanBytes formats a byte count as a short human-readable string (e.g. "1.2 GiB").
func humanBytes(n int64) string {
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%d B", n)
	}
	div, exp := int64(unit), 0
	for x := n / unit; x >= unit; x /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(n)/float64(div), "KMGTPE"[exp])
}
