package server

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"zazi.cool/home/internal/dbtest"
)

// fakeDB lets the tests choose whether the database answers.
type fakeDB struct{ err error }

func (f fakeDB) Ping(context.Context) error { return f.err }

func get(t *testing.T, h http.Handler, path string) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("GET", path, nil))
	return rec
}

func TestHealthz(t *testing.T) {
	// The liveness probe does not depend on the database.
	h := NewHandler(Config{DB: fakeDB{err: errors.New("down")}})
	if rec := get(t, h, "/healthz"); rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rec.Code)
	}
}

func TestReadyzWithoutDatabase(t *testing.T) {
	h := NewHandler(Config{DB: fakeDB{err: errors.New("connection refused")}})
	if rec := get(t, h, "/readyz"); rec.Code != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want 503", rec.Code)
	}
}

func TestReadyzWithDatabase(t *testing.T) {
	h := NewHandler(Config{DB: dbtest.New(t)})
	if rec := get(t, h, "/readyz"); rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200: %s", rec.Code, rec.Body)
	}
}

func TestVersion(t *testing.T) {
	h := NewHandler(Config{DB: fakeDB{}, Version: "abc1234"})
	rec := get(t, h, "/version")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if got := strings.TrimSpace(rec.Body.String()); got != "abc1234" {
		t.Errorf("version = %q, want abc1234", got)
	}
	if got := rec.Header().Get("Cache-Control"); got != "no-store" {
		t.Errorf("Cache-Control = %q, want no-store", got)
	}
}
