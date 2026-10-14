// Package web embeds the front built by Vite (web/dist) into the Go binary.
// Run `npm run build` (or `make build-web`) first: `go build` fails if dist is missing.
package web

import (
	"embed"
	"io/fs"
)

// The "all:" prefix also embeds files starting with "." or "_".
//
//go:embed all:dist
var dist embed.FS

// Dist returns the built front, rooted at the dist directory.
func Dist() (fs.FS, error) {
	return fs.Sub(dist, "dist")
}
