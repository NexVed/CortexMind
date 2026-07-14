// Package web embeds the built SolidJS UI (ui/dist) so cortexd can serve the
// whole application from a single binary — no separate web server needed.
//
// The `dist` directory is populated by the build step (build-windows.ps1 /
// `npm run build` + copy). A committed .gitkeep keeps the package compilable
// before the UI has been built.
package web

import (
	"embed"
	"io/fs"
)

//go:embed all:dist
var distFS embed.FS

// Available reports whether a real UI build has been embedded (more than just
// the .gitkeep placeholder).
func Available() bool {
	entries, err := fs.ReadDir(distFS, "dist")
	if err != nil {
		return false
	}
	for _, e := range entries {
		if e.Name() == "index.html" {
			return true
		}
	}
	return false
}

// FS returns the embedded UI rooted at the dist directory, ready to hand to
// apis.Static.
func FS() fs.FS {
	sub, err := fs.Sub(distFS, "dist")
	if err != nil {
		// Should never happen: dist always exists (at least the placeholder).
		return distFS
	}
	return sub
}
