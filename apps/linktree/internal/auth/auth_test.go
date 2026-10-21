package auth

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// fakeStore keeps sessions in memory.
type fakeStore struct {
	sessions map[string]fakeSession
}

type fakeSession struct {
	fingerprint []byte
	expires     time.Time
}

func newFakeStore() *fakeStore {
	return &fakeStore{sessions: map[string]fakeSession{}}
}

func (f *fakeStore) CreateSession(_ context.Context, tokenHash, fp []byte, exp time.Time) error {
	f.sessions[string(tokenHash)] = fakeSession{fp, exp}
	return nil
}

func (f *fakeStore) SessionValid(_ context.Context, tokenHash, fp []byte, now time.Time) (bool, error) {
	s, ok := f.sessions[string(tokenHash)]
	return ok && bytes.Equal(s.fingerprint, fp) && now.Before(s.expires), nil
}

func (f *fakeStore) DeleteSession(_ context.Context, tokenHash []byte) error {
	delete(f.sessions, string(tokenHash))
	return nil
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

func newManager(t *testing.T, st SessionStore, password string, secure bool) *Manager {
	t.Helper()
	m, err := NewManager(testHash(t, password), st, secure)
	if err != nil {
		t.Fatal(err)
	}
	return m
}

// login logs in and returns a request carrying the session cookie.
func login(t *testing.T, m *Manager, password string) (*http.Request, error) {
	t.Helper()
	rec := httptest.NewRecorder()
	if err := m.Login(context.Background(), rec, password); err != nil {
		return nil, err
	}
	req := httptest.NewRequest(http.MethodGet, "/admin/api/session", nil)
	for _, c := range rec.Result().Cookies() {
		req.AddCookie(c)
	}
	return req, nil
}

func TestLoginAndSession(t *testing.T) {
	st := newFakeStore()
	m := newManager(t, st, "correct horse battery", true)

	if _, err := login(t, m, "wrong password"); !errors.Is(err, ErrBadPassword) {
		t.Fatalf("err = %v, want ErrBadPassword", err)
	}
	req, err := login(t, m, "correct horse battery")
	if err != nil {
		t.Fatal(err)
	}
	if ok, err := m.Authenticated(req); err != nil || !ok {
		t.Fatalf("Authenticated = %v, %v; want true", ok, err)
	}

	anonymous := httptest.NewRequest(http.MethodGet, "/", nil)
	if ok, _ := m.Authenticated(anonymous); ok {
		t.Error("a request without cookie must not be authenticated")
	}
	forged := httptest.NewRequest(http.MethodGet, "/", nil)
	forged.AddCookie(&http.Cookie{Name: "__Host-session", Value: "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"})
	if ok, _ := m.Authenticated(forged); ok {
		t.Error("an unknown token must not be authenticated")
	}
}

func TestCookieAttributes(t *testing.T) {
	m := newManager(t, newFakeStore(), "correct horse battery", true)
	rec := httptest.NewRecorder()
	if err := m.Login(context.Background(), rec, "correct horse battery"); err != nil {
		t.Fatal(err)
	}
	c := rec.Result().Cookies()[0]
	if c.Name != "__Host-session" || !c.HttpOnly || !c.Secure || c.SameSite != http.SameSiteStrictMode || c.Path != "/" {
		t.Errorf("unexpected cookie: %+v", c)
	}

	local := newManager(t, newFakeStore(), "correct horse battery", false)
	rec = httptest.NewRecorder()
	if err := local.Login(context.Background(), rec, "correct horse battery"); err != nil {
		t.Fatal(err)
	}
	if c := rec.Result().Cookies()[0]; c.Name != "session" || c.Secure {
		t.Errorf("local cookie should be plain: %+v", c)
	}
}

func TestSessionExpires(t *testing.T) {
	m := newManager(t, newFakeStore(), "correct horse battery", true)
	req, err := login(t, m, "correct horse battery")
	if err != nil {
		t.Fatal(err)
	}
	m.now = func() time.Time { return time.Now().Add(SessionTTL + time.Minute) }
	if ok, _ := m.Authenticated(req); ok {
		t.Error("an expired session must not be authenticated")
	}
}

func TestPasswordChangeInvalidatesSessions(t *testing.T) {
	st := newFakeStore()
	m := newManager(t, st, "correct horse battery", true)
	req, err := login(t, m, "correct horse battery")
	if err != nil {
		t.Fatal(err)
	}
	// Same database, new ADMIN_PASSWORD_HASH after a restart.
	m2 := newManager(t, st, "another long password", true)
	if ok, _ := m2.Authenticated(req); ok {
		t.Error("sessions created with the old password must be invalid")
	}
}

func TestLogout(t *testing.T) {
	st := newFakeStore()
	m := newManager(t, st, "correct horse battery", true)
	req, err := login(t, m, "correct horse battery")
	if err != nil {
		t.Fatal(err)
	}
	rec := httptest.NewRecorder()
	if err := m.Logout(context.Background(), rec, req); err != nil {
		t.Fatal(err)
	}
	if ok, _ := m.Authenticated(req); ok {
		t.Error("the session must be gone after logout")
	}
	if c := rec.Result().Cookies()[0]; c.MaxAge >= 0 {
		t.Errorf("the cookie should be cleared: %+v", c)
	}
}

func TestNewManagerRejectsInvalidHash(t *testing.T) {
	if _, err := NewManager("not-a-hash", newFakeStore(), true); err == nil {
		t.Error("an invalid hash should be refused at startup")
	}
}

func TestHashPassword(t *testing.T) {
	if _, err := HashPassword("short"); err == nil {
		t.Error("short passwords should be refused")
	}
	h, err := HashPassword("correct horse battery")
	if err != nil {
		t.Fatal(err)
	}
	if bcrypt.CompareHashAndPassword([]byte(h), []byte("correct horse battery")) != nil {
		t.Error("hash does not match the password")
	}
}
