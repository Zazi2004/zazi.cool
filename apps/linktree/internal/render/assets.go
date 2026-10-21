// Package render writes the HTML shells of the pages: meta tags, the page data as
// JSON, a <noscript> fallback and the tags loading the Vite bundles.
package render

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"slices"
	"strings"
)

// Entry lists the files to load for one Vite entry point, as root-relative URLs.
type Entry struct {
	Scripts  []string // module scripts, in order
	Styles   []string // stylesheets
	Preloads []string // chunks imported by the scripts, fetched early
}

// Assets resolves the files of the Vite entry points, either from the build
// manifest (production) or from the Vite dev server (development).
type Assets struct {
	devURL  string
	entries map[string]Entry // by source path, e.g. "src/public/main.tsx"
}

// manifestChunk is one entry of dist/.vite/manifest.json (only the fields we use).
type manifestChunk struct {
	File    string   `json:"file"`
	IsEntry bool     `json:"isEntry"`
	Imports []string `json:"imports"`
	CSS     []string `json:"css"`
}

// FromManifest reads the manifest written by `vite build` (build.manifest: true).
func FromManifest(dist fs.FS) (*Assets, error) {
	data, err := fs.ReadFile(dist, ".vite/manifest.json")
	if err != nil {
		return nil, fmt.Errorf("read Vite manifest (run `npm run build`?): %w", err)
	}
	var manifest map[string]manifestChunk
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, fmt.Errorf("parse Vite manifest: %w", err)
	}

	a := &Assets{entries: make(map[string]Entry)}
	for src, chunk := range manifest {
		if !chunk.IsEntry {
			continue
		}
		e := Entry{Scripts: []string{"/" + chunk.File}}
		seen := map[string]bool{}
		collect(manifest, src, seen, &e)
		a.entries[src] = e
	}
	return a, nil
}

// collect walks the static imports of a chunk to gather their CSS and files.
func collect(manifest map[string]manifestChunk, key string, seen map[string]bool, e *Entry) {
	if seen[key] {
		return
	}
	seen[key] = true
	chunk := manifest[key]
	for _, css := range chunk.CSS {
		if url := "/" + css; !slices.Contains(e.Styles, url) {
			e.Styles = append(e.Styles, url)
		}
	}
	for _, imp := range chunk.Imports {
		if url := "/" + manifest[imp].File; !slices.Contains(e.Preloads, url) {
			e.Preloads = append(e.Preloads, url)
		}
		collect(manifest, imp, seen, e)
	}
}

// FromDevServer loads the entry points from the Vite dev server, with hot reload.
func FromDevServer(url string) *Assets {
	return &Assets{devURL: strings.TrimSuffix(url, "/")}
}

// DevURL returns the Vite dev server URL, or "" in production.
func (a *Assets) DevURL() string {
	return a.devURL
}

// Entry returns the files of the entry point whose source is src.
func (a *Assets) Entry(src string) (Entry, error) {
	if a.devURL != "" {
		// In development CSS is injected by the JS modules themselves.
		return Entry{Scripts: []string{a.devURL + "/@vite/client", a.devURL + "/" + src}}, nil
	}
	e, ok := a.entries[src]
	if !ok {
		return Entry{}, fmt.Errorf("entry %q not found in the Vite manifest", src)
	}
	return e, nil
}
