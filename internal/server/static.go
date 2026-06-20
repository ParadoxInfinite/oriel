package server

import (
	"io/fs"
	"net/http"
	"path"
	"strings"
)

// staticHandler serves the embedded SPA. Real asset requests are served
// directly; anything else falls back to index.html so client-side routing works
// on deep links and reloads.
func (s *Server) staticHandler() http.Handler {
	fileServer := http.FileServer(http.FS(s.web))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upath := strings.TrimPrefix(path.Clean(r.URL.Path), "/")
		if upath == "" {
			upath = "index.html"
		}

		if _, err := fs.Stat(s.web, upath); err != nil {
			// Not a real file → SPA fallback. Hashed assets that genuinely
			// 404 will still resolve to index.html, which is the standard
			// single-page-app behavior.
			r2 := new(http.Request)
			*r2 = *r
			r2.URL.Path = "/"
			cacheControl(w, "index.html")
			fileServer.ServeHTTP(w, r2)
			return
		}

		cacheControl(w, upath)
		fileServer.ServeHTTP(w, r)
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
