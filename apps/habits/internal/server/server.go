// Package server holds the HTTP handlers of the habit tracker.
package server

import (
	"context"
	"log/slog"
	"net/http"
	"time"
)

// readyTimeout bounds the database check of /readyz so a stuck database cannot hang probes.
const readyTimeout = 2 * time.Second

// Store is what the handlers need from the persistence layer. *store.Store satisfies
// it, and tests pass a fake.
type Store interface {
	Ping(ctx context.Context) error
}

// Config gathers the dependencies and settings of the handlers.
type Config struct {
	Store Store
	// Version is the git SHA of the running build, served by /version.
	Version string
}

type handler struct {
	cfg Config
}

// NewHandler returns the HTTP handler of the application.
func NewHandler(cfg Config) http.Handler {
	h := &handler{cfg: cfg}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", healthz)
	mux.HandleFunc("GET /readyz", h.readyz)
	mux.HandleFunc("GET /version", h.version)
	return mux
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
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("X-Robots-Tag", "noindex")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	if _, err := w.Write([]byte(body + "\n")); err != nil {
		slog.Warn("write response", "error", err)
	}
}
