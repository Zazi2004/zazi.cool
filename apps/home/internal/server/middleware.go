package server

import (
	"log/slog"
	"net/http"
	"time"
)

// pageCSP is the CSP of NF5: only the app's own files, no inline script or style.
// Even if some admin input slipped into the page unescaped, no script could run.
const pageCSP = "default-src 'none'; script-src 'self'; style-src 'self'; img-src 'self'; " +
	"font-src 'self'; form-action 'self'; base-uri 'none'; frame-ancestors 'none'"

// Request body limits (NF8): small everywhere, except the admin form, which
// carries the banner upload.
const (
	maxBody          = 16 << 10
	adminFormOverage = 64 << 10
)

func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("Content-Security-Policy", pageCSP)
		h.Set("X-Content-Type-Options", "nosniff")
		// The services learn that the visitor comes from zazi.cool, nothing more.
		h.Set("Referrer-Policy", "strict-origin-when-cross-origin")
		next.ServeHTTP(w, r)
	})
}

// limitBody makes reading more than the allowed body fail (the handler then
// answers 413), so a client cannot make the server buffer an endless request.
func limitBody(maxUpload int64, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		limit := int64(maxBody)
		if r.Method == http.MethodPost && r.URL.Path == "/admin" {
			limit = maxUpload + adminFormOverage
		}
		r.Body = http.MaxBytesReader(w, r.Body, limit)
		next.ServeHTTP(w, r)
	})
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
