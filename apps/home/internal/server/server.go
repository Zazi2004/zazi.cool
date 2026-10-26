// Package server is the HTTP layer of the homepage.
package server

import (
	"context"
	"fmt"
	"html/template"
	"log/slog"
	"net/http"
	"time"

	"zazi.cool/home/internal/auth"
	"zazi.cool/home/internal/web"
)

// Pinger is the part of the database that /readyz needs; *pgxpool.Pool satisfies it.
type Pinger interface {
	Ping(ctx context.Context) error
}

// Config holds what the HTTP layer needs.
type Config struct {
	DB      Pinger
	Content ContentStore
	// Version is the git SHA of the running build, served by /version.
	Version string
	// Auth checks the admin password and sessions.
	Auth *auth.Manager
	// PublicBaseURL is the public URL of the site, without trailing slash: Open
	// Graph URLs, and admin POSTs with another Origin are refused.
	PublicBaseURL string
	// TrustProxy reads the client IP from X-Forwarded-For (behind Traefik only).
	TrustProxy bool
	// LoginLimit is the number of login attempts allowed per IP and per hour.
	LoginLimit int
	// MaxUploadBytes bounds the banner upload of the admin form (MAX_UPLOAD_MB).
	MaxUploadBytes int64
}

const readyTimeout = 2 * time.Second

type server struct {
	cfg    Config
	assets *web.Assets
	tmpl   *template.Template
}

// NewHandler returns the root handler of the application.
func NewHandler(cfg Config) (http.Handler, error) {
	assets, err := web.LoadAssets()
	if err != nil {
		return nil, err
	}
	tmpl, err := web.Templates(assets)
	if err != nil {
		return nil, err
	}
	s := &server{cfg: cfg, assets: assets, tmpl: tmpl}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", s.healthz)
	mux.HandleFunc("GET /readyz", s.readyz)
	mux.HandleFunc("GET /version", s.version)
	// "/{$}" matches "/" only; "/" alone would match every path.
	mux.HandleFunc("GET /{$}", s.home)
	mux.HandleFunc("GET /assets/{file}", s.asset)
	mux.HandleFunc("GET /assets/banners/{file}", s.banner)
	mux.HandleFunc("GET /images/{id}", s.image)
	for _, name := range []string{"favicon.svg", "og-default.png", "robots.txt"} {
		mux.HandleFunc("GET /"+name, s.rootFile(name))
	}
	s.adminRoutes(mux)
	mux.HandleFunc("/", s.notFound)
	return logRequests(securityHeaders(limitBody(cfg.MaxUploadBytes, mux))), nil
}

// healthz only proves that the process answers: it never checks the database, so
// that a database outage does not get the pod restarted.
func (s *server) healthz(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	fmt.Fprintln(w, "ok")
}

func (s *server) readyz(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), readyTimeout)
	defer cancel()
	if err := s.cfg.DB.Ping(ctx); err != nil {
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
