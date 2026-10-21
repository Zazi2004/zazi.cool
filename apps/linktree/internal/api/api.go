// Package api serves the public pages, the admin and its JSON API.
package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"zazi.cool/linktree/internal/auth"
	"zazi.cool/linktree/internal/pages"
	"zazi.cool/linktree/internal/ratelimit"
	"zazi.cool/linktree/internal/render"
	"zazi.cool/linktree/internal/store"
)

// Config holds what the HTTP layer needs.
type Config struct {
	Store    *store.Store
	Auth     *auth.Manager
	Renderer *render.Renderer
	// Static holds the files served as they are: the Vite build (assets/, shapes/,
	// favicon…) in production, web/public in development.
	Static fs.FS
	Shapes pages.Shapes
	// PublicBaseURL is the absolute URL of the site, without trailing slash.
	PublicBaseURL string
	// DevURL is the Vite dev server URL in development, "" in production.
	DevURL string
	// TrustProxy reads the client IP from X-Forwarded-For (only behind Traefik).
	TrustProxy bool
	// LoginLimit is the number of login attempts allowed per IP and per hour.
	LoginLimit int
	// MaxUploadBytes bounds the size of an uploaded image.
	MaxUploadBytes int64
	// Version is the git SHA of the running build, served by /version.
	Version string
}

const (
	maxSmallBody    = 16 << 10 // login, create page, reorder…
	maxDocumentBody = 1 << 20  // a full page document
	readyTimeout    = 2 * time.Second
)

type server struct {
	cfg Config
}

// NewHandler returns the root handler of the application.
func NewHandler(cfg Config) http.Handler {
	s := &server{cfg: cfg}
	mux := http.NewServeMux()

	mux.HandleFunc("GET /healthz", s.healthz)
	mux.HandleFunc("GET /readyz", s.readyz)
	mux.HandleFunc("GET /version", s.version)

	// Public pages.
	mux.HandleFunc("GET /{$}", s.index)
	mux.HandleFunc("GET /{slug}", s.page)
	mux.HandleFunc("GET /images/{id}", s.image)
	static := s.static()
	for _, p := range []string{"GET /assets/", "GET /shapes/", "GET /favicon.svg", "GET /og-default.png", "GET /robots.txt"} {
		mux.Handle(p, static)
	}

	// Admin single-page app: every /admin URL gets the same shell.
	mux.Handle("GET /admin", noIndex(http.HandlerFunc(s.adminShell)))
	mux.Handle("GET /admin/", noIndex(http.HandlerFunc(s.adminShell)))
	// One pattern per method: "/admin/api/" without method would conflict with
	// "GET /admin/" (neither would be more specific than the other).
	adminAPI := noIndex(s.adminAPI())
	for _, method := range []string{"GET", "POST", "PUT", "DELETE"} {
		mux.Handle(method+" /admin/api/", adminAPI)
	}

	return logRequests(securityHeaders(contentSecurityPolicy(cfg.DevURL), mux))
}

// adminAPI returns the handler of /admin/api/. Its own mux lets "/admin/api/" catch
// unknown routes with a JSON 404.
func (s *server) adminAPI() http.Handler {
	mux := http.NewServeMux()
	login := ratelimit.New(s.cfg.LoginLimit, time.Hour)
	mux.Handle("POST /admin/api/login", s.rateLimited(login, s.login))

	authed := func(h http.HandlerFunc) http.Handler { return s.requireAuth(h) }
	mux.Handle("POST /admin/api/logout", authed(s.logout))
	mux.Handle("GET /admin/api/session", authed(s.session))
	mux.Handle("GET /admin/api/pages", authed(s.listPages))
	mux.Handle("POST /admin/api/pages", authed(s.createPage))
	mux.Handle("PUT /admin/api/pages/order", authed(s.reorderPages))
	mux.Handle("GET /admin/api/pages/{id}", authed(s.getPage))
	mux.Handle("PUT /admin/api/pages/{id}", authed(s.savePage))
	mux.Handle("DELETE /admin/api/pages/{id}", authed(s.deletePage))
	mux.Handle("GET /admin/api/slugs/{slug}", authed(s.checkSlug))
	mux.Handle("POST /admin/api/images", authed(s.uploadImage))
	mux.HandleFunc("/admin/api/", func(w http.ResponseWriter, r *http.Request) {
		writeError(w, http.StatusNotFound, CodeNotFound, "Route inconnue.")
	})

	// Rejects state-changing requests sent by other sites (CSRF), based on the
	// Sec-Fetch-Site and Origin headers that browsers set. GET requests pass.
	csrf := http.NewCrossOriginProtection()
	csrf.SetDenyHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeError(w, http.StatusForbidden, CodeBadRequest, "Requête d'une autre origine refusée.")
	}))
	return csrf.Handler(noStore(mux))
}

// noStore keeps API responses out of every cache: they are private and change.
func noStore(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		next.ServeHTTP(w, r)
	})
}

func (s *server) healthz(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	fmt.Fprintln(w, "ok")
}

func (s *server) readyz(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), readyTimeout)
	defer cancel()
	if err := s.cfg.Store.Ping(ctx); err != nil {
		slog.Warn("database not ready", "error", err)
		http.Error(w, "database unreachable", http.StatusServiceUnavailable)
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	fmt.Fprintln(w, "ok")
}

// version answers the git SHA of the build, checked after each deployment.
func (s *server) version(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	fmt.Fprintln(w, s.cfg.Version)
}

// static serves the files of cfg.Static. Hashed Vite assets never change; shapes
// are versioned by name (a changed shape gets a new name), so both are cached for
// a year. Other files are revalidated.
func (s *server) static() http.Handler {
	files := http.FileServerFS(s.cfg.Static)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasPrefix(r.URL.Path, "/assets/"):
			w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		case strings.HasPrefix(r.URL.Path, "/shapes/"):
			w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
			w.Header().Set("Content-Security-Policy", shapeCSP)
		default:
			w.Header().Set("Cache-Control", "public, max-age=3600")
		}
		files.ServeHTTP(w, r)
	})
}

func (s *server) rateLimited(l *ratelimit.Limiter, next http.HandlerFunc) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !l.Allow(clientIP(r, s.cfg.TrustProxy)) {
			writeError(w, http.StatusTooManyRequests, CodeRateLimited, "Trop de tentatives, réessayez plus tard.")
			return
		}
		next(w, r)
	})
}

func (s *server) requireAuth(next http.HandlerFunc) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ok, err := s.cfg.Auth.Authenticated(r)
		if err != nil {
			s.internalError(w, r, err)
			return
		}
		if !ok {
			writeError(w, http.StatusUnauthorized, CodeUnauthorized, "Connexion requise.")
			return
		}
		next(w, r)
	})
}

func logError(r *http.Request, err error) {
	slog.Error("request failed", "method", r.Method, "path", r.URL.Path, "error", err)
}

func (s *server) internalError(w http.ResponseWriter, r *http.Request, err error) {
	logError(r, err)
	writeError(w, http.StatusInternalServerError, CodeInternal, "Erreur interne, réessayez plus tard.")
}

// decodeJSON reads a JSON body of at most limit bytes into v. Unknown fields and
// trailing data are refused: they reveal a client out of sync with the server.
func decodeJSON(w http.ResponseWriter, r *http.Request, limit int64, v any) bool {
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, limit))
	dec.DisallowUnknownFields()
	err := dec.Decode(v)
	if err == nil {
		if _, extra := dec.Token(); !errors.Is(extra, io.EOF) {
			err = errors.New("trailing data")
		}
	}
	if err != nil {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			writeError(w, http.StatusRequestEntityTooLarge, CodeTooLarge, "Requête trop volumineuse.")
			return false
		}
		writeError(w, http.StatusBadRequest, CodeBadRequest, "Corps de requête JSON invalide.")
		return false
	}
	return true
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		// Headers are already sent: the error can only be logged.
		slog.Error("write JSON response", "error", err)
	}
}

func writeError(w http.ResponseWriter, status int, code ErrorCode, message string) {
	writeJSON(w, status, ErrorResponse{Error: ErrorBody{Code: code, Message: message}})
}

func writeValidation(w http.ResponseWriter, fields pages.Errors) {
	writeJSON(w, http.StatusUnprocessableEntity, ErrorResponse{Error: ErrorBody{
		Code:    CodeValidation,
		Message: "Certains champs sont invalides.",
		Fields:  fields,
	}})
}
