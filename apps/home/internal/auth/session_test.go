package auth

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"

	"zazi.cool/home/internal/dbtest"
	"zazi.cool/home/internal/migrate"
	"zazi.cool/home/migrations"
)

const password = "correct horse battery"

func migrated(t *testing.T) *pgxpool.Pool {
	t.Helper()
	pool := dbtest.New(t)
	if err := migrate.Up(context.Background(), pool, migrations.FS); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return pool
}

// testHash uses the minimum cost so that tests stay fast.
func testHash(t *testing.T, password string) string {
	t.Helper()
	h, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	return string(h)
}

// login logs in and returns the recorder and a request carrying the session cookie.
func login(t *testing.T, m *Manager, password string) (*httptest.ResponseRecorder, *http.Request, error) {
	t.Helper()
	rec := httptest.NewRecorder()
	if err := m.Login(context.Background(), rec, password); err != nil {
		return rec, nil, err
	}
	req := httptest.NewRequest(http.MethodGet, "/admin", nil)
	for _, c := range rec.Result().Cookies() {
		req.AddCookie(c)
	}
	return rec, req, nil
}

func authenticated(t *testing.T, m *Manager, r *http.Request) bool {
	t.Helper()
	ok, err := m.Authenticated(r)
	if err != nil {
		t.Fatalf("Authenticated: %v", err)
	}
	return ok
}

func TestLoginAndSession(t *testing.T) {
	db := migrated(t)
	m := NewManager(testHash(t, password), db, true)

	rec, _, err := login(t, m, "wrong password")
	if !errors.Is(err, ErrBadPassword) {
		t.Fatalf("err = %v, want ErrBadPassword", err)
	}
	if len(rec.Result().Cookies()) != 0 {
		t.Error("a bad password must not set a cookie")
	}

	_, req, err := login(t, m, password)
	if err != nil {
		t.Fatal(err)
	}
	if !authenticated(t, m, req) {
		t.Fatal("the session should be valid")
	}

	if authenticated(t, m, httptest.NewRequest(http.MethodGet, "/admin", nil)) {
		t.Error("a request without cookie must not be authenticated")
	}
	forged := httptest.NewRequest(http.MethodGet, "/admin", nil)
	forged.AddCookie(&http.Cookie{Name: "__Host-session", Value: "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"})
	if authenticated(t, m, forged) {
		t.Error("an unknown token must not be authenticated")
	}
}

func TestOnlyTheTokenHashIsStored(t *testing.T) {
	db := migrated(t)
	m := NewManager(testHash(t, password), db, true)
	_, req, err := login(t, m, password)
	if err != nil {
		t.Fatal(err)
	}
	token, ok := m.token(req)
	if !ok {
		t.Fatal("cookie does not hold a token")
	}

	var stored []byte
	var ttl time.Duration
	err = db.QueryRow(context.Background(),
		`SELECT token_hash, expires_at - created_at FROM sessions`).Scan(&stored, &ttl)
	if err != nil {
		t.Fatal(err)
	}
	if string(stored) != string(hashToken(token)) {
		t.Error("the database should hold the SHA-256 of the token, not the token")
	}
	if ttl != SessionTTL {
		t.Errorf("session lifetime = %v, want %v", ttl, SessionTTL)
	}
}

func TestCookieAttributes(t *testing.T) {
	db := migrated(t)
	rec, _, err := login(t, NewManager(testHash(t, password), db, true), password)
	if err != nil {
		t.Fatal(err)
	}
	c := rec.Result().Cookies()[0]
	if c.Name != "__Host-session" || !c.HttpOnly || !c.Secure || c.SameSite != http.SameSiteStrictMode || c.Path != "/" || c.Domain != "" {
		t.Errorf("unexpected HTTPS cookie: %+v", c)
	}

	rec, _, err = login(t, NewManager(testHash(t, password), db, false), password)
	if err != nil {
		t.Fatal(err)
	}
	if c := rec.Result().Cookies()[0]; c.Name != "session" || c.Secure || !c.HttpOnly || c.SameSite != http.SameSiteStrictMode {
		t.Errorf("unexpected HTTP cookie: %+v", c)
	}
}

func TestSessionExpires(t *testing.T) {
	m := NewManager(testHash(t, password), migrated(t), true)
	_, req, err := login(t, m, password)
	if err != nil {
		t.Fatal(err)
	}
	m.now = func() time.Time { return time.Now().Add(SessionTTL + time.Minute) }
	if authenticated(t, m, req) {
		t.Error("an expired session must not be authenticated")
	}
}

// CA14: a new ADMIN_PASSWORD_HASH after a restart refuses the old sessions.
func TestPasswordChangeInvalidatesSessions(t *testing.T) {
	db := migrated(t)
	_, req, err := login(t, NewManager(testHash(t, password), db, true), password)
	if err != nil {
		t.Fatal(err)
	}
	restarted := NewManager(testHash(t, "another long password"), db, true)
	if authenticated(t, restarted, req) {
		t.Error("sessions created with the old password must be invalid")
	}
}

func TestLogout(t *testing.T) {
	m := NewManager(testHash(t, password), migrated(t), true)
	_, req, err := login(t, m, password)
	if err != nil {
		t.Fatal(err)
	}
	rec := httptest.NewRecorder()
	if err := m.Logout(context.Background(), rec, req); err != nil {
		t.Fatal(err)
	}
	if authenticated(t, m, req) {
		t.Error("the session must be gone after logout")
	}
	if c := rec.Result().Cookies()[0]; c.MaxAge >= 0 {
		t.Errorf("the cookie should be cleared: %+v", c)
	}
}

func TestDeleteExpired(t *testing.T) {
	db := migrated(t)
	m := NewManager(testHash(t, password), db, true)
	_, current, err := login(t, m, password)
	if err != nil {
		t.Fatal(err)
	}
	m.now = func() time.Time { return time.Now().Add(-SessionTTL - time.Hour) }
	if _, _, err := login(t, m, password); err != nil {
		t.Fatal(err)
	}
	m.now = time.Now

	n, err := m.DeleteExpired(context.Background(), time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Errorf("deleted %d sessions, want 1", n)
	}
	if !authenticated(t, m, current) {
		t.Error("the current session must be kept")
	}
	// The purge is idempotent.
	if n, err := m.DeleteExpired(context.Background(), time.Now()); err != nil || n != 0 {
		t.Errorf("second purge = %d, %v; want 0, nil", n, err)
	}
}
