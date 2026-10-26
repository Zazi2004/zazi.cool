package server

import (
	"bytes"
	"context"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"

	"zazi.cool/home/internal/auth"
	"zazi.cool/home/internal/content"
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
		DB:             db,
		Content:        content.NewStore(db),
		Auth:           auth.NewManager(string(hash), db, true),
		PublicBaseURL:  testBaseURL,
		LoginLimit:     10,
		MaxUploadBytes: 5 << 20,
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

// editForm is a valid edit form at the given version.
func editForm(version string) url.Values {
	return url.Values{
		"version":    {version},
		"title":      {"Nouveau titre"},
		"body":       {"Un paragraphe.\r\nSur deux lignes."},
		"banner":     {"none"},
		"banner_alt": {""},
		"background": {"grain"},
	}
}

func storedTitle(t *testing.T, db *pgxpool.Pool) (title string, version int) {
	t.Helper()
	err := db.QueryRow(context.Background(), `SELECT title, version FROM content`).Scan(&title, &version)
	if err != nil {
		t.Fatal(err)
	}
	return title, version
}

// CA7: a saved title redirects to /admin with the message and shows on /.
func TestSaveContent(t *testing.T) {
	db := migrated(t)
	h := adminHandler(t, db, testPassword)
	c := sessionCookie(t, h)

	rec := getAdmin(h, c)
	if !strings.Contains(rec.Body.String(), `name="version" value="1"`) {
		t.Fatalf("the form should carry the version: %s", rec.Body)
	}

	rec = postForm(h, "/admin", editForm("1"), c)
	if rec.Code != http.StatusSeeOther || rec.Header().Get("Location") != "/admin?saved=1" {
		t.Fatalf("save: status %d, Location %q; want 303 to /admin?saved=1", rec.Code, rec.Header().Get("Location"))
	}
	if !strings.Contains(getAdminSaved(h, c), "Modifications enregistrées") {
		t.Error("the confirmation message is missing")
	}
	if title, version := storedTitle(t, db); title != "Nouveau titre" || version != 2 {
		t.Errorf("stored title %q, version %d; want the new title at version 2", title, version)
	}
	if home := get(t, h, "/"); !strings.Contains(home.Body.String(), "Nouveau titre") {
		t.Error("the home page should show the new title at once")
	}
	var body, bg string
	if err := db.QueryRow(context.Background(), `SELECT body, background FROM content`).Scan(&body, &bg); err != nil {
		t.Fatal(err)
	}
	if body != "Un paragraphe.\nSur deux lignes." || bg != "grain" {
		t.Errorf("stored body %q, background %q", body, bg)
	}
}

func getAdminSaved(h http.Handler, c *http.Cookie) string {
	req := httptest.NewRequest("GET", "/admin?saved=1", nil)
	req.AddCookie(c)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec.Body.String()
}

func TestSaveContentNeedsSession(t *testing.T) {
	db := migrated(t)
	h := adminHandler(t, db, testPassword)
	rec := postForm(h, "/admin", editForm("1"))
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", rec.Code)
	}
	if title, _ := storedTitle(t, db); title == "Nouveau titre" {
		t.Error("nothing should be saved without a session")
	}
}

// CA8: the second of two forms opened at the same version gets 409 with its input.
func TestSaveContentConflict(t *testing.T) {
	db := migrated(t)
	h := adminHandler(t, db, testPassword)
	c := sessionCookie(t, h)

	if rec := postForm(h, "/admin", editForm("1"), c); rec.Code != http.StatusSeeOther {
		t.Fatalf("first save: status %d", rec.Code)
	}
	second := editForm("1")
	second.Set("title", "Titre du second appareil")
	rec := postForm(h, "/admin", second, c)
	if rec.Code != http.StatusConflict {
		t.Fatalf("second save: status = %d, want 409", rec.Code)
	}
	for _, want := range []string{"Le contenu a été modifié ailleurs, rechargez la page", "Titre du second appareil"} {
		if !strings.Contains(rec.Body.String(), want) {
			t.Errorf("409 page should contain %q", want)
		}
	}
	if title, version := storedTitle(t, db); title != "Nouveau titre" || version != 2 {
		t.Errorf("stored title %q, version %d; the first save should remain", title, version)
	}
}

// CA9: every invalid field is reported at once, with 422, and nothing is saved.
func TestSaveContentValidation(t *testing.T) {
	db := migrated(t)
	h := adminHandler(t, db, testPassword)
	c := sessionCookie(t, h)

	cases := map[string]struct {
		field, value string
	}{
		"empty title":      {"title", ""},
		"title of 81":      {"title", strings.Repeat("é", 81)},
		"body of 601":      {"body", strings.Repeat("a", 601)},
		"control in title": {"title", "a\x07b"},
		"control in body":  {"body", "a\tb"},
		"unknown banner":   {"banner", "aube"},
		"upload no image":  {"banner", "upload"},
		"unknown fond":     {"background", "disco"},
		"alt of 151":       {"banner_alt", strings.Repeat("a", 151)},
	}
	for name, tc := range cases {
		form := editForm("1")
		form.Set(tc.field, tc.value)
		rec := postForm(h, "/admin", form, c)
		if rec.Code != http.StatusUnprocessableEntity {
			t.Errorf("%s: status = %d, want 422", name, rec.Code)
		}
		if !strings.Contains(rec.Body.String(), `role="alert"`) {
			t.Errorf("%s: no error message", name)
		}
	}
	if title, version := storedTitle(t, db); title == "Nouveau titre" || version != 1 {
		t.Errorf("stored title %q, version %d; nothing should be saved", title, version)
	}

	// Several errors come back together.
	form := editForm("1")
	form.Set("title", "")
	form.Set("background", "disco")
	rec := postForm(h, "/admin", form, c)
	if n := strings.Count(rec.Body.String(), `role="alert"`); n != 2 {
		t.Errorf("%d error messages, want 2", n)
	}
	if !strings.Contains(rec.Body.String(), "Un paragraphe.") {
		t.Error("the input should be kept")
	}
}

// Surrounding spaces are removed, lengths count characters and a browser's CRLF
// counts as one line break.
func TestSaveContentTrimsAndCounts(t *testing.T) {
	db := migrated(t)
	h := adminHandler(t, db, testPassword)
	c := sessionCookie(t, h)

	form := editForm("1")
	form.Set("title", "  "+strings.Repeat("é", 80)+" \n")
	form.Set("body", strings.Repeat("a\r\n", 300))
	if rec := postForm(h, "/admin", form, c); rec.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want 303: %s", rec.Code, rec.Body)
	}
	if title, _ := storedTitle(t, db); title != strings.Repeat("é", 80) {
		t.Errorf("stored title %q", title)
	}
}

// A browser sends the form as multipart/form-data (F10).
func TestSaveContentMultipart(t *testing.T) {
	db := migrated(t)
	h := adminHandler(t, db, testPassword)
	c := sessionCookie(t, h)

	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	for k, v := range editForm("1") {
		if err := mw.WriteField(k, v[0]); err != nil {
			t.Fatal(err)
		}
	}
	if err := mw.Close(); err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest("POST", "/admin", &buf)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	req.Header.Set("Origin", testBaseURL)
	req.AddCookie(c)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want 303: %s", rec.Code, rec.Body)
	}
	if title, _ := storedTitle(t, db); title != "Nouveau titre" {
		t.Errorf("stored title %q", title)
	}
}
