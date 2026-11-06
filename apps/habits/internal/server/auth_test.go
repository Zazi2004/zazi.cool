package server

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
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
		Store: fakeStore{}, Auth: fakeAuth{}, Static: testStatic, Version: "abc1234",
		BaseURL: testBaseURL, LoginLimit: limit,
	})
}

func postJSON(h http.Handler, path, body string, headers map[string]string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func postLogin(h http.Handler, password string, headers map[string]string) *httptest.ResponseRecorder {
	body, _ := json.Marshal(loginRequest{Password: password})
	return postJSON(h, "/api/login", string(body), headers)
}

func withSession(req *http.Request) *http.Request {
	req.AddCookie(&http.Cookie{Name: "session", Value: "ok"})
	return req
}

// errorCode returns the "error" field of a JSON error response.
func errorCode(t *testing.T, rec *httptest.ResponseRecorder) string {
	t.Helper()
	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Fatalf("Content-Type = %q, want application/json (body %q)", ct, rec.Body.String())
	}
	var body errorBody
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("body %q is not a JSON error: %v", rec.Body.String(), err)
	}
	if body.Message == "" {
		t.Error("error without a message")
	}
	return body.Error
}

// F7: without a session every /api/ URL is a JSON 401, known or not.
func TestAPIRequiresSession(t *testing.T) {
	h := newTestHandler(10)
	for _, path := range []string{"/api/session", "/api/xyz", "/api/days/2026-11-06"} {
		rec := get(t, h, http.MethodGet, path)
		if rec.Code != http.StatusUnauthorized || errorCode(t, rec) != "unauthorized" {
			t.Errorf("GET %s without session: %d %q, want 401 unauthorized", path, rec.Code, rec.Body.String())
		}
	}
}

func TestSession(t *testing.T) {
	h := newTestHandler(10)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, withSession(httptest.NewRequest(http.MethodGet, "/api/session", nil)))
	if rec.Code != http.StatusOK || strings.TrimSpace(rec.Body.String()) != `{"authenticated":true}` {
		t.Errorf("GET /api/session = %d %q", rec.Code, rec.Body.String())
	}

	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, withSession(httptest.NewRequest(http.MethodGet, "/api/xyz", nil)))
	if rec.Code != http.StatusNotFound || errorCode(t, rec) != "not_found" {
		t.Errorf("GET /api/xyz with session = %d %q, want 404 not_found", rec.Code, rec.Body.String())
	}
}

func TestLogin(t *testing.T) {
	h := newTestHandler(10)

	rec := postLogin(h, "wrong", nil)
	if rec.Code != http.StatusUnauthorized || errorCode(t, rec) != "bad_password" {
		t.Errorf("wrong password = %d %q, want 401 bad_password", rec.Code, rec.Body.String())
	}
	if len(rec.Result().Cookies()) != 0 {
		t.Error("cookie set after a failed login")
	}

	rec = postLogin(h, "right", nil)
	if rec.Code != http.StatusNoContent || rec.Body.Len() != 0 {
		t.Errorf("right password = %d %q, want 204 without body", rec.Code, rec.Body.String())
	}
	if len(rec.Result().Cookies()) != 1 {
		t.Error("no session cookie after login")
	}
}

func TestLoginInvalidBody(t *testing.T) {
	h := newTestHandler(10)
	if rec := postJSON(h, "/api/login", "{not json", nil); rec.Code != http.StatusBadRequest || errorCode(t, rec) != "invalid_json" {
		t.Errorf("invalid JSON = %d %q, want 400 invalid_json", rec.Code, rec.Body.String())
	}
	big := `{"password":"` + strings.Repeat("a", maxBody) + `"}`
	if rec := postJSON(h, "/api/login", big, nil); rec.Code != http.StatusRequestEntityTooLarge {
		t.Errorf("oversized body = %d, want 413", rec.Code)
	}
}

// NF3: the API reads JSON only.
func TestLoginRequiresJSON(t *testing.T) {
	h := newTestHandler(10)
	req := httptest.NewRequest(http.MethodPost, "/api/login", strings.NewReader("password=right"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnsupportedMediaType || errorCode(t, rec) != "unsupported_media_type" {
		t.Errorf("form body = %d %q, want 415", rec.Code, rec.Body.String())
	}
	if len(rec.Result().Cookies()) != 0 {
		t.Error("cookie set for a rejected request")
	}

	// A charset parameter is fine.
	rec = postJSON(h, "/api/login", `{"password":"right"}`, map[string]string{"Content-Type": "application/json; charset=utf-8"})
	if rec.Code != http.StatusNoContent {
		t.Errorf("json with charset = %d, want 204", rec.Code)
	}
}

func TestLoginInternalError(t *testing.T) {
	h := NewHandler(Config{Store: fakeStore{}, Auth: fakeAuth{loginErr: errors.New("db down")}, Static: testStatic, BaseURL: testBaseURL, LoginLimit: 10})
	rec := postLogin(h, "right", nil)
	if rec.Code != http.StatusInternalServerError || errorCode(t, rec) != "internal" {
		t.Errorf("status = %d %q, want 500 internal", rec.Code, rec.Body.String())
	}
}

func TestLogout(t *testing.T) {
	h := newTestHandler(10)
	rec := postJSON(h, "/api/logout", "", nil)
	if rec.Code != http.StatusNoContent {
		t.Errorf("logout = %d, want 204", rec.Code)
	}
	if cs := rec.Result().Cookies(); len(cs) != 1 || cs[0].MaxAge >= 0 {
		t.Errorf("cookie not cleared: %+v", cs)
	}
}

// F7: the 11th attempt from one IP in the hour is 429, even with the right password.
func TestLoginRateLimit(t *testing.T) {
	h := newTestHandler(10)
	for i := 1; i <= 10; i++ {
		if rec := postLogin(h, "wrong", nil); rec.Code != http.StatusUnauthorized {
			t.Fatalf("attempt %d = %d, want 401", i, rec.Code)
		}
	}
	rec := postLogin(h, "right", nil)
	if rec.Code != http.StatusTooManyRequests || errorCode(t, rec) != "too_many_attempts" {
		t.Errorf("11th attempt = %d %q, want 429", rec.Code, rec.Body.String())
	}
	if len(rec.Result().Cookies()) != 0 {
		t.Error("cookie set while rate limited")
	}
}

func TestLoginRateLimitPerIP(t *testing.T) {
	h := NewHandler(Config{Store: fakeStore{}, Auth: fakeAuth{}, Static: testStatic, BaseURL: testBaseURL, LoginLimit: 1, TrustProxy: true})
	a := map[string]string{"X-Forwarded-For": "1.1.1.1"}
	b := map[string]string{"X-Forwarded-For": "9.9.9.9, 2.2.2.2"}
	postLogin(h, "wrong", a)
	if rec := postLogin(h, "right", a); rec.Code != http.StatusTooManyRequests {
		t.Errorf("same IP = %d, want 429", rec.Code)
	}
	if rec := postLogin(h, "right", b); rec.Code != http.StatusNoContent {
		t.Errorf("other IP = %d, want 204", rec.Code)
	}
}

// NF3: cross-site unsafe requests are refused.
func TestCSRF(t *testing.T) {
	tests := []struct {
		name    string
		headers map[string]string
		want    int
	}{
		{"foreign origin", map[string]string{"Origin": "https://evil.example"}, http.StatusForbidden},
		{"sibling subdomain", map[string]string{"Origin": "https://other.habits.example"}, http.StatusForbidden},
		{"cross-site fetch", map[string]string{"Sec-Fetch-Site": "cross-site"}, http.StatusForbidden},
		{"null origin", map[string]string{"Origin": "null"}, http.StatusForbidden},
		{"same origin", map[string]string{"Origin": testBaseURL, "Sec-Fetch-Site": "same-origin"}, http.StatusNoContent},
		{"no headers", nil, http.StatusNoContent},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := postLogin(newTestHandler(10), "right", tt.headers)
			if rec.Code != tt.want {
				t.Errorf("status = %d, want %d", rec.Code, tt.want)
			}
			if tt.want == http.StatusForbidden {
				if errorCode(t, rec) != "forbidden" {
					t.Errorf("body = %q, want a forbidden error", rec.Body.String())
				}
				if len(rec.Result().Cookies()) != 0 {
					t.Error("cookie set for a refused request")
				}
			}
		})
	}
	rec := postJSON(newTestHandler(10), "/api/logout", "", map[string]string{"Origin": "https://evil.example"})
	if rec.Code != http.StatusForbidden {
		t.Errorf("cross-origin logout = %d, want 403", rec.Code)
	}
}

// NF2: the CSP of the spec, on the page as on the API.
func TestSecurityHeaders(t *testing.T) {
	const wantCSP = "default-src 'none'; script-src 'self'; style-src 'self'; img-src 'self' data:; " +
		"font-src 'self'; connect-src 'self'; manifest-src 'self'; base-uri 'none'; " +
		"form-action 'self'; frame-ancestors 'none'"
	for _, path := range []string{"/login", "/", "/healthz", "/api/session"} {
		rec := get(t, newTestHandler(10), http.MethodGet, path)
		want := map[string]string{
			"Content-Security-Policy":   wantCSP,
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
	if got := get(t, newTestHandler(10), http.MethodGet, "/api/session").Header().Get("Cache-Control"); got != "no-store" {
		t.Errorf("API Cache-Control = %q, want no-store", got)
	}
	// No HSTS over plain HTTP (local development).
	h := NewHandler(Config{Store: fakeStore{}, Auth: fakeAuth{}, Static: testStatic, BaseURL: "http://localhost:8080"})
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
