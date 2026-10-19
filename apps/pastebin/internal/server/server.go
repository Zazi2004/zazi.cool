// Package server holds the HTTP handlers of the pastebin.
package server

import (
	"bytes"
	"context"
	"embed"
	"html/template"
	"io/fs"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"zazi.cool/pastebin/internal/highlight"
	"zazi.cool/pastebin/internal/paste"
)

// readyTimeout bounds the database check of /readyz so a stuck database cannot hang probes.
const readyTimeout = 2 * time.Second

// Store is what the handlers need from the persistence layer. *store.Store satisfies
// it, and tests pass an in-memory fake.
type Store interface {
	Ping(ctx context.Context) error
	Create(ctx context.Context, content, language string, ttl time.Duration) (paste.Paste, error)
	Get(ctx context.Context, id string) (paste.Paste, error)
}

// Config gathers the dependencies and settings of the handlers.
type Config struct {
	Store Store
	// Version is the git SHA of the running build, served by /version.
	Version string
	// MaxPasteBytes is the largest accepted paste content, in bytes.
	MaxPasteBytes int
	// BaseURL is the public URL of the service, used for the links shown to users.
	BaseURL string
	// Limiter caps paste creations per client (RATE_LIMIT_CREATE).
	Limiter *RateLimiter
}

//go:embed templates/*.html
var templateFS embed.FS

//go:embed static
var staticFS embed.FS

// templates holds every page; they share the "header" and "footer" blocks.
var templates = template.Must(template.ParseFS(templateFS, "templates/*.html"))

type handler struct {
	cfg Config
}

// NewHandler returns the router, wrapped in the security headers middleware.
func NewHandler(cfg Config) http.Handler {
	cfg.BaseURL = strings.TrimSuffix(cfg.BaseURL, "/")
	h := &handler{cfg: cfg}

	static, err := fs.Sub(staticFS, "static")
	if err != nil {
		panic(err) // the directory is embedded at build time: this cannot fail at run time
	}
	highlightCSS, err := highlight.CSS()
	if err != nil {
		panic(err) // built-in chroma styles written to memory: this cannot fail at run time
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", healthz)
	mux.HandleFunc("GET /readyz", h.readyz)
	mux.HandleFunc("GET /version", h.version)
	mux.Handle("GET /static/", cacheable(http.StripPrefix("/static/", http.FileServerFS(static))))
	mux.Handle("GET /static/highlight.css", cacheable(serveBytes("highlight.css", highlightCSS)))
	mux.HandleFunc("GET /{$}", h.form)
	mux.Handle("POST /{$}", guardCreate(cfg.Limiter, http.HandlerFunc(h.create)))
	mux.HandleFunc("GET /{id}", h.view)
	mux.HandleFunc("GET /raw/{id}", h.raw)
	return securityHeaders(mux)
}

// serveBytes serves a file generated at start-up. The name sets the Content-Type.
func serveBytes(name string, content []byte) http.Handler {
	modTime := time.Now()
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.ServeContent(w, r, name, modTime, bytes.NewReader(content))
	})
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

// render executes a page template. The page is first rendered in memory so that a
// template error can still produce a clean 500 instead of a half-written page.
func render(w http.ResponseWriter, status int, name string, data any) {
	var b strings.Builder
	if err := templates.ExecuteTemplate(&b, name, data); err != nil {
		slog.Error("render template", "template", name, "error", err)
		http.Error(w, "Erreur interne", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	if _, err := w.Write([]byte(b.String())); err != nil {
		slog.Warn("write response", "error", err)
	}
}
