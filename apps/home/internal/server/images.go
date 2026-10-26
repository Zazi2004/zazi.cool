package server

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"zazi.cool/home/internal/content"
)

// image serves an uploaded banner (F15). The identifier changes with every new
// banner, so the response can be cached for a year and never revalidated.
func (s *server) image(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if !content.IsUUID(id) {
		s.notFound(w, r)
		return
	}
	img, err := s.cfg.Content.Image(r.Context(), id)
	if errors.Is(err, content.ErrImageNotFound) {
		s.notFound(w, r)
		return
	}
	if err != nil {
		s.internalError(w, r, err)
		return
	}
	h := w.Header()
	h.Set("Content-Type", img.ContentType)
	h.Set("Content-Length", strconv.Itoa(len(img.Data)))
	h.Set("Cache-Control", "public, max-age=31536000, immutable")
	if _, err := w.Write(img.Data); err != nil {
		slog.Warn("write image", "path", r.URL.Path, "error", err)
	}
}
