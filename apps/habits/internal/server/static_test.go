package server

import (
	"net/http"
	"strings"
	"testing"
)

// NF1: every URL outside /api/, /assets/ and the probes is a client-side route.
func TestClientRoutesServeIndex(t *testing.T) {
	h := newTestHandler(10)
	for _, path := range []string{"/", "/habits", "/stats", "/stats/3", "/day/2026-11-06", "/login", "/nope/deep"} {
		rec := get(t, h, http.MethodGet, path)
		if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "<title>app</title>") {
			t.Errorf("GET %s = %d %q, want 200 with index.html", path, rec.Code, rec.Body.String())
		}
		if got := rec.Header().Get("Cache-Control"); got != "no-cache" {
			t.Errorf("GET %s Cache-Control = %q, want no-cache", path, got)
		}
	}
}

func TestStaticFiles(t *testing.T) {
	h := newTestHandler(10)

	rec := get(t, h, http.MethodGet, "/assets/index-a.js")
	if rec.Code != http.StatusOK || rec.Body.String() != "console.log(1)" {
		t.Errorf("asset = %d %q", rec.Code, rec.Body.String())
	}
	if got := rec.Header().Get("Cache-Control"); !strings.Contains(got, "immutable") {
		t.Errorf("asset Cache-Control = %q, want immutable", got)
	}

	if rec := get(t, h, http.MethodGet, "/favicon.svg"); rec.Body.String() != "<svg/>" {
		t.Errorf("favicon body = %q", rec.Body.String())
	}

	// A missing asset must not become the HTML page: a <script> would receive HTML.
	if rec := get(t, h, http.MethodGet, "/assets/missing.js"); rec.Code != http.StatusNotFound {
		t.Errorf("missing asset = %d, want 404", rec.Code)
	}
}

func TestStaticRejectsUnsafeMethods(t *testing.T) {
	rec := get(t, newTestHandler(10), http.MethodPost, "/habits")
	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("POST /habits = %d, want 405", rec.Code)
	}
}
