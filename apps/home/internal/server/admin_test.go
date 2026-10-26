package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"

	"zazi.cool/home/internal/auth"
	"zazi.cool/home/internal/dbtest"
	"zazi.cool/home/internal/migrate"
	"zazi.cool/home/migrations"
)

const (
	testPassword = "correct horse battery"
	testBaseURL  = "https://zazi.cool"
)

func migrated(t *testing.T) *pgxpool.Pool {
	t.Helper()
	pool := dbtest.New(t)
	if err := migrate.Up(context.Background(), pool, migrations.FS); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return pool
}

// adminHandler returns the application over db, with password as admin password.
func adminHandler(t *testing.T, db *pgxpool.Pool, password string) http.Handler {
	t.Helper()
	// The minimum bcrypt cost keeps the tests fast.
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	return newHandler(t, Config{
		DB:            db,
		Auth:          auth.NewManager(string(hash), db, true),
		PublicBaseURL: testBaseURL,
		LoginLimit:    10,
	})
}

// postForm sends a same-origin form, as a browser on the admin page would.
func postForm(h http.Handler, path string, form url.Values, cookies ...*http.Cookie) *httptest.ResponseRecorder {
	req := httptest.NewRequest("POST", path, strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Origin", testBaseURL)
	req.Header.Set("Sec-Fetch-Site", "same-origin")
	for _, c := range cookies {
		req.AddCookie(c)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func getAdmin(h http.Handler, cookies ...*http.Cookie) *httptest.ResponseRecorder {
	req := httptest.NewRequest("GET", "/admin", nil)
	for _, c := range cookies {
		req.AddCookie(c)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func loginForm(password string) url.Values {
	return url.Values{"password": {password}}
}

// sessionCookie logs in and returns the session cookie.
func sessionCookie(t *testing.T, h http.Handler) *http.Cookie {
	t.Helper()
	rec := postForm(h, "/admin/login", loginForm(testPassword))
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("login status = %d, want 303: %s", rec.Code, rec.Body)
	}
	cookies := rec.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("login set %d cookies, want 1", len(cookies))
	}
	return cookies[0]
}

func isLoginPage(rec *httptest.ResponseRecorder) bool {
	return strings.Contains(rec.Body.String(), `action="/admin/login"`)
}

func isDashboard(rec *httptest.ResponseRecorder) bool {
	return strings.Contains(rec.Body.String(), `action="/admin/logout"`)
}

// CA6: login form without session, 401 without cookie on a bad password, the
// expected cookie and the admin page with the right one.
func TestAdminLogin(t *testing.T) {
	h := adminHandler(t, migrated(t), testPassword)

	rec := getAdmin(h)
	if rec.Code != http.StatusOK || !isLoginPage(rec) {
		t.Fatalf("GET /admin without session: status %d, login page %v", rec.Code, isLoginPage(rec))
	}

	rec = postForm(h, "/admin/login", loginForm("wrong password"))
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("bad password: status = %d, want 401", rec.Code)
	}
	if len(rec.Result().Cookies()) != 0 {
		t.Error("bad password: a cookie was set")
	}
	if !strings.Contains(rec.Body.String(), "Mot de passe incorrect") || !isLoginPage(rec) {
		t.Error("bad password: the login form should come back with an error message")
	}

	rec = postForm(h, "/admin/login", loginForm(testPassword))
	if rec.Code != http.StatusSeeOther || rec.Header().Get("Location") != "/admin" {
		t.Fatalf("good password: status %d, Location %q; want 303 to /admin", rec.Code, rec.Header().Get("Location"))
	}
	c := rec.Result().Cookies()[0]
	if c.Name != "__Host-session" || !c.HttpOnly || !c.Secure || c.SameSite != http.SameSiteStrictMode || c.Path != "/" {
		t.Errorf("unexpected cookie: %+v", c)
	}

	rec = getAdmin(h, c)
	if rec.Code != http.StatusOK || !isDashboard(rec) {
		t.Errorf("GET /admin with session: status %d, admin page %v", rec.Code, isDashboard(rec))
	}
}

func TestAdminHeaders(t *testing.T) {
	h := adminHandler(t, migrated(t), testPassword)
	for name, rec := range map[string]*httptest.ResponseRecorder{
		"GET /admin":         getAdmin(h),
		"POST /admin/login":  postForm(h, "/admin/login", loginForm("wrong password")),
		"POST /admin/logout": postForm(h, "/admin/logout", nil),
	} {
		hdr := rec.Header()
		if got := hdr.Get("Cache-Control"); got != "no-store" {
			t.Errorf("%s: Cache-Control = %q, want no-store", name, got)
		}
		if got := hdr.Get("X-Robots-Tag"); !strings.Contains(got, "noindex") {
			t.Errorf("%s: X-Robots-Tag = %q, want noindex", name, got)
		}
		if got := hdr.Get("Content-Security-Policy"); !strings.HasPrefix(got, "default-src 'none'") {
			t.Errorf("%s: Content-Security-Policy = %q", name, got)
		}
	}
	rec := getAdmin(h)
	if !strings.Contains(rec.Body.String(), `<meta name="robots" content="noindex, nofollow">`) {
		t.Error("the admin page should carry a robots noindex meta tag")
	}
	if strings.Contains(rec.Body.String(), "<script") {
		t.Error("the admin must not contain any script")
	}
}

func TestAdminLogout(t *testing.T) {
	h := adminHandler(t, migrated(t), testPassword)
	c := sessionCookie(t, h)

	rec := postForm(h, "/admin/logout", nil, c)
	if rec.Code != http.StatusSeeOther || rec.Header().Get("Location") != "/admin" {
		t.Fatalf("logout: status %d, Location %q; want 303 to /admin", rec.Code, rec.Header().Get("Location"))
	}
	if cleared := rec.Result().Cookies(); len(cleared) != 1 || cleared[0].MaxAge >= 0 {
		t.Errorf("logout should clear the cookie: %+v", cleared)
	}
	// The old cookie no longer opens the admin, even if the browser kept it.
	if rec := getAdmin(h, c); !isLoginPage(rec) {
		t.Error("the session should be closed after logout")
	}
}

// CA14: a session opened before a change of ADMIN_PASSWORD_HASH and a restart is
// refused.
func TestPasswordChangeClosesSessions(t *testing.T) {
	db := migrated(t)
	c := sessionCookie(t, adminHandler(t, db, testPassword))

	restarted := adminHandler(t, db, "another long password")
	if rec := getAdmin(restarted, c); rec.Code != http.StatusOK || !isLoginPage(rec) {
		t.Errorf("after the password change: status %d, login page %v; want the login form", rec.Code, isLoginPage(rec))
	}
}

// CA15: the 11th attempt from the same IP within the hour gets 429, even with the
// right password.
func TestLoginRateLimit(t *testing.T) {
	h := adminHandler(t, migrated(t), testPassword)
	for i := range 10 {
		if rec := postForm(h, "/admin/login", loginForm("wrong password")); rec.Code != http.StatusUnauthorized {
			t.Fatalf("attempt %d: status = %d, want 401", i+1, rec.Code)
		}
	}
	rec := postForm(h, "/admin/login", loginForm(testPassword))
	if rec.Code != http.StatusTooManyRequests {
		t.Errorf("11th attempt: status = %d, want 429", rec.Code)
	}
	if len(rec.Result().Cookies()) != 0 {
		t.Error("a rate-limited login must not open a session")
	}
}

func TestClientIP(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "10.42.0.7:51234"
	req.Header.Set("X-Forwarded-For", "203.0.113.9, 198.51.100.4")

	if got := clientIP(req, false); got != "10.42.0.7" {
		t.Errorf("without TRUST_PROXY: %q, want the connection address", got)
	}
	// Only the last entry, appended by Traefik, can be trusted.
	if got := clientIP(req, true); got != "198.51.100.4" {
		t.Errorf("with TRUST_PROXY: %q, want the last X-Forwarded-For entry", got)
	}
}

// CA16: admin POSTs from another site get 403 and change nothing.
func TestAdminCSRF(t *testing.T) {
	db := migrated(t)
	h := adminHandler(t, db, testPassword)
	c := sessionCookie(t, h)

	cases := map[string]map[string]string{
		"cross-site":       {"Sec-Fetch-Site": "cross-site", "Origin": "https://evil.example"},
		"foreign Origin":   {"Origin": "https://evil.example"},
		"sibling Origin":   {"Origin": "https://pb.zazi.cool", "Sec-Fetch-Site": "same-site"},
		"http Origin":      {"Origin": "http://zazi.cool"},
		"cross-site alone": {"Sec-Fetch-Site": "cross-site"},
	}
	for name, headers := range cases {
		for _, path := range []string{"/admin/login", "/admin/logout"} {
			req := httptest.NewRequest("POST", path, strings.NewReader(loginForm(testPassword).Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			for k, v := range headers {
				req.Header.Set(k, v)
			}
			req.AddCookie(c)
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, req)
			if rec.Code != http.StatusForbidden {
				t.Errorf("%s %s: status = %d, want 403", name, path, rec.Code)
			}
			if len(rec.Result().Cookies()) != 0 {
				t.Errorf("%s %s: a cookie was set or cleared", name, path)
			}
		}
	}

	var sessions int
	if err := db.QueryRow(context.Background(), `SELECT count(*) FROM sessions`).Scan(&sessions); err != nil {
		t.Fatal(err)
	}
	if sessions != 1 {
		t.Errorf("%d sessions in the database, want 1: refused requests must change nothing", sessions)
	}
	if rec := getAdmin(h, c); !isDashboard(rec) {
		t.Error("the session should still be open")
	}
}

func TestLoginBodyLimit(t *testing.T) {
	h := adminHandler(t, migrated(t), testPassword)
	rec := postForm(h, "/admin/login", loginForm(strings.Repeat("a", maxLoginBody)))
	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Errorf("status = %d, want 413", rec.Code)
	}
}
