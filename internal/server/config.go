package server

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"

	"github.com/ParadoxInfinite/oriel/internal/discovery"
)

// settings is the slice of user config the GUI persists itself (as opposed to
// colima/docker state): the runtime provider URL and compose-discovery config.
type settings struct {
	ProviderURL string           `json:"providerUrl"`
	Discovery   discovery.Config `json:"discovery"`
}

var settingsMu sync.Mutex

func settingsPath() string { return dataPath("settings.json") }

func loadSettings() settings {
	settingsMu.Lock()
	defer settingsMu.Unlock()
	var c settings
	b, err := os.ReadFile(settingsPath())
	if err != nil {
		return c
	}
	_ = json.Unmarshal(b, &c)
	return c
}

func saveSettings(c settings) error {
	settingsMu.Lock()
	defer settingsMu.Unlock()
	path := settingsPath()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	// Write+rename so a crash or concurrent read never sees a torn file.
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}
