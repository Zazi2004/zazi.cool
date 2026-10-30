package server

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"zazi.cool/habits/internal/auth"
)

const testBaseURL = "https://habits.example"

// fakeAuth accepts one password and one session cookie value.
type fakeAuth struct{ loginErr error }

func (f fakeAuth) Login(_ context.Context, w http.ResponseWriter, password string) error {
	if f.loginErr != nil {
		return f.loginErr
	}
	if password != "right" {
		return auth.ErrBadPassword
	}
	http.SetCookie(w, &http.Cookie{Name: "session", Value: "ok"})
	return nil
}

func (fakeAuth) Logout(_ context.Context, w http.ResponseWriter, _ *http.Request) error {
	http.SetCookie(w, &http.Cookie{Name: "session", MaxAge: -1})
	return nil
}

func (fakeAuth) Authenticated(r *http.Request) (bool, error) {
	c, err := r.Cookie("session")
	return err == nil && c.Value == "ok", nil
}

func newTestHandler(limit int) http.Handler {
	return NewHandler(Config{
		Store: fakeStore{}, Auth: fakeAuth{}, Version: "abc1234",
		BaseURL: testBaseURL, LoginLimit: limit,
	})
}

func postLogin(h http.Handler, password string, headers map[string]string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(url.Values{"password": {password}}.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestSessionRequired(t *testing.T) {
	h := newTestHandler(10)
	for _, path := range []string{"/", "/day/2026-10-30", "/anything"} {
		rec := get(t, h, http.MethodGet, path)
		if rec.Code != http.StatusSeeOther || rec.Header().Get("Location") != "/login" {
			t.Errorf("GET %s without session: %d -> %q, want 303 -> /login", path, rec.Code, rec.Header().Get("Location"))
		}
	}
	// Public routes stay reachable.
	for _, path := range []string{"/login", "/healthz", "/readyz", "/version"} {
		if rec := get(t, h, http.MethodGet, path); rec.Code != http.StatusOK {
			t.Errorf("GET %s = %d, want 200", path, rec.Code)
		}
	}
}

func TestHomeWithSession(t *testing.T) {
	h := newTestHandler(10)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: "session", Value: "ok"})
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("GET / with session = %d, want 200", rec.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/nope", nil)
	req.AddCookie(&http.Cookie{Name: "session", Value: "ok"})
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Errorf("GET /nope with session = %d, want 404", rec.Code)
	}
}

func TestLogin(t *testing.T) {
	h := newTestHandler(10)

	rec := postLogin(h, "wrong", nil)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("wrong password = %d, want 401", rec.Code)
	}
	if len(rec.Result().Cookies()) != 0 {
		t.Error("cookie set after a failed login")
	}

	rec = postLogin(h, "right", nil)
	if rec.Code != http.StatusSeeOther || rec.Header().Get("Location") != "/" {
		t.Errorf("right password = %d -> %q, want 303 -> /", rec.Code, rec.Header().Get("Location"))
	}
	if len(rec.Result().Cookies()) != 1 {
		t.Error("no session cookie after login")
	}
}

func TestLoginInternalError(t *testing.T) {
	h := NewHandler(Config{Store: fakeStore{}, Auth: fakeAuth{loginErr: errors.New("db down")}, BaseURL: testBaseURL, LoginLimit: 10})
	if rec := postLogin(h, "right", nil); rec.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500", rec.Code)
	}
}

func TestLogout(t *testing.T) {
	h := newTestHandler(10)
	req := httptest.NewRequest(http.MethodPost, "/logout", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusSeeOther || rec.Header().Get("Location") != "/login" {
		t.Errorf("logout = %d -> %q, want 303 -> /login", rec.Code, rec.Header().Get("Location"))
	}
	if cs := rec.Result().Cookies(); len(cs) != 1 || cs[0].MaxAge >= 0 {
		t.Errorf("cookie not cleared: %+v", cs)
	}
}

// CA12: the 11th attempt from one IP in the hour is 429, even with the right password.
func TestLoginRateLimit(t *testing.T) {
	h := newTestHandler(10)
	for i := 1; i <= 10; i++ {
		if rec := postLogin(h, "wrong", nil); rec.Code != http.StatusUnauthorized {
			t.Fatalf("attempt %d = %d, want 401", i, rec.Code)
		}
	}
	if rec := postLogin(h, "right", nil); rec.Code != http.StatusTooManyRequests {
		t.Errorf("11th attempt = %d, want 429", rec.Code)
	}
}

func TestLoginRateLimitPerIP(t *testing.T) {
	h := NewHandler(Config{Store: fakeStore{}, Auth: fakeAuth{}, BaseURL: testBaseURL, LoginLimit: 1, TrustProxy: true})
	a := map[string]string{"X-Forwarded-For": "1.1.1.1"}
	b := map[string]string{"X-Forwarded-For": "9.9.9.9, 2.2.2.2"}
	postLogin(h, "wrong", a)
	if rec := postLogin(h, "right", a); rec.Code != http.StatusTooManyRequests {
		t.Errorf("same IP = %d, want 429", rec.Code)
	}
	if rec := postLogin(h, "right", b); rec.Code != http.StatusSeeOther {
		t.Errorf("other IP = %d, want 303", rec.Code)
	}
}

// CA13: cross-site POSTs are refused (NF2).
func TestCSRF(t *testing.T) {
	tests := []struct {
		name    string
		headers map[string]string
		want    int
	}{
		{"foreign origin", map[string]string{"Origin": "https://evil.example"}, http.StatusForbidden},
		{"cross-site fetch", map[string]string{"Sec-Fetch-Site": "cross-site"}, http.StatusForbidden},
		{"null origin", map[string]string{"Origin": "null"}, http.StatusForbidden},
		{"same origin", map[string]string{"Origin": testBaseURL, "Sec-Fetch-Site": "same-origin"}, http.StatusSeeOther},
		{"no headers", nil, http.StatusSeeOther},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if rec := postLogin(newTestHandler(10), "right", tt.headers); rec.Code != tt.want {
				t.Errorf("status = %d, want %d", rec.Code, tt.want)
			}
		})
	}
	h := newTestHandler(10)
	req := httptest.NewRequest(http.MethodPost, "/logout", nil)
	req.Header.Set("Origin", "https://evil.example")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Errorf("cross-origin logout = %d, want 403", rec.Code)
	}
}

func TestSecurityHeaders(t *testing.T) {
	for _, path := range []string{"/login", "/", "/healthz"} {
		rec := get(t, newTestHandler(10), http.MethodGet, path)
		want := map[string]string{
			"Content-Security-Policy":   pageCSP,
			"X-Content-Type-Options":    "nosniff",
			"X-Robots-Tag":              "noindex",
			"Strict-Transport-Security": "max-age=31536000",
		}
		for k, v := range want {
			if got := rec.Header().Get(k); got != v {
				t.Errorf("%s %s = %q, want %q", path, k, got, v)
			}
		}
	}
	// No HSTS over plain HTTP (local development).
	h := NewHandler(Config{Store: fakeStore{}, Auth: fakeAuth{}, BaseURL: "http://localhost:8080"})
	if got := get(t, h, http.MethodGet, "/login").Header().Get("Strict-Transport-Security"); got != "" {
		t.Errorf("HSTS over http = %q, want none", got)
	}
}

func TestClientIP(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.RemoteAddr = "10.0.0.1:5555"
	r.Header.Set("X-Forwarded-For", "6.6.6.6, 3.3.3.3")
	if got := clientIP(r, false); got != "10.0.0.1" {
		t.Errorf("untrusted proxy: %q, want 10.0.0.1", got)
	}
	if got := clientIP(r, true); got != "3.3.3.3" {
		t.Errorf("trusted proxy: %q, want 3.3.3.3", got)
	}
}
