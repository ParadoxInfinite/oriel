package server

import (
	"encoding/json"
	"net/http"
	"os/exec"
	"runtime"

	"github.com/ParadoxInfinite/oriel/internal/discovery"
)

// handleGetDiscovery returns the persisted discovery config (normalized so the
// frontend never sees nulls).
func (s *Server) handleGetDiscovery(w http.ResponseWriter, r *http.Request) {
	c := loadSettings().Discovery
	if c.Filter.Mode == "" {
		c.Filter.Mode = "off"
	}
	if c.Roots == nil {
		c.Roots = []discovery.Root{}
	}
	if c.Filter.Patterns == nil {
		c.Filter.Patterns = []string{}
	}
	if c.Aliases == nil {
		c.Aliases = map[string]string{}
	}
	writeJSON(w, http.StatusOK, c)
}

// handlePutDiscovery saves the discovery config, preserving the rest of settings.
func (s *Server) handlePutDiscovery(w http.ResponseWriter, r *http.Request) {
	var c discovery.Config
	if err := json.NewDecoder(r.Body).Decode(&c); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid body"})
		return
	}
	cur := loadSettings()
	cur.Discovery = c
	if err := saveSettings(cur); err != nil {
		writeJSON(w, http.StatusInternalServerError, errorBody(err))
		return
	}
	writeJSON(w, http.StatusOK, c)
}

// handleScanDiscovery scans the configured roots and returns discovered stacks
// that are NOT already deployed (those show in /api/stacks) and pass the filter.
// Running stacks are never excluded by the filter.
func (s *Server) handleScanDiscovery(w http.ResponseWriter, r *http.Request) {
	cfg := loadSettings().Discovery
	res := discovery.Scan(cfg)

	deployed := map[string]bool{}
	if list, err := s.docker.ListStacks(r.Context()); err == nil {
		for _, st := range list {
			deployed[st.Name] = true
		}
	}

	stacks := []discovery.Discovered{}
	hidden := 0
	for _, d := range res.Stacks {
		if deployed[d.Name] {
			continue
		}
		if !cfg.Filter.Allows(d) {
			hidden++
			continue
		}
		stacks = append(stacks, d)
	}
	writeJSON(w, http.StatusOK, map[string]any{"stacks": stacks, "roots": res.Roots, "hidden": hidden})
}

// handleFsList powers the Settings path typeahead. Errors degrade to an empty
// list (with the message) rather than a non-200, so typing stays smooth.
func (s *Server) handleFsList(w http.ResponseWriter, r *http.Request) {
	dir, dirs, err := discovery.ListDirs(r.URL.Query().Get("path"))
	out := map[string]any{"dir": dir, "entries": []string{}}
	if err != nil {
		out["error"] = err.Error()
	} else {
		out["entries"] = dirs
	}
	writeJSON(w, http.StatusOK, out)
}

// handleFsOpen reveals a directory in the OS file manager (best-effort, local).
func (s *Server) handleFsOpen(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("path")
	if path == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "missing path"})
		return
	}
	opener := "xdg-open"
	switch runtime.GOOS {
	case "darwin":
		opener = "open"
	case "windows":
		opener = "explorer"
	}
	_ = exec.Command(opener, path).Start()
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

// handleStackUp deploys a discovered compose project by file path, streaming the
// compose output as SSE (same shape as handleStackAction).
func (s *Server) handleStackUp(w http.ResponseWriter, r *http.Request) {
	dir := r.URL.Query().Get("dir")
	file := r.URL.Query().Get("file")
	if dir == "" || file == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "missing dir or file"})
		return
	}
	name, ownName, _ := discovery.Resolve(file, dir)
	lines, errc, err := s.docker.ComposeUpFile(r.Context(), dir, file, name, ownName)
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
