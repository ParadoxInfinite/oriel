package server

import (
	"bytes"
	"io/fs"
	"net/http"
	"path"
	"strings"
	"time"
)

// basePlaceholder is the base path Vite bakes into every asset URL (and into
// import.meta.env.BASE_URL) at build time. The server swaps it for the real,
// runtime-configured base so a single build can be served at the host root or
// behind a reverse-proxy subpath (e.g. https://host/oriel/) without a rebuild.
const basePlaceholder = "/__ORIEL_BASE__/"

// normalizeBase canonicalizes ORIEL_BASE_PATH input ("oriel", "/oriel",
// "/oriel/", "") into a value that always starts and ends with "/". "/" means
// the SPA is served at the host root.
func normalizeBase(raw string) string {
	raw = strings.Trim(strings.TrimSpace(raw), "/")
	if raw == "" {
		return "/"
	}
	return "/" + raw + "/"
}

// rewriteAssets loads the embedded frontend into memory, replacing the build
// placeholder with base in text assets so every asset URL — and the bundle's
// import.meta.env.BASE_URL, which the API client prepends to requests — resolves
// under the configured base. Binary files (fonts, images) pass through unchanged.
func rewriteAssets(web fs.FS, base string) map[string][]byte {
	out := map[string][]byte{}
	_ = fs.WalkDir(web, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		b, err := fs.ReadFile(web, p)
		if err != nil {
			return err
		}
		switch path.Ext(p) {
		case ".html", ".js", ".css":
			b = bytes.ReplaceAll(b, []byte(basePlaceholder), []byte(base))
		}
		out[p] = b
		return nil
	})
	return out
}

// staticHandler serves the embedded SPA. Real asset requests are served
// directly; anything else falls back to index.html so client-side routing works
// on deep links and reloads. Assets are rewritten once for the configured base.
func (s *Server) staticHandler() http.Handler {
	assets := rewriteAssets(s.web, s.base)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		name := strings.TrimPrefix(path.Clean(r.URL.Path), "/")
		if name == "" {
			name = "index.html"
		}
		if _, ok := assets[name]; !ok {
			// Not a real file → SPA fallback. Hashed assets that genuinely
			// 404 still resolve to index.html, the standard single-page-app
			// behavior.
			name = "index.html"
		}

		cacheControl(w, name)
		// Zero modtime so ServeContent skips Last-Modified/If-Modified-Since; we
		// drive freshness with the Cache-Control header set above. It still sets
		// the correct Content-Type by extension and honors Range requests.
		http.ServeContent(w, r, name, time.Time{}, bytes.NewReader(assets[name]))
	})
}

// cacheControl applies long-lived caching to fingerprinted assets and no-cache
// to the HTML entrypoint so deploys are picked up immediately.
func cacheControl(w http.ResponseWriter, name string) {
	if strings.HasPrefix(name, "assets/") {
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		return
	}
	w.Header().Set("Cache-Control", "no-cache")
}
