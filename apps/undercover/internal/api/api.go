// Package api exposes the HTTP API and serves the front.
package api

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"math/rand/v2"
	"net/http"
	"strconv"
	"strings"
	"time"

	"zazi.cool/undercover/internal/pairs"
	"zazi.cool/undercover/internal/ratelimit"
	"zazi.cool/undercover/internal/store"
)

// PairStore is what the handlers need from the database. *store.Store satisfies it;
// tests use an in-memory fake.
type PairStore interface {
	Ping(ctx context.Context) error
	RandomPair(ctx context.Context, f store.Filter) (pairs.Pair, error)
	Pair(ctx context.Context, id int64) (pairs.Pair, error)
	AllPairs(ctx context.Context) ([]pairs.Pair, error)
	CreatePair(ctx context.Context, c pairs.Candidate, extreme bool, source pairs.Source) (pairs.Pair, error)
	Report(ctx context.Context, pairID int64, reason pairs.Reason, reporterHash string) error
}

// Config holds the settings of the HTTP layer.
type Config struct {
	// TrustProxy reads the client IP from X-Forwarded-For (only behind Traefik).
	TrustProxy bool
	// IPHashSalt is mixed into IP hashes so that stored hashes cannot be reversed
	// by hashing every possible IP address.
	IPHashSalt string
	// Requests allowed per IP and per hour.
	SuggestionLimit, ReportLimit, CheckLimit int
	// Static holds the built front (index.html and assets).
	Static fs.FS
}

const (
	maxBodyBytes = 4 << 10 // 4 KiB is plenty for the largest request
	maxExclude   = 500     // played pairs a client may ask to exclude
	readyTimeout = 2 * time.Second
)

type server struct {
	store PairStore
	cfg   Config
}

// NewHandler returns the root handler of the application.
func NewHandler(st PairStore, cfg Config) http.Handler {
	s := &server{store: st, cfg: cfg}
	limit := func(n int, h http.HandlerFunc) http.Handler {
		return s.rateLimited(ratelimit.New(n, time.Hour), h)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", s.healthz)
	mux.HandleFunc("GET /readyz", s.readyz)
	mux.HandleFunc("GET /api/pairs/random", s.randomPair)
	mux.HandleFunc("POST /api/pairs/{id}/guess", s.guess)
	mux.Handle("GET /api/pairs/check", limit(cfg.CheckLimit, s.checkPair))
	mux.Handle("POST /api/pairs", limit(cfg.SuggestionLimit, s.createPair))
	mux.Handle("POST /api/pairs/{id}/reports", limit(cfg.ReportLimit, s.reportPair))
	// Unknown API routes must answer JSON, not the front's index.html.
	mux.HandleFunc("/api/", func(w http.ResponseWriter, r *http.Request) {
		writeError(w, http.StatusNotFound, "Route inconnue.")
	})
	// Registered without a method: "GET /" would conflict with "/api/" above.
	mux.Handle("/", staticHandler(cfg.Static))

	return logRequests(securityHeaders(mux))
}

func (s *server) healthz(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	fmt.Fprintln(w, "ok")
}

func (s *server) readyz(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), readyTimeout)
	defer cancel()
	if err := s.store.Ping(ctx); err != nil {
		slog.Warn("database not ready", "error", err)
		http.Error(w, "database unreachable", http.StatusServiceUnavailable)
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	fmt.Fprintln(w, "ok")
}

func (s *server) randomPair(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	exclude, err := parseIDs(q.Get("exclude"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "Paramètre exclude invalide.")
		return
	}
	filter := store.Filter{
		Extreme:         parseExtreme(q.Get("extreme")),
		SuggestionsOnly: q.Get("suggestionsOnly") == "true",
		Exclude:         exclude,
	}

	p, err := s.store.RandomPair(r.Context(), filter)
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, "Aucune paire ne correspond à ces réglages.")
		return
	}
	if err != nil {
		s.internalError(w, r, err)
		return
	}

	// Which word goes to the civilians is drawn at random, otherwise players would
	// learn which word of a pair is "the undercover one".
	resp := DrawResponse{ID: p.ID, CivilWord: p.WordA, UndercoverWord: p.WordB, CivilSide: SideA, Extreme: p.Extreme}
	if rand.IntN(2) == 1 {
		resp.CivilWord, resp.UndercoverWord, resp.CivilSide = p.WordB, p.WordA, SideB
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *server) guess(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	var req GuessRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if req.CivilSide != SideA && req.CivilSide != SideB {
		writeError(w, http.StatusUnprocessableEntity, `civilSide doit valoir "a" ou "b".`)
		return
	}

	p, err := s.store.Pair(r.Context(), id)
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, "Paire introuvable.")
		return
	}
	if err != nil {
		s.internalError(w, r, err)
		return
	}

	expected := p.WordA
	if req.CivilSide == SideB {
		expected = p.WordB
	}
	writeJSON(w, http.StatusOK, GuessResponse{
		Correct:  pairs.GuessMatches(req.Guess, expected),
		Expected: expected,
	})
}

func (s *server) checkPair(w http.ResponseWriter, r *http.Request) {
	c, err := pairs.NewCandidate(r.URL.Query().Get("a"), r.URL.Query().Get("b"))
	if err != nil {
		writeJSON(w, http.StatusOK, invalidResponse(err))
		return
	}
	resp, err := s.check(r.Context(), c)
	if err != nil {
		s.internalError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *server) createPair(w http.ResponseWriter, r *http.Request) {
	var req CreatePairRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if req.Website != "" {
		slog.Info("honeypot filled, suggestion rejected", "ip_hash", s.hashIP(r))
		writeError(w, http.StatusBadRequest, "Requête refusée.")
		return
	}
	if req.Extreme == nil {
		writeError(w, http.StatusUnprocessableEntity, "Choisissez la catégorie : tout public ou extrême.")
		return
	}
	c, err := pairs.NewCandidate(req.WordA, req.WordB)
	if err != nil {
		writeJSON(w, http.StatusUnprocessableEntity, invalidResponse(err))
		return
	}

	resp, err := s.check(r.Context(), c)
	if err != nil {
		s.internalError(w, r, err)
		return
	}
	if resp.Status == CheckDuplicate || (resp.Status == CheckSimilar && !req.ConfirmSimilar) {
		writeJSON(w, http.StatusConflict, resp)
		return
	}

	p, err := s.store.CreatePair(r.Context(), c, *req.Extreme, pairs.SourceSuggestion)
	if errors.Is(err, store.ErrDuplicate) {
		// Someone submitted the same pair between the check and the insert.
		writeJSON(w, http.StatusConflict, CheckResponse{Status: CheckDuplicate, Similar: []PairView{}, Related: []PairView{}})
		return
	}
	if err != nil {
		s.internalError(w, r, err)
		return
	}
	slog.Info("pair suggested", "pair_id", p.ID, "extreme", p.Extreme)
	writeJSON(w, http.StatusCreated, CreatePairResponse{Pair: viewOf(p)})
}

func (s *server) reportPair(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	var req ReportRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if !req.Reason.Valid() {
		writeError(w, http.StatusUnprocessableEntity, `reason doit valoir "extreme" ou "remove".`)
		return
	}

	err := s.store.Report(r.Context(), id, req.Reason, s.hashIP(r))
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, "Paire introuvable.")
		return
	}
	if err != nil {
		s.internalError(w, r, err)
		return
	}
	slog.Info("pair reported", "pair_id", id, "reason", req.Reason)
	w.WriteHeader(http.StatusNoContent)
}

func (s *server) check(ctx context.Context, c pairs.Candidate) (CheckResponse, error) {
	existing, err := s.store.AllPairs(ctx)
	if err != nil {
		return CheckResponse{}, fmt.Errorf("load pairs: %w", err)
	}
	report := pairs.Check(c, existing)
	resp := CheckResponse{
		Status:  CheckStatus(report.Status),
		Similar: viewsOf(report.Similar),
		Related: viewsOf(report.Related),
	}
	if report.Duplicate != nil {
		v := viewOf(*report.Duplicate)
		resp.Duplicate = &v
	}
	return resp, nil
}

func invalidResponse(err error) CheckResponse {
	// Validation messages look like "invalid pair: <French message>"; only the
	// French part is meant for users.
	msg := strings.TrimPrefix(err.Error(), pairs.ErrInvalid.Error()+": ")
	return CheckResponse{Status: CheckInvalid, Error: msg, Similar: []PairView{}, Related: []PairView{}}
}

// hashIP returns a salted hash of the client IP, so reports can be deduplicated per
// client without storing IP addresses.
func (s *server) hashIP(r *http.Request) string {
	sum := sha256.Sum256([]byte(s.cfg.IPHashSalt + "|" + clientIP(r, s.cfg.TrustProxy)))
	return hex.EncodeToString(sum[:])
}

func (s *server) rateLimited(l *ratelimit.Limiter, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !l.Allow(clientIP(r, s.cfg.TrustProxy)) {
			writeError(w, http.StatusTooManyRequests, "Trop de requêtes, réessayez plus tard.")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *server) internalError(w http.ResponseWriter, r *http.Request, err error) {
	slog.Error("request failed", "method", r.Method, "path", r.URL.Path, "error", err)
	writeError(w, http.StatusInternalServerError, "Erreur interne, réessayez plus tard.")
}

// pathID reads the {id} path parameter, answering 400 itself when it is invalid.
func pathID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		writeError(w, http.StatusBadRequest, "Identifiant de paire invalide.")
		return 0, false
	}
	return id, true
}

// parseIDs parses a comma-separated list of ids, such as "3,14,15".
func parseIDs(s string) ([]int64, error) {
	if s == "" {
		return nil, nil
	}
	parts := strings.Split(s, ",")
	if len(parts) > maxExclude {
		parts = parts[len(parts)-maxExclude:] // keep the most recent ones
	}
	ids := make([]int64, 0, len(parts))
	for _, part := range parts {
		id, err := strconv.ParseInt(part, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("parse id %q: %w", part, err)
		}
		ids = append(ids, id)
	}
	return ids, nil
}

// parseExtreme reads the "extreme" query parameter: "true" adds extreme pairs to
// the draw, "only" restricts it to them, anything else keeps the game all-ages.
func parseExtreme(s string) store.ExtremeMode {
	switch s {
	case "true":
		return store.ExtremeIncluded
	case "only":
		return store.ExtremeOnly
	default:
		return store.ExtremeExcluded
	}
}

// decodeJSON decodes a small JSON body into v, answering 400 itself on failure.
func decodeJSON(w http.ResponseWriter, r *http.Request, v any) bool {
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxBodyBytes))
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		writeError(w, http.StatusBadRequest, "Corps de requête JSON invalide.")
		return false
	}
	if _, err := dec.Token(); !errors.Is(err, io.EOF) {
		writeError(w, http.StatusBadRequest, "Corps de requête JSON invalide.")
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

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, ErrorResponse{Error: message})
}
