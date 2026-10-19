package server

import "net/http"

// pageCSP is the policy of every response (NF1). Pasted content is rendered as text
// by html/template; the CSP is the second line of defence: no inline script or style,
// no third-party resource, no framing. script-src only allows the "copy link" script.
const pageCSP = "default-src 'none'; script-src 'self'; style-src 'self'; img-src 'self'; " +
	"form-action 'self'; frame-ancestors 'none'; base-uri 'none'"

// rawCSP applies to /raw/{id}: even if a browser rendered the text as HTML, sandbox
// would block scripts and give the page an opaque origin.
const rawCSP = "sandbox"

// securityHeaders sets the headers shared by every response. Handlers may override
// them (the raw view tightens the CSP, static files are cacheable).
func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("Content-Security-Policy", pageCSP)
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("X-Robots-Tag", "noindex, nofollow")
		// Paste URLs are the only secret protecting a paste: never leak them to
		// another site through the Referer header.
		h.Set("Referrer-Policy", "no-referrer")
		h.Set("Cache-Control", "no-store")
		next.ServeHTTP(w, r)
	})
}

// cacheable lets browsers keep static files (style sheet, script) for an hour.
func cacheable(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "public, max-age=3600")
		next.ServeHTTP(w, r)
	})
}
