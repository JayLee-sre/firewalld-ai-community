package dashboard

import (
	"embed"
	"io/fs"
)

// Embedded frontend files. During build, copy web/dist → internal/dashboard/dist
// before running `go build`. If dist/ is absent or only has a placeholder,
// the server falls back to os.DirFS("web/dist") for development.
//
//go:embed all:dist
var embeddedFS embed.FS

// FrontendSubFS returns a sub-filesystem for the frontend, or nil if not embedded.
func FrontendSubFS() fs.FS {
	sub, err := fs.Sub(embeddedFS, "dist")
	if err != nil {
		return nil
	}
	// Check for real content (not just a placeholder)
	if _, err := sub.Open("favicon.png"); err != nil {
		return nil
	}
	return sub
}
