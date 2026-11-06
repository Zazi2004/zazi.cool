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
	// logged is what DayHabits returns; entryErr is returned by SetEntry, which records
	// its calls in entries.
	logged   []habit.Logged
	entryErr error
	entries  *[]entryCall
	// stats is what DayStats returns, note what DayNote returns; SetNote records its
	// calls in notes.
	stats []habit.DayStat
	note  string
	notes *[]noteCall
	// editErr is returned by UpdateHabit, ArchiveHabit, RestoreHabit and ReorderHabits;
	// patches and orders record the requests they received.
	editErr error
	patches *[]habit.Patch
	orders  *[]orderCall
}

type orderCall struct {
	moment habit.Moment
	ids    []int64
}

type noteCall struct {
	day  time.Time
	text string
}

type entryCall struct {
	habitID int64
	day     time.Time
	value   int
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

func (f fakeStore) UpdateHabit(_ context.Context, id int64, p habit.Patch, _ time.Time) (habit.Habit, error) {
	if f.patches != nil {
		*f.patches = append(*f.patches, p)
	}
	return habit.Habit{ID: id, Name: "Course", Period: habit.PeriodWeek, Target: 4}, f.editErr
}

func (f fakeStore) ArchiveHabit(_ context.Context, id int64, today time.Time) (habit.Habit, error) {
	return habit.Habit{ID: id, Name: "Course", ArchivedOn: &today}, f.editErr
}

func (f fakeStore) RestoreHabit(_ context.Context, id int64, _ time.Time) (habit.Habit, error) {
	return habit.Habit{ID: id, Name: "Course"}, f.editErr
}

func (f fakeStore) ReorderHabits(_ context.Context, moment habit.Moment, ids []int64, _ time.Time) error {
	if f.orders != nil {
		*f.orders = append(*f.orders, orderCall{moment, ids})
	}
	return f.editErr
}

func (f fakeStore) DayHabits(context.Context, time.Time) ([]habit.Logged, error) {
	return f.logged, nil
}

func (f fakeStore) SetEntry(_ context.Context, habitID int64, day time.Time, value int) error {
	if f.entries != nil {
		*f.entries = append(*f.entries, entryCall{habitID, day, value})
	}
	return f.entryErr
}

func (f fakeStore) DayStats(context.Context, time.Time, time.Time) ([]habit.DayStat, error) {
	return f.stats, nil
}

func (f fakeStore) DayNote(context.Context, time.Time) (string, error) { return f.note, nil }

func (f fakeStore) SetNote(_ context.Context, day time.Time, text string) error {
	if f.notes != nil {
		*f.notes = append(*f.notes, noteCall{day, text})
	}
	return nil
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
