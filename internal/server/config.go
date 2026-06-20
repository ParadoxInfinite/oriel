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
	return os.WriteFile(path, b, 0o600)
}
