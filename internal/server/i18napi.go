package server

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sync"
	"time"

	"github.com/ParadoxInfinite/oriel/internal/userdata"
)

// Translation catalogs are served from the repo over a CDN, so a translation fix
// reaches users without a new binary. English ships inside the binary and is the
// fallback; the browser fetches any other locale through these endpoints (which
// avoids CORS and works when only the backend has outbound network).
var (
	i18nClient  = &http.Client{Timeout: 10 * time.Second}
	i18nCDNBase = "https://cdn.jsdelivr.net/gh/ParadoxInfinite/oriel@main/web/src/i18n"
)

const (
	maxCatalogBytes = 1 << 20 // 1 MiB: a catalog is small text
	i18nCacheTTL    = 6 * time.Hour
)

// A locale tag is BCP-47-ish; the bound also stops the value from reaching for a
// different path or host on the CDN.
var localeTagRe = regexp.MustCompile(`^[a-z]{2,3}(-[A-Za-z0-9]{2,8})*$`)

var i18nMu sync.Mutex

// handleI18nManifest returns the list of locales available on the CDN, so the UI
// can offer languages that were published after this binary was built.
func (s *Server) handleI18nManifest(w http.ResponseWriter, r *http.Request) {
	b, err := i18nFetch(r.Context(), "manifest.json")
	if err != nil {
		writeJSON(w, http.StatusBadGateway, errorBody(err))
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_, _ = w.Write(b)
}

// handleI18nCatalog returns one locale's catalog.
func (s *Server) handleI18nCatalog(w http.ResponseWriter, r *http.Request) {
	tag := r.PathValue("locale")
	if !localeTagRe.MatchString(tag) {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid locale"})
		return
	}
	b, err := i18nFetch(r.Context(), tag+".json")
	if err != nil {
		writeJSON(w, http.StatusBadGateway, errorBody(err))
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_, _ = w.Write(b)
}

// i18nFetch returns a file from the CDN, caching it on disk for i18nCacheTTL. A
// failed refresh falls back to a stale cached copy if one exists, so a brief CDN
// hiccup doesn't drop a working translation.
func i18nFetch(ctx context.Context, name string) ([]byte, error) {
	cache := userdata.Path(filepath.Join("i18n", name))
	if b, ok := freshCache(cache); ok {
		return b, nil
	}
	i18nMu.Lock()
	defer i18nMu.Unlock()
	if b, ok := freshCache(cache); ok { // another request may have refreshed it
		return b, nil
	}
	b, err := i18nDownload(ctx, name)
	if err != nil {
		if stale, e := os.ReadFile(cache); e == nil {
			return stale, nil
		}
		return nil, err
	}
	writeCache(cache, b)
	return b, nil
}

func i18nDownload(ctx context.Context, name string) ([]byte, error) {
	b, err := downloadCapped(ctx, i18nCDNBase+"/"+name, maxCatalogBytes)
	if err != nil {
		return nil, err
	}
	if !json.Valid(b) {
		return nil, fmt.Errorf("catalog fetch: response was not JSON")
	}
	return b, nil
}

// downloadCapped GETs url with the shared client, reading at most max bytes.
// Callers layer their own caching/validation on top.
func downloadCapped(ctx context.Context, url string, max int64) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := i18nClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("download: HTTP %d", resp.StatusCode)
	}
	return io.ReadAll(io.LimitReader(resp.Body, max))
}

func freshCache(path string) ([]byte, bool) {
	fi, err := os.Stat(path)
	if err != nil || time.Since(fi.ModTime()) > i18nCacheTTL {
		return nil, false
	}
	b, err := os.ReadFile(path)
	return b, err == nil
}

func writeCache(path string, b []byte) {
	dir := filepath.Dir(path)
	if os.MkdirAll(dir, 0o755) != nil {
		return
	}
	f, err := os.CreateTemp(dir, "cat-*.json.tmp")
	if err != nil {
		return
	}
	tmp := f.Name()
	if _, err := f.Write(b); err != nil {
		f.Close()
		os.Remove(tmp)
		return
	}
	if f.Close() != nil {
		os.Remove(tmp)
		return
	}
	_ = os.Rename(tmp, path)
}
