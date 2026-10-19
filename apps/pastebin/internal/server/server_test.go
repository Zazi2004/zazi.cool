package server

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type fakeDB struct{ err error }

func (f fakeDB) Ping(context.Context) error { return f.err }

func get(t *testing.T, h http.Handler, method, path string) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(method, path, nil))
	return rec
}

func TestEndpoints(t *testing.T) {
	tests := []struct {
		name       string
		db         fakeDB
		path       string
		wantStatus int
		wantBody   string
	}{
		{"healthz ignores the database", fakeDB{errors.New("down")}, "/healthz", http.StatusOK, "ok"},
		{"readyz ok", fakeDB{}, "/readyz", http.StatusOK, "ok"},
		{"readyz database down", fakeDB{errors.New("down")}, "/readyz", http.StatusServiceUnavailable, "database unavailable"},
		{"version", fakeDB{}, "/version", http.StatusOK, "abc1234"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := get(t, NewHandler(tt.db, "abc1234"), http.MethodGet, tt.path)
			if rec.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
			if got := strings.TrimSpace(rec.Body.String()); got != tt.wantBody {
				t.Errorf("body = %q, want %q", got, tt.wantBody)
			}
			if ct := rec.Header().Get("Content-Type"); ct != "text/plain; charset=utf-8" {
				t.Errorf("Content-Type = %q", ct)
			}
		})
	}
}

func TestNonGetIsRejected(t *testing.T) {
	rec := get(t, NewHandler(fakeDB{}, "dev"), http.MethodPost, "/healthz")
	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("POST /healthz status = %d, want 405", rec.Code)
	}
}
