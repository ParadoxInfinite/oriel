package server

import (
	"encoding/json"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

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
	if err := updateSettings(func(st *settings) { st.Discovery = c }); err != nil {
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
	// Only reveal a real directory. Otherwise open/xdg-open would launch an
	// arbitrary file through its handler, and a leading '-' would inject a flag.
	if strings.HasPrefix(path, "-") {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid path"})
		return
	}
	if fi, err := os.Stat(path); err != nil || !fi.IsDir() {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "not a directory"})
		return
	}
	opener := "xdg-open"
	args := []string{path}
	switch runtime.GOOS {
	case "darwin":
		opener = "open"
		// On macOS a directory whose name ends in a launchable bundle suffix
		// (.app/.pkg/…) is executed by `open`, not merely revealed, even though it
		// passed the IsDir check. Reveal it in Finder (-R) instead of running it.
		switch strings.ToLower(filepath.Ext(path)) {
		case ".app", ".bundle", ".pkg", ".command", ".workflow", ".prefpane", ".osx":
			args = []string{"-R", path}
		}
	case "windows":
		opener = "explorer"
	}
	if err := exec.Command(opener, args...).Start(); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "could not open: " + err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

// discoveredFile reports whether (dir, file) is a compose project the current
// discovery config actually finds, the allowlist for what handleStackUp may run.
func discoveredFile(cfg discovery.Config, dir, file string) bool {
	for _, d := range discovery.Scan(cfg).Stacks {
		if d.Dir == dir && d.File == file {
			return true
		}
	}
	return false
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
	// Only deploy a compose file discovery actually found under the configured
	// roots, never an arbitrary path from the request. This blocks running any
	// YAML on disk and flag-injection via a leading '-'.
	if !discoveredFile(loadSettings().Discovery, dir, file) {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "not a discovered compose file"})
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
