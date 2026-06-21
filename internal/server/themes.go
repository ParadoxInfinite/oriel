package server

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// Themes are loaded from disk, not from arbitrary URLs: drop a built theme
// bundle (an ES module, *.js) into the themes directory and Oriel serves it
// same-origin. Explicit, offline, and no paste-a-malicious-link vector.
func themesDir() string { return dataPath("themes") }

// handleListThemes lists installed theme bundles and the directory to drop them in.
func (s *Server) handleListThemes(w http.ResponseWriter, r *http.Request) {
	dir := themesDir()
	_ = os.MkdirAll(dir, 0o755) // so the dir exists for the user to find

	type theme struct {
		File string `json:"file"`
		Name string `json:"name"`
	}
	themes := []theme{}
	entries, _ := os.ReadDir(dir)
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".js") {
			continue
		}
		themes = append(themes, theme{File: e.Name(), Name: strings.TrimSuffix(e.Name(), ".js")})
	}
	writeJSON(w, http.StatusOK, map[string]any{"dir": dir, "themes": themes})
}

// handleServeTheme serves one theme bundle from disk as a JS module. The name
// must be a bare .js filename in the themes dir — no path traversal.
func (s *Server) handleServeTheme(w http.ResponseWriter, r *http.Request) {
	file := r.PathValue("file")
	if file == "" || file != filepath.Base(file) || !strings.HasSuffix(file, ".js") {
		http.Error(w, "bad theme name", http.StatusBadRequest)
		return
	}
	b, err := os.ReadFile(filepath.Join(themesDir(), file))
	if err != nil {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/javascript; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	_, _ = w.Write(b)
}
