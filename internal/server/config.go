package server

import (
	"log"
	"os"
	"strings"

	settingspkg "github.com/ParadoxInfinite/oriel/internal/settings"
)

// Settings persistence lives in internal/settings so the standalone `oriel mcp`
// process can read/write it too. These aliases keep the server's existing call
// sites (settings, updateSettings, loadSettings, …) unchanged.
type settings = settingspkg.Settings

var (
	loadSettings   = settingspkg.Load
	saveSettings   = settingspkg.Save
	updateSettings = settingspkg.Update
	settingsPath   = settingspkg.Path
)

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
