package server

import (
	"bytes"
	"net/http"
	"path"
	"time"

	"zazi.cool/home/internal/web"
)

// svgCSP applies to the SVG files: the coolshapes use inline style attributes,
// which the page CSP would block. An SVG shown by <img> never runs scripts.
const svgCSP = "default-src 'none'; style-src 'unsafe-inline'"

// asset serves a fingerprinted file of /assets: its name changes with its
// content, so it can be cached for a year.
func (s *server) asset(w http.ResponseWriter, r *http.Request) {
	f, ok := s.assets.Asset(r.PathValue("file"))
	if !ok {
		s.notFound(w, r)
		return
	}
	w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	serveFile(w, r, f)
}

// rootFile serves a file whose URL cannot change (favicon.svg, og-default.png,
// robots.txt): it is cached for a day only.
func (s *server) rootFile(name string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		f, ok := s.assets.Root(name)
		if !ok {
			s.notFound(w, r)
			return
		}
		w.Header().Set("Cache-Control", "public, max-age=86400")
		serveFile(w, r, f)
	}
}

func serveFile(w http.ResponseWriter, r *http.Request, f web.File) {
	w.Header().Set("Content-Type", f.ContentType)
	if path.Ext(f.Name) == ".svg" {
		w.Header().Set("Content-Security-Policy", svgCSP)
	}
	// ServeContent handles HEAD, Range and conditional requests. The files are
	// embedded, so they have no modification time.
	http.ServeContent(w, r, f.Name, time.Time{}, bytes.NewReader(f.Data))
}
