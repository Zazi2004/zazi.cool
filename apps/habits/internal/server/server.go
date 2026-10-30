// Package server holds the HTTP handlers of the habit tracker.
package server

import (
	"bytes"
	"context"
	"embed"
	"html/template"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"zazi.cool/habits/internal/ratelimit"
)

// readyTimeout bounds the database check of /readyz so a stuck database cannot hang probes.
const readyTimeout = 2 * time.Second

//go:embed templates/*.html
var templateFS embed.FS

var templates = template.Must(template.ParseFS(templateFS, "templates/*.html"))

// Store is what the handlers need from the persistence layer. *store.Store satisfies
// it, and tests pass a fake.
type Store interface {
	Ping(ctx context.Context) error
}

// Authenticator checks the password and handles sessions. *auth.Manager satisfies it.
type Authenticator interface {
	Login(ctx context.Context, w http.ResponseWriter, password string) error
	Logout(ctx context.Context, w http.ResponseWriter, r *http.Request) error
	Authenticated(r *http.Request) (bool, error)
}

// Config gathers the dependencies and settings of the handlers.
type Config struct {
	Store Store
	Auth  Authenticator
	// Version is the git SHA of the running build, served by /version.
	Version string
	// BaseURL is the public URL without trailing slash: POSTs from another Origin are
	// refused, and HSTS is sent when it starts with https://.
	BaseURL string
	// TrustProxy makes the client IP come from X-Forwarded-For (behind Traefik).
	TrustProxy bool
	// LoginLimit is the number of login attempts allowed per IP and per hour.
	LoginLimit int
}

type handler struct {
	cfg Config
}

// NewHandler returns the HTTP handler of the application.
func NewHandler(cfg Config) http.Handler {
	h := &handler{cfg: cfg}
	loginLimiter := ratelimit.New(cfg.LoginLimit, time.Hour)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", healthz)
	mux.HandleFunc("GET /readyz", h.readyz)
	mux.HandleFunc("GET /version", h.version)

	mux.HandleFunc("GET /login", h.loginForm)
	mux.Handle("POST /login", h.rateLimited(loginLimiter, h.login))
	mux.Handle("POST /logout", http.HandlerFunc(h.logout))

	// "GET /" matches every path no other pattern claims, so an unknown URL without a
	// session leads to the login page rather than revealing which pages exist.
	mux.Handle("GET /", h.requireSession(http.HandlerFunc(h.home)))

	return h.securityHeaders(h.sameOrigin(mux))
}

// healthz is the liveness probe: the process answers, nothing else is checked.
func healthz(w http.ResponseWriter, _ *http.Request) {
	writeText(w, http.StatusOK, "ok")
}

// readyz is the readiness probe: it succeeds only when the database answers.
func (h *handler) readyz(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), readyTimeout)
	defer cancel()
	if err := h.cfg.Store.Ping(ctx); err != nil {
		slog.Error("readiness check failed", "error", err)
		writeText(w, http.StatusServiceUnavailable, "database unavailable")
		return
	}
	writeText(w, http.StatusOK, "ok")
}

func (h *handler) version(w http.ResponseWriter, _ *http.Request) {
	writeText(w, http.StatusOK, h.cfg.Version)
}

func writeText(w http.ResponseWriter, status int, body string) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(status)
	if _, err := w.Write([]byte(body + "\n")); err != nil {
		slog.Warn("write response", "error", err)
	}
}

// render executes a template into a buffer first, so that a template error becomes
// a clean 500 instead of a half-written page.
func (h *handler) render(w http.ResponseWriter, r *http.Request, status int, name string, data any) {
	var buf bytes.Buffer
	if err := templates.ExecuteTemplate(&buf, name, data); err != nil {
		h.internalError(w, r, err)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	if _, err := buf.WriteTo(w); err != nil {
		slog.Warn("write response", "path", r.URL.Path, "error", err)
	}
}

func (h *handler) internalError(w http.ResponseWriter, r *http.Request, err error) {
	slog.Error("request failed", "method", r.Method, "path", r.URL.Path, "error", err)
	http.Error(w, "Erreur interne, réessayez plus tard.", http.StatusInternalServerError)
}

// home is the page behind the session; the day view replaces it with the habits ticket.
func (h *handler) home(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	h.render(w, r, http.StatusOK, "home", nil)
}

func isHTTPS(baseURL string) bool {
	return strings.HasPrefix(baseURL, "https://")
}
