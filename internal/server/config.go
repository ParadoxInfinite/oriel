package server

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/ParadoxInfinite/oriel/internal/discovery"
	"github.com/ParadoxInfinite/oriel/internal/provider"
)

// settings is the single source of truth for everything the user configures (as
// opposed to colima/docker state): reverse-proxy base path, the allowed-host
// list, the AI provider URL, and compose-discovery config. It is persisted as
// settings.json and edited via the UI, the CLI, or by hand.
type settings struct {
	BasePath     string           `json:"basePath"`    // reverse-proxy sub-path, e.g. /oriel ("" = root)
	ProviderURL  string           `json:"providerUrl"` // AI resolver endpoint ("" = dormant)
	Discovery    discovery.Config `json:"discovery"`
	AllowedHosts []string         `json:"allowedHosts"` // non-loopback Hosts allowed to reach /api
	MaskEnv      string           `json:"maskEnv"`      // inspect env masking: "all" (default) | "sensitive" | "off"
	EnvReveal    string           `json:"envReveal"`    // where "reveal values" works: "local" (default) | "remote" | "off"
}

// mergeEnvConfig folds pre-0.2 environment config into c without overwriting
// values already present (settings.json always wins). Returns the merged config
// and the names of the env vars adopted. Pure, for testability.
func mergeEnvConfig(c settings, getenv func(string) string) (settings, []string) {
	var migrated []string
	if c.BasePath == "" {
		if v := strings.TrimSpace(getenv("ORIEL_BASE_PATH")); v != "" {
			c.BasePath = v
			migrated = append(migrated, "ORIEL_BASE_PATH")
		}
	}
	if len(c.AllowedHosts) == 0 {
		if v := strings.TrimSpace(getenv("ORIEL_ALLOWED_HOSTS")); v != "" {
			c.AllowedHosts = normalizeHosts(strings.Split(v, ","))
			migrated = append(migrated, "ORIEL_ALLOWED_HOSTS")
		}
	}
	if c.ProviderURL == "" {
		if v := strings.TrimSpace(getenv(provider.EnvURL)); v != "" {
			c.ProviderURL = v
			migrated = append(migrated, provider.EnvURL)
		}
	}
	return c, migrated
}

// migrateLegacyEnvConfig is a one-time bridge: 0.2 makes settings.json the only
// config source, but older deployments set config via env vars (in the service
// unit, a shell, etc.). On startup we adopt any set env vars into settings.json
// and log that they're deprecated, so upgrades keep working without breakage.
func migrateLegacyEnvConfig() {
	merged, migrated := mergeEnvConfig(loadSettings(), os.Getenv)
	if len(migrated) == 0 {
		return
	}
	if err := saveSettings(merged); err != nil {
		log.Printf("oriel: could not migrate legacy env config: %v", err)
		return
	}
	for _, k := range migrated {
		log.Printf("oriel: %s is deprecated — migrated into %s; remove it from your service unit / environment", k, settingsPath())
	}
}

var settingsMu sync.Mutex

func settingsPath() string { return dataPath("settings.json") }

func loadSettings() settings {
	settingsMu.Lock()
	defer settingsMu.Unlock()
	return loadLocked()
}

func saveSettings(c settings) error {
	settingsMu.Lock()
	defer settingsMu.Unlock()
	return saveLocked(c)
}

// updateSettings performs a read-modify-write under a single hold of the lock,
// so concurrent updates to different fields can't clobber one another (each
// caller used to load → mutate one field → save, racing the others).
func updateSettings(mutate func(*settings)) error {
	settingsMu.Lock()
	defer settingsMu.Unlock()
	c := loadLocked()
	mutate(&c)
	return saveLocked(c)
}

// loadLocked / saveLocked assume settingsMu is already held.
func loadLocked() settings {
	var c settings
	b, err := os.ReadFile(settingsPath())
	if err != nil {
		return c
	}
	_ = json.Unmarshal(b, &c)
	return c
}

func saveLocked(c settings) error {
	path := settingsPath()
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
