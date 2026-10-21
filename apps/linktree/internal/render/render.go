package render

import (
	"bytes"
	"embed"
	"fmt"
	"html/template"
	"net/http"
)

//go:embed templates/*.html
var templates embed.FS

// Source paths of the Vite entry points (see web/vite.config.ts).
const (
	publicEntry = "src/public/main.tsx"
	adminEntry  = "src/admin/main.tsx"
)

// Shell describes one HTML page. Every string is escaped by html/template.
type Shell struct {
	Title       string
	Description string
	URL         string // absolute canonical URL (og:url)
	Image       string // absolute URL of the Open Graph image
	NoIndex     bool
	Theme       string // data-theme on <html>, so the CSS background is right before JS runs
	Accent      string
	// Data is serialized as JSON in <script id="page-data"> for the React app.
	// html/template escapes it, so a "</script>" inside a title cannot end the tag.
	Data     any
	Fallback *Fallback
	Entry    Entry // set by the Renderer
}

// Fallback is the plain HTML version of a page, shown when JavaScript is off.
type Fallback struct {
	Heading string
	Text    string
	Items   []FallbackItem
}

// FallbackItem is a link, or a group of links when Links is not empty.
type FallbackItem struct {
	Title string
	// URL must already be validated (http, https, mailto or tel only): template.URL
	// tells html/template not to filter it, which it would otherwise do for tel:.
	URL   template.URL
	Links []FallbackItem
}

// Renderer writes the HTML shells of the public pages and of the admin.
type Renderer struct {
	assets *Assets
	tmpl   *template.Template
}

// New parses the templates; assets locates the Vite bundles.
func New(assets *Assets) (*Renderer, error) {
	tmpl, err := template.ParseFS(templates, "templates/shell.html")
	if err != nil {
		return nil, fmt.Errorf("parse templates: %w", err)
	}
	return &Renderer{assets: assets, tmpl: tmpl}, nil
}

// Public writes a public page (index, links page or 404) with the given status.
func (r *Renderer) Public(w http.ResponseWriter, status int, s Shell) error {
	return r.write(w, status, publicEntry, s)
}

// Admin writes the shell of the admin single-page app.
func (r *Renderer) Admin(w http.ResponseWriter) error {
	return r.write(w, http.StatusOK, adminEntry, Shell{Title: "Administration — link.zazi.cool", NoIndex: true})
}

func (r *Renderer) write(w http.ResponseWriter, status int, entry string, s Shell) error {
	e, err := r.assets.Entry(entry)
	if err != nil {
		return err
	}
	s.Entry = e

	// Rendering into a buffer first: a template error then becomes a clean 500
	// instead of a half-written page with a 200 status.
	var buf bytes.Buffer
	if err := r.tmpl.ExecuteTemplate(&buf, "shell.html", s); err != nil {
		return fmt.Errorf("render shell: %w", err)
	}
	h := w.Header()
	h.Set("Content-Type", "text/html; charset=utf-8")
	// The HTML embeds the current data and the current bundle names: never cache it.
	h.Set("Cache-Control", "no-cache")
	w.WriteHeader(status)
	_, err = buf.WriteTo(w)
	return err
}
