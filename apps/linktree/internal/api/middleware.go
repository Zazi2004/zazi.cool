package api

import (
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// productionCSP only allows resources served by the app itself. Titles and URLs
// come from the admin: even if one slipped into the page unescaped, no script
// could run. blob: is used by the grain texture of the Noise effect.
const productionCSP = "default-src 'self'; script-src 'self'; style-src 'self'; " +
	"img-src 'self' data: blob:; font-src 'self'; connect-src 'self'; object-src 'none'; " +
	"frame-ancestors 'none'; base-uri 'none'; form-action 'self'"

// shapeCSP applies when a shape SVG is opened directly: coolshapes files use
// inline style attributes, and nothing else is needed.
const shapeCSP = "default-src 'none'; style-src 'unsafe-inline'"

// contentSecurityPolicy returns the CSP of the HTML pages. In development, the
// front comes from the Vite dev server, which also injects <style> tags and opens
// a WebSocket for hot reload.
func contentSecurityPolicy(devURL string) string {
	if devURL == "" {
		return productionCSP
	}
	ws := devURL
	if u, err := url.Parse(devURL); err == nil {
		u.Scheme = "ws"
		ws = u.String()
	}
	return "default-src 'self' " + devURL + "; script-src 'self' " + devURL +
		"; style-src 'self' 'unsafe-inline' " + devURL + "; img-src 'self' data: blob: " + devURL +
		"; font-src 'self' " + devURL + "; connect-src 'self' " + devURL + " " + ws +
		"; object-src 'none'; frame-ancestors 'none'; base-uri 'none'; form-action 'self'"
}

func securityHeaders(csp string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("Content-Security-Policy", csp)
		h.Set("X-Content-Type-Options", "nosniff")
		// Visited sites learn that the visitor comes from link.zazi.cool, not from
		// which page: the origin is enough and reveals nothing private.
		h.Set("Referrer-Policy", "strict-origin-when-cross-origin")
		next.ServeHTTP(w, r)
	})
}

// noIndex keeps the admin out of search engines.
func noIndex(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Robots-Tag", "noindex, nofollow")
		next.ServeHTTP(w, r)
	})
}

// statusRecorder remembers the status code written by a handler, for logging.
// Embedding http.ResponseWriter forwards every other method to the original writer.
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

// clientIP returns the IP of the client. Behind Traefik, the connection comes from
// the proxy, which appends the real client IP to X-Forwarded-For: the last entry is
// the one written by our proxy, earlier ones may be forged by the client.
func clientIP(r *http.Request, trustProxy bool) string {
	if trustProxy {
		if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
			parts := strings.Split(xff, ",")
			return strings.TrimSpace(parts[len(parts)-1])
		}
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
