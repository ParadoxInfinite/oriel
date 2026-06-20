package main

import (
	"embed"
	"io/fs"
)

// webDist holds the built Svelte frontend. The `all:` prefix ensures files
// beginning with `_` or `.` (common in Vite output) are included.
//
//go:embed all:web/dist
var webDist embed.FS

// webFS returns the embedded frontend rooted at web/dist, so paths are served
// as "/index.html" rather than "/web/dist/index.html".
func webFS() fs.FS {
	sub, err := fs.Sub(webDist, "web/dist")
	if err != nil {
		// Unreachable: the path is a compile-time constant embedded above.
		panic(err)
	}
	return sub
}
