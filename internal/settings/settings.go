// Package settings owns the user's persisted configuration (settings.json):
// base path, allowed hosts, auth token, masking policy, and compose-discovery
// config. It lives in its own package so both the server and the
// standalone `oriel mcp` process can read and atomically write it, the server
// is the usual writer, but MCP needs to set a stack alias.
package settings

import (
	"crypto/subtle"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/ParadoxInfinite/oriel/internal/discovery"
	"github.com/ParadoxInfinite/oriel/internal/userdata"
)

// Settings is the single source of truth for everything the user configures (as
// opposed to colima/docker state). Persisted as settings.json and edited via the
// UI, the CLI, MCP, or by hand.
type Settings struct {
	BasePath     string           `json:"basePath"` // reverse-proxy sub-path, e.g. /oriel ("" = root)
	Discovery    discovery.Config `json:"discovery"`
	AllowedHosts []string         `json:"allowedHosts"` // non-loopback Hosts allowed to reach /api
	MaskEnv      string           `json:"maskEnv"`      // inspect env masking: "all" (default) | "sensitive" | "off"
	MaskLogs     string           `json:"maskLogs"`     // UI log masking: "sensitive" (default, redact secrets) | "off". The MCP/agent path is always at least "sensitive".
	EnvReveal    string           `json:"envReveal"`    // where "reveal values" works: "local" (default) | "remote" | "off"
	AuthToken    string           `json:"authToken"`    // opt-in bearer token required for non-loopback /api ("" = off)
}

var mu sync.Mutex

// Path is the settings.json location.
func Path() string { return userdata.Path("settings.json") }

// Load reads settings.json (zero value if missing or unreadable).
func Load() Settings {
	mu.Lock()
	defer mu.Unlock()
	return loadLocked()
}

// Save writes settings.json atomically.
func Save(c Settings) error {
	mu.Lock()
	defer mu.Unlock()
	return saveLocked(c)
}

// Update performs a read-modify-write under a single hold of the lock, so
// concurrent in-process updates to different fields can't clobber one another.
// Across processes (server vs. `oriel mcp`) the atomic temp+rename prevents torn
// files; the last writer wins, acceptable for these low-stakes fields.
func Update(mutate func(*Settings)) error {
	mu.Lock()
	defer mu.Unlock()
	c := loadLocked()
	mutate(&c)
	return saveLocked(c)
}

// Bearer extracts the token from an "Authorization: Bearer <token>" header value.
// The scheme is case-insensitive; the token is trimmed. Returns "" if absent.
func Bearer(header string) string {
	const p = "Bearer "
	if len(header) > len(p) && strings.EqualFold(header[:len(p)], p) {
		return strings.TrimSpace(header[len(p):])
	}
	return ""
}

// TokenOK reports whether the provided bearer token matches the configured one,
// in constant time so a wrong token can't be guessed byte-by-byte via timing. An
// empty configured token means auth is off (always OK). The single source of the
// security-critical compare, shared by the GUI gate and the MCP-over-HTTP gate.
func TokenOK(provided, configured string) bool {
	if configured == "" {
		return true
	}
	return provided != "" && subtle.ConstantTimeCompare([]byte(provided), []byte(configured)) == 1
}

// SetAlias sets (or, with an empty alias, clears) the Oriel display alias for a
// compose project. Display only, the real project name is unchanged.
func SetAlias(name, alias string) error {
	return Update(func(c *Settings) {
		if c.Discovery.Aliases == nil {
			c.Discovery.Aliases = map[string]string{}
		}
		if strings.TrimSpace(alias) == "" {
			delete(c.Discovery.Aliases, name)
		} else {
			c.Discovery.Aliases[name] = strings.TrimSpace(alias)
		}
	})
}

func loadLocked() Settings {
	var c Settings
	b, err := os.ReadFile(Path())
	if err != nil {
		return c
	}
	_ = json.Unmarshal(b, &c)
	return c
}

func saveLocked(c Settings) error {
	path := Path()
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	// Unique temp + rename so a crash or a concurrent writer never sees (or
	// collides on) a torn file. os.CreateTemp creates it 0600.
	f, err := os.CreateTemp(dir, "settings-*.json.tmp")
	if err != nil {
		return err
	}
	tmp := f.Name()
	if _, err := f.Write(b); err != nil {
		f.Close()
		os.Remove(tmp)
		return err
	}
	if err := f.Close(); err != nil {
		os.Remove(tmp)
		return err
	}
	return os.Rename(tmp, path)
}
