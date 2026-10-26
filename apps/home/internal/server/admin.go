package server

import (
	"bytes"
	"context"
	"embed"
	"errors"
	"fmt"
	"html/template"
	"io"
	"log/slog"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"zazi.cool/home/internal/auth"
	"zazi.cool/home/internal/content"
	"zazi.cool/home/internal/images"
	"zazi.cool/home/internal/ratelimit"
)

// maxLoginBody bounds the login form: a password field and little else (NF8).
const maxLoginBody = 16 << 10

// adminCSP is the CSP of NF5. The admin has no script at all.
const adminCSP = "default-src 'none'; script-src 'self'; style-src 'self'; img-src 'self'; " +
	"font-src 'self'; form-action 'self'; base-uri 'none'; frame-ancestors 'none'"

//go:embed templates/admin.html
var adminFS embed.FS

var adminTemplates = template.Must(template.ParseFS(adminFS, "templates/admin.html"))

// ContentStore reads and saves the homepage content; *content.Store satisfies it.
type ContentStore interface {
	ContentLoader
	Save(ctx context.Context, version int, e content.Edit, img *images.Image) error
	Image(ctx context.Context, id string) (images.Image, error)
}

// loginPage is the data of the "login" template.
type loginPage struct {
	Error string
}

// adminRoutes registers the admin pages on mux.
func (s *server) adminRoutes(mux *http.ServeMux) {
	login := ratelimit.New(s.cfg.LoginLimit, time.Hour)
	mux.Handle("GET /admin", adminPage(http.HandlerFunc(s.admin)))
	mux.Handle("POST /admin", adminPage(s.sameOrigin(http.HandlerFunc(s.save))))
	mux.Handle("POST /admin/login", adminPage(s.sameOrigin(s.rateLimited(login, s.login))))
	mux.Handle("POST /admin/logout", adminPage(s.sameOrigin(http.HandlerFunc(s.logout))))
}

// admin shows the login form without a session, the admin page with one.
func (s *server) admin(w http.ResponseWriter, r *http.Request) {
	ok, err := s.cfg.Auth.Authenticated(r)
	if err != nil {
		s.internalError(w, r, err)
		return
	}
	if !ok {
		s.render(w, r, http.StatusOK, "login", loginPage{})
		return
	}
	c, err := s.cfg.Content.Load(r.Context())
	if err != nil {
		s.internalError(w, r, err)
		return
	}
	p := newEditPage(c, s.cfg.MaxUploadBytes)
	if r.URL.Query().Get("saved") == "1" {
		p.Notice = "Modifications enregistrées"
	}
	s.render(w, r, http.StatusOK, "dashboard", p)
}

// save validates the edit form and stores it (F11 to F13). The version the form
// carries is the one it was built from.
func (s *server) save(w http.ResponseWriter, r *http.Request) {
	ok, err := s.cfg.Auth.Authenticated(r)
	if err != nil {
		s.internalError(w, r, err)
		return
	}
	if !ok {
		s.render(w, r, http.StatusUnauthorized, "login", loginPage{})
		return
	}
	// A browser sends multipart/form-data; other clients may send urlencoded
	// forms, which ParseMultipartForm reports as ErrNotMultipart after parsing them.
	// The whole body is bounded by limitBody (NF8), so the parser may keep all of it
	// in RAM: the container has no writable disk for the temporary files it would
	// create above this size.
	if err := r.ParseMultipartForm(s.cfg.MaxUploadBytes + adminFormOverage); err != nil && !errors.Is(err, http.ErrNotMultipart) {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			http.Error(w, "Requête trop volumineuse.", http.StatusRequestEntityTooLarge)
			return
		}
		http.Error(w, "Formulaire invalide.", http.StatusBadRequest)
		return
	}
	version, err := strconv.Atoi(r.PostForm.Get("version"))
	if err != nil {
		http.Error(w, "Formulaire invalide.", http.StatusBadRequest)
		return
	}
	file, tooLarge, err := s.bannerFile(r)
	if err != nil {
		http.Error(w, "Formulaire invalide.", http.StatusBadRequest)
		return
	}
	if tooLarge {
		http.Error(w, "Image trop volumineuse.", http.StatusRequestEntityTooLarge)
		return
	}
	current, err := s.cfg.Content.Load(r.Context())
	if err != nil {
		s.internalError(w, r, err)
		return
	}
	edit, errs := content.Edit{
		Title:        r.PostForm.Get("title"),
		Body:         r.PostForm.Get("body"),
		BannerChoice: r.PostForm.Get("banner"),
		BannerAlt:    r.PostForm.Get("banner_alt"),
		Background:   r.PostForm.Get("background"),
	}.Validate(current.Upload != nil, file != nil)

	var img *images.Image
	if file != nil {
		processed, err := images.Process(file)
		switch {
		case errors.Is(err, images.ErrTooLarge):
			errs["banner_file"] = fmt.Sprintf("L'image dépasse %d × %d pixels.", images.MaxInputSide, images.MaxInputSide)
		case err != nil:
			errs["banner_file"] = "Le fichier doit être une image JPEG ou PNG."
		default:
			img = &processed
		}
	}

	p := newEditPage(current, s.cfg.MaxUploadBytes)
	p.fill(edit, version)
	p.Errors = errs
	if len(errs) > 0 {
		s.render(w, r, http.StatusUnprocessableEntity, "dashboard", p)
		return
	}
	err = s.cfg.Content.Save(r.Context(), version, edit, img)
	if errors.Is(err, content.ErrConflict) {
		p.Conflict = true
		s.render(w, r, http.StatusConflict, "dashboard", p)
		return
	}
	if err != nil {
		s.internalError(w, r, err)
		return
	}
	http.Redirect(w, r, "/admin?saved=1", http.StatusSeeOther)
}

// bannerFile returns the bytes of the file sent in the "banner_file" field, nil if
// there is none (an untouched file input sends an empty part, which Go does not
// count as a file). tooLarge is true when it is over MAX_UPLOAD_MB: the body limit
// leaves some slack for the other fields, so the file is measured on its own.
func (s *server) bannerFile(r *http.Request) (data []byte, tooLarge bool, err error) {
	f, header, err := r.FormFile("banner_file")
	if errors.Is(err, http.ErrMissingFile) || errors.Is(err, http.ErrNotMultipart) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	defer f.Close()
	if header.Size > s.cfg.MaxUploadBytes {
		return nil, true, nil
	}
	data, err = io.ReadAll(f)
	if err != nil {
		return nil, false, err
	}
	return data, false, nil
}

// backgroundLabels names the backgrounds of content.Backgrounds in the form.
var backgroundLabels = map[string]string{
	"aurore":   "Aurore",
	"fils":     "Fils lumineux",
	"grain":    "Dégradé mouvant",
	"statique": "Dégradé statique (sans animation)",
}

// BackgroundOptions lists the backgrounds with their labels, for the template.
func (editPage) BackgroundOptions() []backgroundOption {
	options := make([]backgroundOption, 0, len(content.Backgrounds))
	for _, id := range content.Backgrounds {
		options = append(options, backgroundOption{ID: id, Label: backgroundLabels[id]})
	}
	return options
}

type backgroundOption struct{ ID, Label string }

// editPage is the data of the "dashboard" template: the edit form.
type editPage struct {
	Title        string
	Body         string
	BannerChoice string
	BannerAlt    string
	Background   string
	Version      int
	Upload       *content.Image
	MaxUploadMB  int64
	Errors       map[string]string
	Notice       string
	Conflict     bool
}

func newEditPage(c content.Content, maxUploadBytes int64) editPage {
	return editPage{
		MaxUploadMB:  maxUploadBytes >> 20,
		Title:        c.Title,
		Body:         c.Body,
		BannerChoice: c.BannerChoice,
		BannerAlt:    c.BannerAlt,
		Background:   c.Background,
		Version:      c.Version,
		Upload:       c.Upload,
	}
}

// fill replaces the stored values by what the user submitted, so that a rejected
// form comes back with its input.
func (p *editPage) fill(e content.Edit, version int) {
	p.Title, p.Body, p.BannerChoice, p.BannerAlt, p.Background = e.Title, e.Body, e.BannerChoice, e.BannerAlt, e.Background
	p.Version = version
}

func (s *server) login(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxLoginBody)
	if err := r.ParseForm(); err != nil {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			http.Error(w, "Requête trop volumineuse.", http.StatusRequestEntityTooLarge)
			return
		}
		http.Error(w, "Formulaire invalide.", http.StatusBadRequest)
		return
	}
	err := s.cfg.Auth.Login(r.Context(), w, r.PostForm.Get("password"))
	if errors.Is(err, auth.ErrBadPassword) {
		s.render(w, r, http.StatusUnauthorized, "login", loginPage{Error: "Mot de passe incorrect."})
		return
	}
	if err != nil {
		s.internalError(w, r, err)
		return
	}
	http.Redirect(w, r, "/admin", http.StatusSeeOther)
}

func (s *server) logout(w http.ResponseWriter, r *http.Request) {
	if err := s.cfg.Auth.Logout(r.Context(), w, r); err != nil {
		s.internalError(w, r, err)
		return
	}
	http.Redirect(w, r, "/admin", http.StatusSeeOther)
}

// render executes an admin template into a buffer first, so that a template error
// becomes a clean 500 instead of a half-written page.
func (s *server) render(w http.ResponseWriter, r *http.Request, status int, name string, data any) {
	var buf bytes.Buffer
	if err := adminTemplates.ExecuteTemplate(&buf, name, data); err != nil {
		s.internalError(w, r, err)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	if _, err := buf.WriteTo(w); err != nil {
		slog.Warn("write response", "path", r.URL.Path, "error", err)
	}
}

func (s *server) internalError(w http.ResponseWriter, r *http.Request, err error) {
	slog.Error("request failed", "method", r.Method, "path", r.URL.Path, "error", err)
	http.Error(w, "Erreur interne, réessayez plus tard.", http.StatusInternalServerError)
}

// adminPage sets the headers shared by every admin response: private, never
// cached, never indexed, no script.
func adminPage(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("Cache-Control", "no-store")
		h.Set("X-Robots-Tag", "noindex, nofollow")
		h.Set("Content-Security-Policy", adminCSP)
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("Referrer-Policy", "strict-origin-when-cross-origin")
		next.ServeHTTP(w, r)
	})
}

// sameOrigin rejects requests sent by other sites (CSRF, NF6). Browsers set
// Sec-Fetch-Site and Origin on every form POST; the SameSite=Strict cookie is a
// second line of defence. Requests without these headers (curl) pass: they carry
// no cookie from a victim's browser.
func (s *server) sameOrigin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if r.Header.Get("Sec-Fetch-Site") == "cross-site" || (origin != "" && origin != s.cfg.PublicBaseURL) {
			http.Error(w, "Requête d'une autre origine refusée.", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// rateLimited answers 429 once the client IP has used its budget, before the
// password is even checked: the right password does not get through either.
func (s *server) rateLimited(l *ratelimit.Limiter, next http.HandlerFunc) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !l.Allow(clientIP(r, s.cfg.TrustProxy)) {
			http.Error(w, "Trop de tentatives, réessayez plus tard.", http.StatusTooManyRequests)
			return
		}
		next(w, r)
	})
}

// clientIP returns the IP of the client. Behind Traefik, the connection comes from
// the proxy, which appends the real client IP to X-Forwarded-For: the last entry is
// the one written by our proxy, earlier ones may be forged by the client.
// Copied from apps/linktree/internal/api/middleware.go.
func clientIP(r *http.Request, trustProxy bool) string {
	if trustProxy {
		if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
			parts := strings.Split(xff, ",")
			return strings.TrimSpace(parts[len(parts)-1])
		}
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
