package server

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/ParadoxInfinite/oriel/internal/service"
)

// updateRepo is the GitHub repo whose releases Oriel checks against.
const updateRepo = "ParadoxInfinite/oriel"

// updateInfo is the result of an update check.
type updateInfo struct {
	Current         string `json:"current"`
	Latest          string `json:"latest"`
	UpdateAvailable bool   `json:"updateAvailable"`
	Managed         bool   `json:"managed"` // service-managed install → self-update is offered
	URL             string `json:"url"`
	PublishedAt     string `json:"publishedAt,omitempty"`
	Error           string `json:"error,omitempty"`
}

// ghRelease is the slice of the GitHub release payload we use.
type ghRelease struct {
	TagName     string `json:"tag_name"`
	HTMLURL     string `json:"html_url"`
	PublishedAt string `json:"published_at"`
	Assets      []struct {
		Name string `json:"name"`
		URL  string `json:"browser_download_url"`
	} `json:"assets"`
}

func (r *ghRelease) assetURL(name string) string {
	for _, a := range r.Assets {
		if a.Name == name {
			return a.URL
		}
	}
	return ""
}

// The check hits the GitHub API, so cache it: unauthenticated callers are rate
// limited (~60/hr/IP) and the answer rarely changes.
var (
	updateMu      sync.Mutex
	updateCache   *updateInfo
	updateFetched time.Time // when we last actually reached GitHub
)

const (
	passiveTTL = 24 * time.Hour  // background re-reads reuse the cache for a day
	manualTTL  = time.Hour       // a forced "check now" can refresh at most hourly
	errorTTL   = 5 * time.Minute // retry sooner after a failed check
)

// updateClient handles all self-update network I/O. It refuses to follow a
// redirect to anything but HTTPS, so a downgrade can't strip transport security
// out from under the checksum verification.
var updateClient = &http.Client{
	CheckRedirect: func(req *http.Request, via []*http.Request) error {
		if req.URL.Scheme != "https" {
			return fmt.Errorf("refusing non-HTTPS redirect")
		}
		if len(via) >= 10 {
			return fmt.Errorf("too many redirects")
		}
		return nil
	},
}

// handleUpdateCheck reports whether a newer release exists. Lazy and cached, so
// it only reaches GitHub when the cache is stale. `?force=1` (the manual "check
// for updates" button) lowers the staleness floor to an hour, but never below it.
func (s *Server) handleUpdateCheck(w http.ResponseWriter, r *http.Request) {
	force := r.URL.Query().Has("force")
	updateMu.Lock()
	defer updateMu.Unlock()

	if updateCache != nil {
		ttl := passiveTTL
		if force {
			ttl = manualTTL
		}
		if updateCache.Error != "" && errorTTL < ttl {
			ttl = errorTTL
		}
		if time.Since(updateFetched) < ttl {
			writeJSON(w, http.StatusOK, updateCache)
			return
		}
	}
	info := s.fetchLatestRelease(r.Context())
	updateCache = &info
	updateFetched = time.Now()
	writeJSON(w, http.StatusOK, info)
}

func (s *Server) fetchLatestRelease(ctx context.Context) updateInfo {
	info := updateInfo{Current: s.version, Managed: service.IsManaged()}
	rel, err := githubLatestRelease(ctx)
	if err != nil {
		info.Error = err.Error()
		return info
	}
	info.Latest = strings.TrimPrefix(rel.TagName, "v")
	info.URL = rel.HTMLURL
	info.PublishedAt = rel.PublishedAt
	info.UpdateAvailable = isNewer(info.Current, info.Latest)
	return info
}

// githubLatestRelease fetches the latest release (tag + assets) from GitHub.
func githubLatestRelease(ctx context.Context) (*ghRelease, error) {
	reqCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet,
		"https://api.github.com/repos/"+updateRepo+"/releases/latest", nil)
	if err != nil {
		return nil, fmt.Errorf("could not build request")
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "oriel-update-check")

	resp, err := updateClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("could not reach GitHub")
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub returned %d", resp.StatusCode)
	}
	var rel ghRelease
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		return nil, fmt.Errorf("unexpected response from GitHub")
	}
	return &rel, nil
}

// isNewer reports whether latest is a strictly higher version than current. A
// non-release current ("dev" or empty) never reports an update — local builds
// shouldn't nag.
func isNewer(current, latest string) bool {
	if latest == "" || current == "" || current == "dev" {
		return false
	}
	return compareSemver(latest, current) > 0
}

func compareSemver(a, b string) int {
	pa, pb := parseVer(a), parseVer(b)
	for i := 0; i < 3; i++ {
		switch {
		case pa[i] > pb[i]:
			return 1
		case pa[i] < pb[i]:
			return -1
		}
	}
	return 0
}

// parseVer extracts the major/minor/patch numbers from a version string,
// ignoring any leading "v" and any pre-release/build suffix.
func parseVer(v string) [3]int {
	var out [3]int
	v = strings.TrimPrefix(strings.TrimSpace(v), "v")
	if i := strings.IndexAny(v, "-+"); i >= 0 {
		v = v[:i]
	}
	for i, part := range strings.SplitN(v, ".", 3) {
		n, _ := strconv.Atoi(part)
		out[i] = n
	}
	return out
}

// ---- self-update (service-managed installs only) ----------------------------

func httpError(w http.ResponseWriter, code int, msg string) {
	writeJSON(w, code, map[string]string{"error": msg})
}

// handleUpdateApply downloads the latest release binary for this platform,
// verifies it against the release's SHA256SUMS, and atomically replaces the
// running executable. It refuses unless this is a service-managed install, so a
// restart cleanly brings the new binary up. It does NOT restart — the client
// calls /api/update/restart when the user is ready.
func (s *Server) handleUpdateApply(w http.ResponseWriter, r *http.Request) {
	if !service.IsManaged() {
		httpError(w, http.StatusBadRequest, "self-update is only available for service-managed installs — run: oriel service install")
		return
	}
	exe, err := os.Executable()
	if err != nil {
		httpError(w, http.StatusInternalServerError, "cannot locate the running binary")
		return
	}
	exe, _ = filepath.EvalSymlinks(exe)

	rel, err := githubLatestRelease(r.Context())
	if err != nil {
		httpError(w, http.StatusBadGateway, err.Error())
		return
	}
	latest := strings.TrimPrefix(rel.TagName, "v")
	if !isNewer(s.version, latest) {
		writeJSON(w, http.StatusOK, map[string]any{"updated": false, "current": s.version, "latest": latest, "message": "already up to date"})
		return
	}

	asset := fmt.Sprintf("oriel-%s-%s", runtime.GOOS, runtime.GOARCH)
	binURL, sumURL := rel.assetURL(asset), rel.assetURL("SHA256SUMS.txt")
	if binURL == "" || sumURL == "" {
		httpError(w, http.StatusBadGateway, "release has no downloadable asset for "+asset)
		return
	}

	want, err := fetchChecksum(r.Context(), sumURL, asset)
	if err != nil {
		httpError(w, http.StatusBadGateway, "could not read checksums: "+err.Error())
		return
	}

	// Download alongside the current binary so the final rename is atomic (same
	// filesystem). The temp file is cleaned up unless it's renamed into place.
	tmp, got, err := downloadFile(r.Context(), binURL, filepath.Dir(exe))
	if tmp != "" {
		defer os.Remove(tmp)
	}
	if err != nil {
		httpError(w, http.StatusBadGateway, "download failed: "+err.Error())
		return
	}
	if !strings.EqualFold(got, want) {
		httpError(w, http.StatusBadGateway, "checksum mismatch — refusing to install")
		return
	}
	if err := os.Chmod(tmp, 0o755); err != nil {
		httpError(w, http.StatusInternalServerError, "could not set permissions: "+err.Error())
		return
	}
	// Keep the old binary as <exe>.bak so a bad update can be rolled back, and so
	// the swap stays recoverable if the second rename fails.
	bak := exe + ".bak"
	_ = os.Remove(bak)
	if err := os.Rename(exe, bak); err != nil {
		httpError(w, http.StatusInternalServerError, "could not back up current binary: "+err.Error())
		return
	}
	if err := os.Rename(tmp, exe); err != nil {
		_ = os.Rename(bak, exe) // roll back to the previous binary
		httpError(w, http.StatusInternalServerError, "could not install update: "+err.Error())
		return
	}
	// Refresh the cached check so the UI reflects the new state.
	updateMu.Lock()
	updateCache = nil
	updateMu.Unlock()
	writeJSON(w, http.StatusOK, map[string]any{"updated": true, "current": s.version, "latest": latest, "restartRequired": true})
}

// handleUpdateRestart restarts the managed service so a freshly-installed binary
// takes effect. It replies first, then restarts after the response flushes.
func (s *Server) handleUpdateRestart(w http.ResponseWriter, r *http.Request) {
	if !service.IsManaged() {
		httpError(w, http.StatusBadRequest, "restart is only available for service-managed installs")
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
	go func() {
		time.Sleep(500 * time.Millisecond)
		_ = service.Restart()
	}()
}

// fetchChecksum downloads a SHA256SUMS file and returns the hex digest for asset.
func fetchChecksum(ctx context.Context, url, asset string) (string, error) {
	reqCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	req, _ := http.NewRequestWithContext(reqCtx, http.MethodGet, url, nil)
	req.Header.Set("User-Agent", "oriel-update")
	resp, err := updateClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	sc := bufio.NewScanner(resp.Body)
	for sc.Scan() {
		// lines: "<hex>  <filename>"
		f := strings.Fields(sc.Text())
		if len(f) == 2 && f[1] == asset {
			return f[0], nil
		}
	}
	return "", fmt.Errorf("no checksum for %s", asset)
}

// downloadFile streams url into a temp file under dir, returning its path and
// hex SHA-256. The caller removes the temp file (or renames it into place).
func downloadFile(ctx context.Context, url, dir string) (string, string, error) {
	reqCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()
	req, _ := http.NewRequestWithContext(reqCtx, http.MethodGet, url, nil)
	req.Header.Set("User-Agent", "oriel-update")
	resp, err := updateClient.Do(req)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	f, err := os.CreateTemp(dir, ".oriel-update-*")
	if err != nil {
		return "", "", err
	}
	h := sha256.New()
	if _, err := io.Copy(io.MultiWriter(f, h), resp.Body); err != nil {
		f.Close()
		return f.Name(), "", err
	}
	if err := f.Close(); err != nil {
		return f.Name(), "", err
	}
	return f.Name(), hex.EncodeToString(h.Sum(nil)), nil
}
