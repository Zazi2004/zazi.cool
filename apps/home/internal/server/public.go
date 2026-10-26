package server

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"zazi.cool/home/internal/content"
)

// ContentLoader reads the homepage content; *content.Store satisfies it.
type ContentLoader interface {
	Load(ctx context.Context) (content.Content, error)
}

// descriptionLength bounds og:description (F6).
const descriptionLength = 200

// page is the data of the templates. html/template escapes every string
// according to where it is written (text, attribute, URL).
type page struct {
	Title       string // <title> and og:title
	Description string
	URL         string // absolute; the Open Graph tags are only written when set
	Image       string // absolute URL of og:image
	ImageAlt    string
	NoIndex     bool
	Background  string // data-background, read by the bootstrap script of the animated backgrounds
	Script      string // URL of that script; empty for the static background (no script at all)

	Heading    string
	Banner     *content.Banner
	Paragraphs [][]string
	Services   []content.Service
}

func (s *server) home(w http.ResponseWriter, r *http.Request) {
	c, err := s.cfg.Content.Load(r.Context())
	if err != nil {
		s.internalError(w, r, err)
		return
	}
	p := page{
		Title:       c.Title,
		Description: content.Summary(c.Body, descriptionLength),
		URL:         s.cfg.PublicBaseURL + "/",
		Image:       s.cfg.PublicBaseURL + "/og-default.png",
		Background:  c.BackgroundOrStatic(),
		Heading:     c.Title,
		Banner:      c.Banner(s.assets.IllustrationURL),
		Paragraphs:  content.Paragraphs(c.Body),
		Services:    content.Services,
	}
	if p.Background != content.StaticBackground {
		p.Script = s.assets.Script()
	}
	if p.Banner != nil {
		p.Image = s.cfg.PublicBaseURL + p.Banner.URL
		p.ImageAlt = p.Banner.Alt
	}
	s.renderPage(w, r, http.StatusOK, "home.html", p)
}

// notFound answers every path that no other route matches (F7).
func (s *server) notFound(w http.ResponseWriter, r *http.Request) {
	s.renderPage(w, r, http.StatusNotFound, "notfound.html", page{
		Title:      "Page introuvable — zazi.cool",
		NoIndex:    true,
		Background: content.StaticBackground,
	})
}

// renderPage writes a public page; admin pages use render (admin.go).
func (s *server) renderPage(w http.ResponseWriter, r *http.Request, status int, name string, p page) {
	// Rendering into a buffer first: a template error then becomes a clean 500
	// instead of a half-written page with a 200 status.
	var buf bytes.Buffer
	if err := s.tmpl.ExecuteTemplate(&buf, name, p); err != nil {
		s.internalError(w, r, fmt.Errorf("render %s: %w", name, err))
		return
	}
	h := w.Header()
	h.Set("Content-Type", "text/html; charset=utf-8")
	// The content may change at any time (F11): always revalidate.
	h.Set("Cache-Control", "no-cache")
	w.WriteHeader(status)
	if _, err := buf.WriteTo(w); err != nil {
		slog.Warn("write response", "path", r.URL.Path, "error", err)
	}
}
