// Package userdata resolves Oriel's per-user data directory. It's the single
// source of truth for where settings.json, the stats recording, and the
// destructive-grant state live, so the server, the `oriel mcp` process, and the
// CLI all agree on the same files.
package userdata

import (
	"os"
	"path/filepath"
)

// Path returns a stable per-user file under <config>/oriel, falling back through
// the cache dir then temp so a missing config dir never breaks anything.
func Path(name string) string {
	dir, err := os.UserConfigDir()
	if err != nil || dir == "" {
		if dir, err = os.UserCacheDir(); err != nil || dir == "" {
			dir = os.TempDir()
		}
	}
	return filepath.Join(dir, "oriel", name)
}
