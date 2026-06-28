package server

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/ParadoxInfinite/oriel/internal/userdata"
)

// handleContainerShell upgrades to a WebSocket and runs an interactive shell in
// the container. Output (the PTY stream) flows container→browser as binary
// frames; browser→container, a binary frame is stdin and a text frame is a
// control message ({"resize":{"cols":N,"rows":M}}). It's a UI-only feature — gated
// by the same /api auth as everything else, never exposed over MCP — and can be
// turned off entirely in Settings.
func (s *Server) handleContainerShell(w http.ResponseWriter, r *http.Request) {
	if loadSettings().ShellDisabled {
		http.Error(w, "container shell is disabled", http.StatusForbidden)
		return
	}
	id := r.PathValue("id")
	ws, err := wsUpgrade(w, r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	defer ws.Close()

	// Not r.Context(): the server cancels the request context once the connection
	// is hijacked, which would kill the exec attach immediately. The shell's
	// lifetime is the WebSocket's, torn down via the read loops below and cancel().
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Prefer bash, fall back to sh — covers the vast majority of images.
	// A failed `exec` is fatal in POSIX sh, so probe for bash before exec'ing it;
	// otherwise fall back to sh (which every image with /bin/sh has).
	sess, err := s.docker.Exec(ctx, id, []string{"/bin/sh", "-c", "if command -v bash >/dev/null 2>&1; then exec bash; else exec sh; fi"}, 24, 80)
	if err != nil {
		_ = ws.WriteBinary([]byte("oriel: cannot start shell: " + err.Error() + "\r\n"))
		return
	}
	defer sess.Close()

	// Container → browser. When the shell exits, closing the socket unblocks the
	// read loop below so the handler returns and the defers fire.
	go func() {
		buf := make([]byte, 32*1024)
		for {
			n, rerr := sess.Conn.Reader.Read(buf)
			if n > 0 {
				if werr := ws.WriteBinary(buf[:n]); werr != nil {
					println("SHELL write err:", werr.Error())
					break
				}
			}
			if rerr != nil {
				break
			}
		}
		ws.Close()
	}()

	// Browser → container: stdin (binary frames) and resize (text control frames).
	for {
		op, data, rerr := ws.ReadMessage()
		if rerr != nil {
			return
		}
		if op == wsText {
			var ctl struct {
				Resize *struct {
					Cols uint `json:"cols"`
					Rows uint `json:"rows"`
				} `json:"resize"`
			}
			if json.Unmarshal(data, &ctl) == nil && ctl.Resize != nil {
				_ = sess.Resize(ctx, ctl.Resize.Rows, ctl.Resize.Cols)
			}
			continue
		}
		if _, werr := sess.Conn.Conn.Write(data); werr != nil {
			return
		}
	}
}

// The terminal emulator (xterm.js) is fetched on demand rather than bundled, so
// the binary and the main UI bundle carry no terminal dependency. The backend
// proxies it from the CDN once, caches it on disk, and serves it same-origin so
// it loads under the app's CSP without a third-party script source.
const xtermVersion = "5.5.0"

var termAssets = map[string]struct{ url, mime string }{
	"xterm.js":  {"https://cdn.jsdelivr.net/npm/@xterm/xterm@" + xtermVersion + "/+esm", "application/javascript; charset=utf-8"},
	"xterm.css": {"https://cdn.jsdelivr.net/npm/@xterm/xterm@" + xtermVersion + "/css/xterm.css", "text/css; charset=utf-8"},
}

const maxTermAssetBytes = 4 << 20 // 4 MiB; xterm's JS+CSS are well under this

// handleTermAsset serves a cached xterm asset, downloading it on first use.
func (s *Server) handleTermAsset(w http.ResponseWriter, r *http.Request) {
	a, ok := termAssets[r.PathValue("file")]
	if !ok {
		http.NotFound(w, r)
		return
	}
	b, err := termFetch(r.Context(), r.PathValue("file"), a.url)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	w.Header().Set("Content-Type", a.mime)
	w.Header().Set("Cache-Control", "public, max-age=86400")
	_, _ = w.Write(b)
}

// termFetch returns a terminal asset from disk cache, downloading it once. The
// version is pinned, so the cache is effectively permanent; bumping the version
// downloads fresh. Uses the size-capped HTTP client shared with the i18n proxy.
func termFetch(ctx context.Context, name, url string) ([]byte, error) {
	cache := userdata.Path(filepath.Join("assets", "xterm-"+xtermVersion, name))
	if b, err := os.ReadFile(cache); err == nil && len(b) > 0 {
		return b, nil
	}
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
		return nil, fmt.Errorf("asset fetch: HTTP %d", resp.StatusCode)
	}
	b, err := io.ReadAll(io.LimitReader(resp.Body, maxTermAssetBytes))
	if err != nil {
		return nil, err
	}
	writeCache(cache, b)
	return b, nil
}
