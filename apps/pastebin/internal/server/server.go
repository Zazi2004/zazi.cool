// Package server holds the HTTP handlers of the pastebin.
package server

import (
	"context"
	"log/slog"
	"net/http"
	"time"
)

// readyTimeout bounds the database check of /readyz so a stuck database cannot hang probes.
const readyTimeout = 2 * time.Second

// Pinger reports whether a dependency (the database) is reachable.
// *pgxpool.Pool satisfies it, and tests can pass a fake.
type Pinger interface {
	Ping(ctx context.Context) error
}

// NewHandler returns the router. version is the git SHA of the running build;
// limiter caps paste creations per client (RATE_LIMIT_CREATE).
func NewHandler(db Pinger, version string, limiter *RateLimiter) http.Handler {
	mux := http.NewServeMux()
	mux.Handle("POST /{$}", guardCreate(limiter, http.HandlerFunc(createPaste)))
	mux.HandleFunc("GET /healthz", healthz)
	mux.HandleFunc("GET /readyz", readyz(db))
	mux.HandleFunc("GET /version", versionHandler(version))
	return mux
}

// healthz is the liveness probe: the process answers, nothing else is checked.
func healthz(w http.ResponseWriter, _ *http.Request) {
	writeText(w, http.StatusOK, "ok")
}

// readyz is the readiness probe: it succeeds only when the database answers.
func readyz(db Pinger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), readyTimeout)
		defer cancel()
		if err := db.Ping(ctx); err != nil {
			slog.Error("readiness check failed", "error", err)
			writeText(w, http.StatusServiceUnavailable, "database unavailable")
			return
		}
		writeText(w, http.StatusOK, "ok")
	}
}

// createPaste is a placeholder until paste creation lands (ONBO-5), so that the
// anti-abuse guard in front of it can already be wired and tested.
func createPaste(w http.ResponseWriter, _ *http.Request) {
	writeText(w, http.StatusNotImplemented, "not implemented")
}

func versionHandler(version string) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		writeText(w, http.StatusOK, version)
	}
}

func writeText(w http.ResponseWriter, status int, body string) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	if _, err := w.Write([]byte(body + "\n")); err != nil {
		slog.Warn("write response", "error", err)
	}
}
