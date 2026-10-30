package api

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"net/http"
	"strconv"

	"zazi.cool/linktree/internal/pages"
	"zazi.cool/linktree/internal/render"
	"zazi.cool/linktree/internal/store"
)

const (
	indexTitle       = "Liens — zazi.cool"
	indexDescription = "Les pages de liens de zazi.cool."
	defaultOGImage   = "/og-default.png"
)

func (s *server) index(w http.ResponseWriter, r *http.Request) {
	list, err := s.cfg.Store.PublicPages(r.Context())
	if err != nil {
		s.pageError(w, r, err)
		return
	}
	entries := make([]IndexEntry, len(list))
	for i, p := range list {
		entries[i] = IndexEntry{Slug: p.Slug, Title: p.Title, Description: p.Description, Avatar: toVisual(p.Avatar)}
	}
	s.renderPublic(w, r, http.StatusOK, render.Shell{
		Title:       indexTitle,
		Description: indexDescription,
		URL:         s.cfg.PublicBaseURL + "/",
		Image:       s.cfg.PublicBaseURL + defaultOGImage,
		Theme:       pages.DefaultTheme,
		Accent:      pages.DefaultAccent,
		Data:        IndexView{View: ViewIndex, Pages: entries},
		Fallback:    indexFallback(list),
	})
}

func (s *server) page(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")
	// An invalid slug cannot exist: no need to ask the database.
	if pages.ValidateSlug(slug) != "" {
		s.notFound(w, r)
		return
	}
	p, err := s.cfg.Store.PublishedPage(r.Context(), slug)
	if errors.Is(err, store.ErrNotFound) {
		s.notFound(w, r)
		return
	}
	if err != nil {
		s.pageError(w, r, err)
		return
	}

	// Drafts are removed here, on the server: they never reach the browser.
	p = p.Visible()
	image := s.cfg.PublicBaseURL + defaultOGImage
	if p.Avatar.ImageID != "" {
		image = s.cfg.PublicBaseURL + imageURL(p.Avatar.ImageID)
	}
	s.renderPublic(w, r, http.StatusOK, render.Shell{
		Title:       p.Title,
		Description: p.Description,
		URL:         s.cfg.PublicBaseURL + "/" + p.Slug,
		Image:       image,
		NoIndex:     !p.Listed,
		Theme:       p.Theme,
		Accent:      p.Accent,
		Data:        PageView{View: ViewPage, Page: toPublicPage(p)},
		Fallback:    pageFallback(p),
	})
}

func (s *server) notFound(w http.ResponseWriter, r *http.Request) {
	s.renderPublic(w, r, http.StatusNotFound, render.Shell{
		Title:    "Page introuvable — zazi.cool",
		NoIndex:  true,
		Theme:    pages.DefaultTheme,
		Accent:   pages.DefaultAccent,
		Data:     NotFoundView{View: ViewNotFound},
		Fallback: &render.Fallback{Heading: "Page introuvable", Items: []render.FallbackItem{{Title: "Toutes les pages", URL: "/"}}},
	})
}

func (s *server) renderPublic(w http.ResponseWriter, r *http.Request, status int, shell render.Shell) {
	if err := s.cfg.Renderer.Public(w, status, shell); err != nil {
		s.pageError(w, r, err)
	}
}

// pageError answers an HTML request that failed: a JSON error would make no sense
// to a visitor.
func (s *server) pageError(w http.ResponseWriter, r *http.Request, err error) {
	logError(r, err)
	http.Error(w, "Erreur interne, réessayez plus tard.", http.StatusInternalServerError)
}

func (s *server) image(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if !pages.IsUUID(id) {
		http.NotFound(w, r)
		return
	}
	img, err := s.cfg.Store.Image(r.Context(), id)
	if errors.Is(err, store.ErrNotFound) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		s.pageError(w, r, err)
		return
	}
	h := w.Header()
	h.Set("Content-Type", img.ContentType)
	h.Set("Content-Length", strconv.Itoa(len(img.Data)))
	// An image never changes: a new upload gets a new ID.
	h.Set("Cache-Control", "public, max-age=31536000, immutable")
	if _, err := w.Write(img.Data); err != nil {
		logError(r, err)
	}
}

func (s *server) adminShell(w http.ResponseWriter, r *http.Request) {
	// In development the CSP already allows inline styles (Vite injects them), and
	// a nonce would make browsers ignore 'unsafe-inline': no nonce then.
	nonce := ""
	if s.cfg.DevURL == "" {
		nonce = newNonce()
		w.Header().Set("Content-Security-Policy", adminCSP(nonce))
	}
	if err := s.cfg.Renderer.Admin(w, nonce); err != nil {
		s.pageError(w, r, err)
	}
}

// newNonce returns 128 random bits, base64url-encoded, for a CSP nonce. The URL-safe
// alphabet has no "+", which html/template would escape in the page's meta tag.
func newNonce() string {
	b := make([]byte, 16)
	// crypto/rand.Read never returns an error on supported platforms.
	_, _ = rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}
