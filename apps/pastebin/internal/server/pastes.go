package server

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"
	_ "time/tzdata" // the distroless image has no time zone database
	"unicode/utf8"

	"zazi.cool/pastebin/internal/paste"
	"zazi.cool/pastebin/internal/store"
)

// formOverhead is the room left in the request body for the other form fields.
const formOverhead = 4 << 10

// displayZone is the time zone of the dates shown to users.
var displayZone = mustLoadLocation("Europe/Paris")

func mustLoadLocation(name string) *time.Location {
	loc, err := time.LoadLocation(name)
	if err != nil {
		panic(err) // time/tzdata is embedded: this cannot fail at run time
	}
	return loc
}

// formPage is the data of the creation form.
type formPage struct {
	Languages   []paste.Option
	Expirations []paste.Expiration
	MaxKiB      int
	Content     string
	Language    string
	ExpiresIn   string
	Error       string
}

func (h *handler) newFormPage() formPage {
	return formPage{
		Languages:   paste.Languages,
		Expirations: paste.Expirations,
		MaxKiB:      h.cfg.MaxPasteBytes / 1024,
		Language:    paste.DefaultLanguage,
		ExpiresIn:   paste.DefaultExpiration,
	}
}

// form serves the creation form (F1).
func (h *handler) form(w http.ResponseWriter, _ *http.Request) {
	render(w, http.StatusOK, "form", h.newFormPage())
}

// create validates the form, stores the paste and redirects to it (F2, F7).
func (h *handler) create(w http.ResponseWriter, r *http.Request) {
	page := h.newFormPage()

	// URL encoding can triple the size of non-ASCII text (one byte becomes "%XX"), so
	// the body limit is three times the content limit. The content itself is checked
	// against MaxPasteBytes once decoded.
	r.Body = http.MaxBytesReader(w, r.Body, int64(3*h.cfg.MaxPasteBytes+formOverhead))
	if err := r.ParseForm(); err != nil {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			page.Error = h.tooLargeMessage()
			render(w, http.StatusRequestEntityTooLarge, "form", page)
			return
		}
		page.Error = "Le formulaire envoyé est invalide."
		render(w, http.StatusBadRequest, "form", page)
		return
	}

	page.Content = r.PostForm.Get("content")
	if v := r.PostForm.Get("language"); v != "" {
		page.Language = v
	}
	if v := r.PostForm.Get("expires_in"); v != "" {
		page.ExpiresIn = v
	}

	if status, msg := h.validate(page); msg != "" {
		page.Error = msg
		page.Content = strings.ToValidUTF8(page.Content, "�")
		render(w, status, "form", page)
		return
	}

	exp, _ := paste.LookupExpiration(page.ExpiresIn) // checked by validate
	p, err := h.cfg.Store.Create(r.Context(), page.Content, page.Language, exp.Duration)
	if err != nil {
		slog.ErrorContext(r.Context(), "create paste", "error", err)
		page.Error = "Le paste n'a pas pu être enregistré. Réessayez dans un instant."
		render(w, http.StatusInternalServerError, "form", page)
		return
	}
	slog.InfoContext(r.Context(), "paste created", "id", p.ID, "bytes", len(p.Content), "expires_in", exp.Key)
	http.Redirect(w, r, "/"+p.ID, http.StatusSeeOther)
}

// validate returns the HTTP status and the French message of the first problem
// found in the form, or an empty message when the form is valid.
func (h *handler) validate(page formPage) (int, string) {
	switch {
	case strings.TrimSpace(page.Content) == "":
		return http.StatusUnprocessableEntity, "Le paste est vide."
	case len(page.Content) > h.cfg.MaxPasteBytes:
		return http.StatusRequestEntityTooLarge, h.tooLargeMessage()
	case !utf8.ValidString(page.Content):
		return http.StatusUnprocessableEntity, "Le contenu doit être du texte UTF-8 valide."
	case !paste.ValidLanguage(page.Language):
		return http.StatusUnprocessableEntity, "Ce langage n'est pas proposé."
	}
	if _, ok := paste.LookupExpiration(page.ExpiresIn); !ok {
		return http.StatusUnprocessableEntity, "Cette durée d'expiration n'est pas proposée."
	}
	return 0, ""
}

func (h *handler) tooLargeMessage() string {
	return fmt.Sprintf("Le paste dépasse la taille maximale de %d Kio.", h.cfg.MaxPasteBytes/1024)
}

// viewPage is the data of the formatted view.
type viewPage struct {
	ID        string
	URL       string
	Language  string
	Lines     []string
	ExpiresAt string
}

// view serves the formatted view of a paste (F3).
func (h *handler) view(w http.ResponseWriter, r *http.Request) {
	p, ok := h.lookup(w, r)
	if !ok {
		return
	}
	render(w, http.StatusOK, "view", viewPage{
		ID:        p.ID,
		URL:       h.cfg.BaseURL + "/" + p.ID,
		Language:  languageLabel(p.Language),
		Lines:     lines(p.Content),
		ExpiresAt: p.ExpiresAt.In(displayZone).Format("02/01/2006 à 15:04"),
	})
}

// raw serves the paste exactly as it was sent, as plain text (F4).
func (h *handler) raw(w http.ResponseWriter, r *http.Request) {
	p, ok := h.lookup(w, r)
	if !ok {
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Content-Security-Policy", rawCSP)
	if _, err := io.WriteString(w, p.Content); err != nil {
		slog.Warn("write response", "error", err)
	}
}

// lookup loads the paste named by the URL. When it cannot, it writes the response
// itself and returns false. Unknown, malformed and expired identifiers all get the
// same 404 (F5).
func (h *handler) lookup(w http.ResponseWriter, r *http.Request) (paste.Paste, bool) {
	id := r.PathValue("id")
	if !paste.ValidID(id) {
		render(w, http.StatusNotFound, "notfound", nil)
		return paste.Paste{}, false
	}
	p, err := h.cfg.Store.Get(r.Context(), id)
	if errors.Is(err, store.ErrNotFound) {
		render(w, http.StatusNotFound, "notfound", nil)
		return paste.Paste{}, false
	}
	if err != nil {
		slog.ErrorContext(r.Context(), "get paste", "error", err)
		http.Error(w, "Erreur interne", http.StatusInternalServerError)
		return paste.Paste{}, false
	}
	return p, true
}

// lines splits content for display with line numbers. A final line break does not
// open an extra empty line, and Windows line endings lose their "\r".
func lines(content string) []string {
	content = strings.TrimSuffix(content, "\n")
	out := strings.Split(content, "\n")
	for i, l := range out {
		out[i] = strings.TrimSuffix(l, "\r")
	}
	return out
}

func languageLabel(key string) string {
	for _, l := range paste.Languages {
		if l.Key == key {
			return l.Label
		}
	}
	return key
}
