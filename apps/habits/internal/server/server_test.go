package server

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"zazi.cool/habits/internal/habit"
)

// fakeStore answers the probes and keeps habits in memory.
type fakeStore struct {
	pingErr error
	// createErr is returned by CreateHabit; active and archived are what ListHabits returns.
	createErr        error
	active, archived []habit.Habit
	// created records the inputs CreateHabit received (a pointer: the fake is passed by value).
	created *[]habit.Input
}

func (f fakeStore) Ping(context.Context) error { return f.pingErr }

func (f fakeStore) CreateHabit(_ context.Context, in habit.Input, _ time.Time) (habit.Habit, error) {
	if f.createErr != nil {
		return habit.Habit{}, f.createErr
	}
	if f.created != nil {
		*f.created = append(*f.created, in)
	}
	return habit.Habit{
		ID: 7, Name: in.Name, Emoji: in.Emoji, Color: in.Color, Moment: in.Moment, Position: 0,
		Unit: in.Unit, Step: in.Step, Period: in.Period, Target: in.Target, Weekdays: in.Weekdays,
	}, nil
}

func (f fakeStore) ListHabits(context.Context, time.Time) ([]habit.Habit, []habit.Habit, error) {
	return f.active, f.archived, nil
}

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
