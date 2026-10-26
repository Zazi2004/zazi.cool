// Package server is the HTTP layer of the homepage.
package server

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"zazi.cool/home/internal/auth"
)

// Pinger is the part of the database that /readyz needs; *pgxpool.Pool satisfies it.
type Pinger interface {
	Ping(ctx context.Context) error
}

// Config holds what the HTTP layer needs.
type Config struct {
	DB Pinger
	// Version is the git SHA of the running build, served by /version.
	Version string
	// Auth checks the admin password and sessions.
	Auth *auth.Manager
	// PublicBaseURL is the origin of the site, without trailing slash: admin POSTs
	// with another Origin are refused.
	PublicBaseURL string
	// TrustProxy reads the client IP from X-Forwarded-For (behind Traefik only).
	TrustProxy bool
	// LoginLimit is the number of login attempts allowed per IP and per hour.
	LoginLimit int
}

const readyTimeout = 2 * time.Second

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
	s.adminRoutes(mux)
	return logRequests(mux)
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

// statusRecorder remembers the status code written, for the request log.
type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

func logRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rec, r)
		// Health probes run every few seconds and would drown the useful logs.
		if r.URL.Path == "/healthz" || r.URL.Path == "/readyz" {
			return
		}
		slog.Info("request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", rec.status,
			"duration_ms", time.Since(start).Milliseconds(),
		)
	})
}
