// Package server holds the HTTP handlers of the habit tracker: a JSON API under /api/
// and the embedded front, which handles every other URL on the client side.
package server

import (
	"context"
	"encoding/json"
	"io/fs"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"zazi.cool/habits/internal/habit"
	"zazi.cool/habits/internal/ratelimit"
)

// readyTimeout bounds the database check of /readyz so a stuck database cannot hang probes.
const readyTimeout = 2 * time.Second

// Store is what the handlers need from the persistence layer. *store.Store satisfies
// it, and tests pass a fake.
type Store interface {
	Ping(ctx context.Context) error
	CreateHabit(ctx context.Context, in habit.Input, today time.Time) (habit.Habit, error)
	ListHabits(ctx context.Context, today time.Time) (active, archived []habit.Habit, err error)
	UpdateHabit(ctx context.Context, id int64, p habit.Patch, today time.Time) (habit.Habit, error)
	ArchiveHabit(ctx context.Context, id int64, today time.Time) (habit.Habit, error)
	RestoreHabit(ctx context.Context, id int64, today time.Time) (habit.Habit, error)
	ReorderHabits(ctx context.Context, moment habit.Moment, ids []int64, today time.Time) error
	DayHabits(ctx context.Context, day time.Time) ([]habit.Logged, error)
	SetEntry(ctx context.Context, habitID int64, day time.Time, value int) error
	DayStats(ctx context.Context, from, to time.Time) ([]habit.DayStat, error)
	DayNote(ctx context.Context, day time.Time) (string, error)
	SetNote(ctx context.Context, day time.Time, text string) error
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
	// Static holds the built front (index.html and assets).
	Static fs.FS
	// Version is the git SHA of the running build, served by /version.
	Version string
	// BaseURL is the public URL without trailing slash: unsafe requests from another
	// Origin are refused, and HSTS is sent when it starts with https://.
	BaseURL string
	// TrustProxy makes the client IP come from X-Forwarded-For (behind Traefik).
	TrustProxy bool
	// LoginLimit is the number of login attempts allowed per IP and per hour.
	LoginLimit int
	// Now is the clock (time.Now when nil) and Location the time zone that defines the
	// civil day (UTC when nil); both are injected so tests control the day.
	Now      func() time.Time
	Location *time.Location
	// EditWindowDays is how many days, today included, can be edited (EDIT_WINDOW_DAYS).
	EditWindowDays int
}

type handler struct {
	cfg Config
}

// NewHandler returns the HTTP handler of the application.
func NewHandler(cfg Config) http.Handler {
	if cfg.Now == nil {
		cfg.Now = time.Now
	}
	if cfg.Location == nil {
		cfg.Location = time.UTC
	}
	h := &handler{cfg: cfg}
	loginLimiter := ratelimit.New(cfg.LoginLimit, time.Hour)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", healthz)
	mux.HandleFunc("GET /readyz", h.readyz)
	mux.HandleFunc("GET /version", h.version)

	mux.Handle("POST /api/login", h.rateLimited(loginLimiter, h.login))
	mux.HandleFunc("POST /api/logout", h.logout)
	mux.Handle("GET /api/session", h.requireSession(http.HandlerFunc(h.session)))
	mux.Handle("GET /api/habits", h.requireSession(http.HandlerFunc(h.listHabits)))
	mux.Handle("POST /api/habits", h.requireSession(http.HandlerFunc(h.createHabit)))
	mux.Handle("PUT /api/habits/order", h.requireSession(http.HandlerFunc(h.reorderHabits)))
	mux.Handle("PATCH /api/habits/{id}", h.requireSession(http.HandlerFunc(h.updateHabit)))
	mux.Handle("POST /api/habits/{id}/archive", h.requireSession(http.HandlerFunc(h.archiveHabit)))
	mux.Handle("POST /api/habits/{id}/restore", h.requireSession(http.HandlerFunc(h.restoreHabit)))
	mux.Handle("GET /api/days/{date}", h.requireSession(http.HandlerFunc(h.getDay)))
	mux.Handle("PUT /api/days/{date}/entries/{habitId}", h.requireSession(http.HandlerFunc(h.putEntry)))
	mux.Handle("PUT /api/days/{date}/note", h.requireSession(http.HandlerFunc(h.putNote)))
	// Any other /api/ URL answers JSON, never the front: 401 without a session (so an
	// unknown URL does not reveal which routes exist), 404 with one.
	mux.Handle("/api/", h.requireSession(http.HandlerFunc(apiNotFound)))

	mux.Handle("/", staticHandler(cfg.Static))

	return h.securityHeaders(h.sameOrigin(h.jsonOnly(mux)))
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

// writeJSON writes v as the JSON body of the response.
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		slog.Warn("write response", "error", err)
	}
}

type errorBody struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

// writeError answers with the error format of the spec (§7): a stable code for the
// front and a French message to show.
func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, errorBody{Error: code, Message: message})
}

func apiNotFound(w http.ResponseWriter, _ *http.Request) {
	writeError(w, http.StatusNotFound, "not_found", "Ressource introuvable.")
}

func (h *handler) internalError(w http.ResponseWriter, r *http.Request, err error) {
	slog.Error("request failed", "method", r.Method, "path", r.URL.Path, "error", err)
	writeError(w, http.StatusInternalServerError, "internal", "Erreur interne, réessayez plus tard.")
}

func isHTTPS(baseURL string) bool {
	return strings.HasPrefix(baseURL, "https://")
}

// today is the civil day now, in the configured time zone.
func (h *handler) today() time.Time {
	return habit.Today(h.cfg.Now(), h.cfg.Location)
}
