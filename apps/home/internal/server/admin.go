package server

import (
	"bytes"
	"embed"
	"errors"
	"html/template"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"time"

	"zazi.cool/home/internal/auth"
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

// loginPage is the data of the "login" template.
type loginPage struct {
	Error string
}

// adminRoutes registers the admin pages on mux.
func (s *server) adminRoutes(mux *http.ServeMux) {
	login := ratelimit.New(s.cfg.LoginLimit, time.Hour)
	mux.Handle("GET /admin", adminPage(http.HandlerFunc(s.admin)))
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
	s.render(w, r, http.StatusOK, "dashboard", nil)
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
