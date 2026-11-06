package server

import (
	"encoding/json"
	"errors"
	"mime"
	"net"
	"net/http"
	"strings"

	"zazi.cool/habits/internal/auth"
	"zazi.cool/habits/internal/ratelimit"
)

// pageCSP is the policy of NF2: only the app's own files, no inline script or style,
// so even a name that slipped into a page unescaped could not run a script.
const pageCSP = "default-src 'none'; script-src 'self'; style-src 'self'; img-src 'self' data:; " +
	"font-src 'self'; connect-src 'self'; manifest-src 'self'; base-uri 'none'; " +
	"form-action 'self'; frame-ancestors 'none'"

type loginRequest struct {
	Password string `json:"password"`
}

func (h *handler) login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	err := h.cfg.Auth.Login(r.Context(), w, req.Password)
	if errors.Is(err, auth.ErrBadPassword) {
		writeError(w, http.StatusUnauthorized, "bad_password", "Mot de passe incorrect.")
		return
	}
	if err != nil {
		h.internalError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *handler) logout(w http.ResponseWriter, r *http.Request) {
	if err := h.cfg.Auth.Logout(r.Context(), w, r); err != nil {
		h.internalError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// session lets the front know, at startup, whether it must show the login screen.
// It is only reached with a valid session (see requireSession).
func (h *handler) session(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, struct {
		Authenticated bool `json:"authenticated"`
	}{true})
}

// requireSession answers 401 when the request has no valid session (F7).
func (h *handler) requireSession(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ok, err := h.cfg.Auth.Authenticated(r)
		if err != nil {
			h.internalError(w, r, err)
			return
		}
		if !ok {
			writeError(w, http.StatusUnauthorized, "unauthorized", "Connexion requise.")
			return
		}
		next.ServeHTTP(w, r)
	})
}

// securityHeaders sets the headers shared by every response: private, never cached,
// never indexed, strict CSP, and HSTS when the site is served over HTTPS.
func (h *handler) securityHeaders(next http.Handler) http.Handler {
	https := isHTTPS(h.cfg.BaseURL)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hdr := w.Header()
		hdr.Set("Content-Security-Policy", pageCSP)
		hdr.Set("X-Content-Type-Options", "nosniff")
		hdr.Set("X-Robots-Tag", "noindex")
		hdr.Set("Cache-Control", "no-store")
		hdr.Set("Referrer-Policy", "same-origin")
		if https {
			hdr.Set("Strict-Transport-Security", "max-age=31536000")
		}
		next.ServeHTTP(w, r)
	})
}

// sameOrigin rejects unsafe requests sent by other sites (CSRF, NF3). Browsers set
// Sec-Fetch-Site and Origin on every fetch that is not a GET; the SameSite=Strict
// cookie is a second line of defence. Requests without these headers (curl) pass: they
// carry no cookie from a victim's browser.
func (h *handler) sameOrigin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !isSafeMethod(r.Method) {
			origin := r.Header.Get("Origin")
			if r.Header.Get("Sec-Fetch-Site") == "cross-site" || (origin != "" && origin != h.cfg.BaseURL) {
				writeError(w, http.StatusForbidden, "forbidden", "Requête d'une autre origine refusée.")
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

// jsonOnly refuses a request body that is not declared as JSON (NF3). A request
// without a body, such as the logout, needs no Content-Type.
func (h *handler) jsonOnly(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !isSafeMethod(r.Method) && r.ContentLength != 0 {
			mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
			if err != nil || mediaType != "application/json" {
				writeError(w, http.StatusUnsupportedMediaType, "unsupported_media_type",
					"Le corps de la requête doit être du JSON.")
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

func isSafeMethod(method string) bool {
	return method == http.MethodGet || method == http.MethodHead || method == http.MethodOptions
}

// rateLimited answers 429 once the client IP has used its budget, before the
// password is even checked: the right password does not get through either (F7).
func (h *handler) rateLimited(l *ratelimit.Limiter, next http.HandlerFunc) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !l.Allow(clientIP(r, h.cfg.TrustProxy)) {
			writeError(w, http.StatusTooManyRequests, "too_many_attempts", "Trop de tentatives, réessayez plus tard.")
			return
		}
		next(w, r)
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

// maxBody bounds every JSON body: a few short fields and nothing else.
const maxBody = 16 << 10

// decodeJSON reads the request body into v. On failure it has already answered and
// returns false.
func decodeJSON(w http.ResponseWriter, r *http.Request, v any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, maxBody)
	if err := json.NewDecoder(r.Body).Decode(v); err != nil {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			writeError(w, http.StatusRequestEntityTooLarge, "too_large", "Requête trop volumineuse.")
			return false
		}
		writeError(w, http.StatusBadRequest, "invalid_json", "JSON invalide.")
		return false
	}
	return true
}
