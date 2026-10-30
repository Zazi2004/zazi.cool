package server

import (
	"errors"
	"net"
	"net/http"
	"strings"

	"zazi.cool/habits/internal/auth"
	"zazi.cool/habits/internal/ratelimit"
)

// maxLoginBody bounds the login form: a password field and little else.
const maxLoginBody = 16 << 10

// pageCSP is the policy of NF1: only the app's own files, no inline script or style,
// so even a name that slipped into a page unescaped could not run a script.
const pageCSP = "default-src 'none'; script-src 'self'; style-src 'self'; connect-src 'self'; " +
	"form-action 'self'; base-uri 'none'; frame-ancestors 'none'"

type loginPage struct{ Error string }

func (h *handler) loginForm(w http.ResponseWriter, r *http.Request) {
	h.render(w, r, http.StatusOK, "login", loginPage{})
}

func (h *handler) login(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxLoginBody)
	if err := r.ParseForm(); err != nil {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			http.Error(w, "Requête trop volumineuse.", http.StatusRequestEntityTooLarge)
			return
		}
		http.Error(w, "Formulaire invalide.", http.StatusBadRequest)
		return
	}
	err := h.cfg.Auth.Login(r.Context(), w, r.PostForm.Get("password"))
	if errors.Is(err, auth.ErrBadPassword) {
		h.render(w, r, http.StatusUnauthorized, "login", loginPage{Error: "Mot de passe incorrect."})
		return
	}
	if err != nil {
		h.internalError(w, r, err)
		return
	}
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (h *handler) logout(w http.ResponseWriter, r *http.Request) {
	if err := h.cfg.Auth.Logout(r.Context(), w, r); err != nil {
		h.internalError(w, r, err)
		return
	}
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

// requireSession redirects to the login page when the request has no valid session (F12).
func (h *handler) requireSession(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ok, err := h.cfg.Auth.Authenticated(r)
		if err != nil {
			h.internalError(w, r, err)
			return
		}
		if !ok {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
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

// sameOrigin rejects POSTs sent by other sites (CSRF, NF2). Browsers set
// Sec-Fetch-Site and Origin on every form POST; the SameSite=Strict cookie is a
// second line of defence. Requests without these headers (curl) pass: they carry
// no cookie from a victim's browser.
func (h *handler) sameOrigin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			origin := r.Header.Get("Origin")
			if r.Header.Get("Sec-Fetch-Site") == "cross-site" || (origin != "" && origin != h.cfg.BaseURL) {
				http.Error(w, "Requête d'une autre origine refusée.", http.StatusForbidden)
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

// rateLimited answers 429 once the client IP has used its budget, before the
// password is even checked: the right password does not get through either (F14).
func (h *handler) rateLimited(l *ratelimit.Limiter, next http.HandlerFunc) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !l.Allow(clientIP(r, h.cfg.TrustProxy)) {
			http.Error(w, "Trop de tentatives, réessayez plus tard.", http.StatusTooManyRequests)
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
