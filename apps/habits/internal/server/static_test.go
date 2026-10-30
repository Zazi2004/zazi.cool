package server

import (
	"net/http"
	"strings"
	"testing"

	"zazi.cool/habits/internal/habit"
)

// The login page needs its stylesheet before any session exists, and the CSP only
// allows 'self' sources, so assets must be served without authentication.
func TestStaticAssetsArePublic(t *testing.T) {
	h := newTestHandler(10)
	tests := []struct {
		path, contentType string
	}{
		{"/static/style.css", "text/css"},
		{"/static/app.js", "text/javascript"},
	}
	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			rec := get(t, h, http.MethodGet, tt.path)
			if rec.Code != http.StatusOK {
				t.Fatalf("status = %d, want 200", rec.Code)
			}
			if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, tt.contentType) {
				t.Errorf("Content-Type = %q, want prefix %q", ct, tt.contentType)
			}
			if rec.Body.Len() == 0 {
				t.Error("empty body")
			}
		})
	}
	if rec := get(t, h, http.MethodGet, "/static/missing.css"); rec.Code != http.StatusNotFound {
		t.Errorf("missing asset status = %d, want 404", rec.Code)
	}
}

// Every page links the stylesheet and the script, and nothing inline (CSP, NF1).
func TestPagesLinkAssetsWithoutInlineCode(t *testing.T) {
	m := &memStore{items: []habit.DayItem{{HabitID: 1, Name: "Lire", Done: true}}}
	h := newHabitsHandler(t, m)
	for _, path := range []string{"/login", "/"} {
		rec := do(h, http.MethodGet, path, nil, nil)
		body := rec.Body.String()
		for _, want := range []string{`href="/static/style.css"`, `src="/static/app.js"`} {
			if !strings.Contains(body, want) {
				t.Errorf("%s: missing %s", path, want)
			}
		}
		for _, bad := range []string{" style=", " onclick=", "<style", "<script>"} {
			if strings.Contains(body, bad) {
				t.Errorf("%s: contains inline %q", path, bad)
			}
		}
	}
}

// The check boxes must stay at least 44 px tall (NF3, CA14); the CSS is the only place
// that says so, so pin it.
func TestStylesheetKeepsTouchTargets(t *testing.T) {
	css, err := staticFS.ReadFile("static/style.css")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"min-height: 44px", "width: 44px", "prefers-color-scheme: dark"} {
		if !strings.Contains(string(css), want) {
			t.Errorf("style.css lacks %q", want)
		}
	}
}
