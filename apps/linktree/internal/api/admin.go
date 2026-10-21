package api

import (
	"errors"
	"io"
	"math/rand/v2"
	"net/http"
	"strings"
	"unicode/utf8"

	"zazi.cool/linktree/internal/auth"
	"zazi.cool/linktree/internal/images"
	"zazi.cool/linktree/internal/pages"
	"zazi.cool/linktree/internal/store"
)

func (s *server) login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if !decodeJSON(w, r, maxSmallBody, &req) {
		return
	}
	err := s.cfg.Auth.Login(r.Context(), w, req.Password)
	if errors.Is(err, auth.ErrBadPassword) {
		writeError(w, http.StatusUnauthorized, CodeBadPassword, "Mot de passe incorrect.")
		return
	}
	if err != nil {
		s.internalError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, SessionResponse{Authenticated: true})
}

func (s *server) logout(w http.ResponseWriter, r *http.Request) {
	if err := s.cfg.Auth.Logout(r.Context(), w, r); err != nil {
		s.internalError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *server) session(w http.ResponseWriter, r *http.Request) {
	// requireAuth already answered 401 if there is no valid session.
	writeJSON(w, http.StatusOK, SessionResponse{Authenticated: true})
}

func (s *server) listPages(w http.ResponseWriter, r *http.Request) {
	list, err := s.cfg.Store.Pages(r.Context())
	if err != nil {
		s.internalError(w, r, err)
		return
	}
	resp := PagesResponse{Pages: make([]PageSummary, len(list))}
	for i, p := range list {
		resp.Pages[i] = toSummary(p)
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *server) createPage(w http.ResponseWriter, r *http.Request) {
	var req CreatePageRequest
	if !decodeJSON(w, r, maxSmallBody, &req) {
		return
	}
	title, slug := strings.TrimSpace(req.Title), strings.TrimSpace(req.Slug)
	errs := pages.Errors{}
	if n := utf8.RuneCountInString(title); n == 0 || n > pages.MaxTitle {
		errs["title"] = "Le titre fait de 1 à 80 caractères."
	}
	if msg := pages.ValidateSlug(slug); msg != "" {
		errs["slug"] = msg
	}
	if len(errs) > 0 {
		writeValidation(w, errs)
		return
	}

	p, err := s.cfg.Store.CreatePage(r.Context(), title, slug, s.randomShape())
	if errors.Is(err, store.ErrSlugTaken) {
		writeValidation(w, pages.Errors{"slug": "Ce slug est déjà utilisé."})
		return
	}
	if err != nil {
		s.internalError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, toDocument(p))
}

// randomShape picks the avatar proposed for a new page.
func (s *server) randomShape() string {
	if len(s.cfg.Shapes) == 0 {
		return ""
	}
	// Map iteration order is not something to rely on for randomness: count down
	// a random index instead.
	n := rand.IntN(len(s.cfg.Shapes))
	for id := range s.cfg.Shapes {
		if n == 0 {
			return id
		}
		n--
	}
	return ""
}

// pageID reads the {id} of the path; an invalid UUID cannot exist.
func pageID(w http.ResponseWriter, r *http.Request) (string, bool) {
	id := r.PathValue("id")
	if !pages.IsUUID(id) {
		writeError(w, http.StatusNotFound, CodeNotFound, "Page introuvable.")
		return "", false
	}
	return id, true
}

func (s *server) getPage(w http.ResponseWriter, r *http.Request) {
	id, ok := pageID(w, r)
	if !ok {
		return
	}
	p, err := s.cfg.Store.Page(r.Context(), id)
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, CodeNotFound, "Page introuvable.")
		return
	}
	if err != nil {
		s.internalError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, toDocument(p))
}

func (s *server) savePage(w http.ResponseWriter, r *http.Request) {
	id, ok := pageID(w, r)
	if !ok {
		return
	}
	var doc Document
	if !decodeJSON(w, r, maxDocumentBody, &doc) {
		return
	}
	if doc.ID != id {
		writeError(w, http.StatusBadRequest, CodeBadRequest, "L'identifiant du document ne correspond pas à l'URL.")
		return
	}

	p, errs := fromDocument(doc)
	pages.Normalize(&p)
	// Merge both sets of errors: the admin sees everything to fix at once.
	for field, msg := range pages.Validate(p, s.cfg.Shapes) {
		errs[field] = msg
	}
	if len(errs) > 0 {
		writeValidation(w, errs)
		return
	}

	version, err := s.cfg.Store.SavePage(r.Context(), p)
	switch {
	case err == nil:
		writeJSON(w, http.StatusOK, SaveResponse{Version: version})
	case errors.Is(err, store.ErrNotFound):
		writeError(w, http.StatusNotFound, CodeNotFound, "Page introuvable.")
	case errors.Is(err, store.ErrVersionConflict):
		writeError(w, http.StatusConflict, CodeConflict,
			"La page a été modifiée ailleurs (un autre onglet ?). Rechargez-la avant d'enregistrer.")
	case errors.Is(err, store.ErrSlugTaken):
		writeValidation(w, pages.Errors{"slug": "Ce slug est déjà utilisé."})
	case errors.Is(err, store.ErrForeignItem):
		writeValidation(w, pages.Errors{"items": "Un élément appartient à une autre page."})
	case errors.Is(err, store.ErrImageNotFound):
		writeValidation(w, pages.Errors{"images": "Une image n'existe plus : envoyez-la à nouveau."})
	default:
		s.internalError(w, r, err)
	}
}

func (s *server) deletePage(w http.ResponseWriter, r *http.Request) {
	id, ok := pageID(w, r)
	if !ok {
		return
	}
	err := s.cfg.Store.DeletePage(r.Context(), id)
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, CodeNotFound, "Page introuvable.")
		return
	}
	if err != nil {
		s.internalError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *server) reorderPages(w http.ResponseWriter, r *http.Request) {
	var req ReorderRequest
	if !decodeJSON(w, r, maxSmallBody, &req) {
		return
	}
	for _, id := range req.IDs {
		if !pages.IsUUID(id) {
			writeValidation(w, pages.Errors{"ids": "Identifiant de page invalide."})
			return
		}
	}
	err := s.cfg.Store.ReorderPages(r.Context(), req.IDs)
	if errors.Is(err, store.ErrInvalidOrder) {
		writeValidation(w, pages.Errors{"ids": "L'ordre doit lister chaque page une fois (rechargez la liste)."})
		return
	}
	if err != nil {
		s.internalError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *server) checkSlug(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")
	resp := SlugCheckResponse{Slug: slug, Error: pages.ValidateSlug(slug)}
	if resp.Error == "" {
		except := r.URL.Query().Get("except")
		if except != "" && !pages.IsUUID(except) {
			writeError(w, http.StatusBadRequest, CodeBadRequest, "Paramètre except invalide.")
			return
		}
		available, err := s.cfg.Store.SlugAvailable(r.Context(), slug, except)
		if err != nil {
			s.internalError(w, r, err)
			return
		}
		resp.Available = available
		if !available {
			resp.Error = "Ce slug est déjà utilisé."
		}
	}
	writeJSON(w, http.StatusOK, resp)
}

// uploadImage reads the "file" part of a multipart body. The parts are streamed
// rather than parsed with ParseMultipartForm, which spills big files to a
// temporary directory: the container's file system is read-only.
func (s *server) uploadImage(w http.ResponseWriter, r *http.Request) {
	limit := s.cfg.MaxUploadBytes
	// Headroom for the multipart boundaries and headers around the file.
	r.Body = http.MaxBytesReader(w, r.Body, limit+64<<10)
	mr, err := r.MultipartReader()
	if err != nil {
		writeError(w, http.StatusBadRequest, CodeBadRequest, "Envoi multipart/form-data attendu.")
		return
	}

	var data []byte
	for {
		part, err := mr.NextPart()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			uploadError(w, err)
			return
		}
		if part.FormName() != "file" {
			continue
		}
		// Reading one byte past the limit tells "exactly the limit" from "too big".
		data, err = io.ReadAll(io.LimitReader(part, limit+1))
		if err != nil {
			uploadError(w, err)
			return
		}
		break
	}
	if data == nil {
		writeError(w, http.StatusBadRequest, CodeBadRequest, "Champ « file » manquant.")
		return
	}
	if int64(len(data)) > limit {
		writeError(w, http.StatusRequestEntityTooLarge, CodeTooLarge, "Image trop lourde.")
		return
	}

	img, err := images.Process(data)
	switch {
	case errors.Is(err, images.ErrFormat):
		writeValidation(w, pages.Errors{"file": "Format non pris en charge : JPEG ou PNG uniquement."})
		return
	case errors.Is(err, images.ErrTooLarge):
		writeValidation(w, pages.Errors{"file": "Image trop grande : 2048 × 2048 pixels au plus."})
		return
	case err != nil:
		s.internalError(w, r, err)
		return
	}

	id, err := s.cfg.Store.CreateImage(r.Context(), img)
	if err != nil {
		s.internalError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, ImageResponse{Visual: Visual{
		Kind: VisualImage, ID: id, URL: imageURL(id), Width: img.Width, Height: img.Height,
	}})
}

func uploadError(w http.ResponseWriter, err error) {
	var tooLarge *http.MaxBytesError
	if errors.As(err, &tooLarge) {
		writeError(w, http.StatusRequestEntityTooLarge, CodeTooLarge, "Image trop lourde.")
		return
	}
	writeError(w, http.StatusBadRequest, CodeBadRequest, "Envoi interrompu ou invalide.")
}
