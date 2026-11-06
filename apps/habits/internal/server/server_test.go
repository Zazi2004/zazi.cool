package server

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"
)

// fakeStore answers the probes, the only thing the handlers ask of the store for now.
type fakeStore struct{ pingErr error }

func (f fakeStore) Ping(context.Context) error { return f.pingErr }

var testStatic = fstest.MapFS{
	"index.html":        {Data: []byte("<!doctype html><title>app</title>")},
	"favicon.svg":       {Data: []byte("<svg/>")},
	"assets/index-a.js": {Data: []byte("console.log(1)")},
}

func get(t *testing.T, h http.Handler, method, path string) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(method, path, nil))
	return rec
}

func TestProbesAndVersion(t *testing.T) {
	tests := []struct {
		name     string
		pingErr  error
		path     string
		wantCode int
		wantBody string
	}{
		{"healthz", nil, "/healthz", http.StatusOK, "ok"},
		{"healthz ignores the database", errors.New("down"), "/healthz", http.StatusOK, "ok"},
		{"readyz ready", nil, "/readyz", http.StatusOK, "ok"},
		{"readyz database down", errors.New("down"), "/readyz", http.StatusServiceUnavailable, "database unavailable"},
		{"version", nil, "/version", http.StatusOK, "abc1234"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewHandler(Config{Store: fakeStore{pingErr: tt.pingErr}, Static: testStatic, Version: "abc1234"})
			rec := get(t, h, http.MethodGet, tt.path)
			if rec.Code != tt.wantCode {
				t.Errorf("status = %d, want %d", rec.Code, tt.wantCode)
			}
			if got := strings.TrimSpace(rec.Body.String()); got != tt.wantBody {
				t.Errorf("body = %q, want %q", got, tt.wantBody)
			}
			if got := rec.Header().Get("X-Robots-Tag"); got != "noindex" {
				t.Errorf("X-Robots-Tag = %q, want noindex", got)
			}
		})
	}
}

func TestOnlyGetIsAllowedOnProbes(t *testing.T) {
	h := NewHandler(Config{Store: fakeStore{}, Static: testStatic, Version: "abc1234"})
	if rec := get(t, h, http.MethodPost, "/healthz"); rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("POST /healthz = %d, want 405", rec.Code)
	}
}
