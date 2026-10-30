package auth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"

	"zazi.cool/habits/internal/dbtest"
	"zazi.cool/habits/internal/migrate"
	"zazi.cool/habits/internal/store"
	"zazi.cool/habits/migrations"
)

const password = "correct horse battery"

func newManager(t *testing.T, hashOf string, secure bool, pool *pgxpool.Pool) *Manager {
	t.Helper()
	// The minimum cost keeps the tests fast.
	h, err := bcrypt.GenerateFromPassword([]byte(hashOf), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	return NewManager(string(h), store.New(pool), secure)
}

func migrated(t *testing.T) *pgxpool.Pool {
	t.Helper()
	pool := dbtest.New(t)
	if err := migrate.Up(context.Background(), pool, migrations.FS); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return pool
}

// loginRequest logs in and returns the cookies set, plus a request carrying them.
func loginRequest(t *testing.T, m *Manager) ([]*http.Cookie, *http.Request) {
	t.Helper()
	rec := httptest.NewRecorder()
	if err := m.Login(context.Background(), rec, password); err != nil {
		t.Fatalf("Login: %v", err)
	}
	cookies := rec.Result().Cookies()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	for _, c := range cookies {
		req.AddCookie(c)
	}
	return cookies, req
}

func authenticated(t *testing.T, m *Manager, r *http.Request) bool {
	t.Helper()
	ok, err := m.Authenticated(r)
	if err != nil {
		t.Fatalf("Authenticated: %v", err)
	}
	return ok
}

func TestLoginCookieAttributes(t *testing.T) {
	tests := []struct {
		name       string
		secure     bool
		wantName   string
		wantSecure bool
	}{
		{"https", true, "__Host-session", true},
		{"http", false, "session", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := newManager(t, password, tt.secure, migrated(t))
			cookies, req := loginRequest(t, m)
			if len(cookies) != 1 {
				t.Fatalf("got %d cookies, want 1", len(cookies))
			}
			c := cookies[0]
			if c.Name != tt.wantName || c.Secure != tt.wantSecure || !c.HttpOnly ||
				c.SameSite != http.SameSiteStrictMode || c.Path != "/" || c.Domain != "" {
				t.Errorf("cookie = %+v", c)
			}
			if got := time.Until(c.Expires); got < SessionTTL-time.Minute || got > SessionTTL {
				t.Errorf("cookie lifetime = %v, want about %v", got, SessionTTL)
			}
			if !authenticated(t, m, req) {
				t.Error("fresh session not authenticated")
			}
		})
	}
}

func TestLoginBadPassword(t *testing.T) {
	m := newManager(t, password, false, migrated(t))
	rec := httptest.NewRecorder()
	if err := m.Login(context.Background(), rec, "wrong"); err != ErrBadPassword {
		t.Fatalf("Login err = %v, want ErrBadPassword", err)
	}
	if len(rec.Result().Cookies()) != 0 {
		t.Error("a cookie was set after a failed login")
	}
}

func TestAuthenticatedRejects(t *testing.T) {
	pool := migrated(t)
	m := newManager(t, password, false, pool)
	_, req := loginRequest(t, m)

	t.Run("no cookie", func(t *testing.T) {
		if authenticated(t, m, httptest.NewRequest(http.MethodGet, "/", nil)) {
			t.Error("authenticated without cookie")
		}
	})
	t.Run("forged token", func(t *testing.T) {
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		r.AddCookie(&http.Cookie{Name: "session", Value: "AAAA"})
		if authenticated(t, m, r) {
			t.Error("authenticated with a forged cookie")
		}
	})
	t.Run("password changed", func(t *testing.T) {
		other := newManager(t, "another password!", false, pool)
		if authenticated(t, other, req) {
			t.Error("session survived a password change")
		}
	})
	t.Run("expired", func(t *testing.T) {
		later := newManager(t, password, false, pool)
		later.fingerprint = m.fingerprint
		later.now = func() time.Time { return time.Now().Add(SessionTTL + time.Hour) }
		if authenticated(t, later, req) {
			t.Error("expired session accepted")
		}
	})
}

func TestLogout(t *testing.T) {
	m := newManager(t, password, false, migrated(t))
	_, req := loginRequest(t, m)

	rec := httptest.NewRecorder()
	if err := m.Logout(context.Background(), rec, req); err != nil {
		t.Fatalf("Logout: %v", err)
	}
	if cs := rec.Result().Cookies(); len(cs) != 1 || cs[0].MaxAge >= 0 {
		t.Errorf("cookie not cleared: %+v", cs)
	}
	if authenticated(t, m, req) {
		t.Error("session still valid after logout")
	}
}

func TestDeleteExpired(t *testing.T) {
	m := newManager(t, password, false, migrated(t))
	_, req := loginRequest(t, m)
	ctx := context.Background()

	if n, err := m.DeleteExpired(ctx, time.Now()); err != nil || n != 0 {
		t.Fatalf("DeleteExpired now = %d, %v; want 0, nil", n, err)
	}
	if !authenticated(t, m, req) {
		t.Fatal("live session was deleted")
	}
	if n, err := m.DeleteExpired(ctx, time.Now().Add(SessionTTL+time.Hour)); err != nil || n != 1 {
		t.Fatalf("DeleteExpired later = %d, %v; want 1, nil", n, err)
	}
}

func TestHashPasswordAndCheckHash(t *testing.T) {
	if _, err := HashPassword("short"); err == nil {
		t.Error("short password accepted")
	}
	if err := CheckHash("not a hash"); err == nil {
		t.Error("garbage accepted as hash")
	}
	h, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	if err := CheckHash(string(h)); err != nil {
		t.Errorf("CheckHash(valid) = %v", err)
	}
}
