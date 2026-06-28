package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"sync/atomic"
	"testing"
)

// pointI18nAtTestServer redirects the CDN base at a test server and an isolated
// cache dir, so the fetch/cache logic can be exercised offline.
func pointI18nAtTestServer(t *testing.T, h http.Handler) {
	t.Helper()
	ts := httptest.NewServer(h)
	t.Cleanup(ts.Close)
	prev := i18nCDNBase
	i18nCDNBase = ts.URL
	t.Cleanup(func() { i18nCDNBase = prev })

	tmp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmp)
	t.Setenv("HOME", tmp) // darwin's UserConfigDir keys off HOME
}

func TestI18nFetchCachesAndSurvivesOutage(t *testing.T) {
	var hits int32
	pointI18nAtTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&hits, 1)
		if filepath.Base(r.URL.Path) != "fr.json" {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write([]byte(`{"action.remove":"Supprimer"}`))
	}))

	b, err := i18nFetch(context.Background(), "fr.json")
	if err != nil {
		t.Fatalf("first fetch: %v", err)
	}
	if string(b) != `{"action.remove":"Supprimer"}` {
		t.Fatalf("unexpected body: %s", b)
	}
	// Second fetch is served from the fresh cache, not the network.
	if _, err := i18nFetch(context.Background(), "fr.json"); err != nil {
		t.Fatalf("second fetch: %v", err)
	}
	if got := atomic.LoadInt32(&hits); got != 1 {
		t.Fatalf("expected 1 network hit (rest cached), got %d", got)
	}
}

func TestI18nDownloadRejectsNonJSON(t *testing.T) {
	pointI18nAtTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("<html>not json</html>"))
	}))
	if _, err := i18nDownload(context.Background(), "xx.json"); err == nil {
		t.Fatal("expected an error for a non-JSON catalog, got nil")
	}
}

func TestI18nCatalogHandlerValidatesTag(t *testing.T) {
	s := &Server{}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/i18n/../secrets", nil)
	req.SetPathValue("locale", "../secrets")
	s.handleI18nCatalog(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for a malformed tag, got %d", rec.Code)
	}
}
