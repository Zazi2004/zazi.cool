// Package web embeds the HTML templates and the static files of the site, and
// gives the static assets fingerprinted names so they can be cached for a year.
package web

import (
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"fmt"
	"html/template"
	"io/fs"
	"path"
	"strings"
)

//go:embed templates/*.html
var templateFS embed.FS

//go:embed static
var staticFS embed.FS

// File is an embedded file ready to be served.
type File struct {
	Name        string
	ContentType string
	Data        []byte
}

// contentTypes covers every extension of static/: distroless images have no
// /etc/mime.types, and Go's built-in table lacks woff2.
var contentTypes = map[string]string{
	".css":   "text/css; charset=utf-8",
	".woff2": "font/woff2",
	".svg":   "image/svg+xml",
	".png":   "image/png",
	".txt":   "text/plain; charset=utf-8",
}

// Assets maps the files of static/assets to fingerprinted names
// ("home.css" → "home.0123456789.css"), and serves the root files of static/.
type Assets struct {
	urls   map[string]string // asset name → "/assets/<fingerprinted name>"
	hashed map[string]File   // fingerprinted name → file
	root   map[string]File   // "favicon.svg", "robots.txt"…
}

// LoadAssets reads the embedded static files.
func LoadAssets() (*Assets, error) {
	a := &Assets{urls: map[string]string{}, hashed: map[string]File{}, root: map[string]File{}}

	entries, err := fs.ReadDir(staticFS, "static/assets")
	if err != nil {
		return nil, fmt.Errorf("read assets: %w", err)
	}
	// CSS files reference the other assets with url("name"): those are named first,
	// so that the CSS can point to their fingerprinted names (and its own
	// fingerprint changes when one of them does).
	var styles []string
	for _, e := range entries {
		if path.Ext(e.Name()) == ".css" {
			styles = append(styles, e.Name())
			continue
		}
		data, err := fs.ReadFile(staticFS, "static/assets/"+e.Name())
		if err != nil {
			return nil, fmt.Errorf("read asset: %w", err)
		}
		if err := a.add(e.Name(), data); err != nil {
			return nil, err
		}
	}
	for _, name := range styles {
		data, err := fs.ReadFile(staticFS, "static/assets/"+name)
		if err != nil {
			return nil, fmt.Errorf("read asset: %w", err)
		}
		css := string(data)
		for asset, url := range a.urls {
			css = strings.ReplaceAll(css, `url("`+asset+`")`, `url("`+path.Base(url)+`")`)
		}
		if err := a.add(name, []byte(css)); err != nil {
			return nil, err
		}
	}

	entries, err = fs.ReadDir(staticFS, "static")
	if err != nil {
		return nil, fmt.Errorf("read static: %w", err)
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		data, err := fs.ReadFile(staticFS, "static/"+e.Name())
		if err != nil {
			return nil, fmt.Errorf("read static file: %w", err)
		}
		ct, err := contentType(e.Name())
		if err != nil {
			return nil, err
		}
		a.root[e.Name()] = File{Name: e.Name(), ContentType: ct, Data: data}
	}
	return a, nil
}

func (a *Assets) add(name string, data []byte) error {
	ct, err := contentType(name)
	if err != nil {
		return err
	}
	sum := sha256.Sum256(data)
	ext := path.Ext(name)
	hashed := strings.TrimSuffix(name, ext) + "." + hex.EncodeToString(sum[:5]) + ext
	a.urls[name] = "/assets/" + hashed
	a.hashed[hashed] = File{Name: hashed, ContentType: ct, Data: data}
	return nil
}

func contentType(name string) (string, error) {
	ct, ok := contentTypes[path.Ext(name)]
	if !ok {
		return "", fmt.Errorf("static file %q: unknown content type", name)
	}
	return ct, nil
}

// URL returns the root-relative URL of an asset. An unknown name is a programming
// error, caught by the tests that render every template.
func (a *Assets) URL(name string) (string, error) {
	url, ok := a.urls[name]
	if !ok {
		return "", fmt.Errorf("unknown asset %q", name)
	}
	return url, nil
}

// Asset returns the asset with the given fingerprinted name.
func (a *Assets) Asset(hashed string) (File, bool) {
	f, ok := a.hashed[hashed]
	return f, ok
}

// Root returns a file served at the root of the site, such as "robots.txt".
func (a *Assets) Root(name string) (File, bool) {
	f, ok := a.root[name]
	return f, ok
}

// Templates parses the page templates; {{asset "home.css"}} gives an asset URL.
func Templates(a *Assets) (*template.Template, error) {
	t, err := template.New("").Funcs(template.FuncMap{"asset": a.URL}).ParseFS(templateFS, "templates/*.html")
	if err != nil {
		return nil, fmt.Errorf("parse templates: %w", err)
	}
	return t, nil
}
