package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"zazi.cool/habits/internal/habit"
)

func habitsHandler(store fakeStore) http.Handler {
	return NewHandler(Config{
		Store: store, Auth: fakeAuth{}, Static: testStatic, BaseURL: testBaseURL, LoginLimit: 10,
		Now: func() time.Time { return time.Date(2026, time.November, 6, 10, 0, 0, 0, time.UTC) },
	})
}

func apiRequest(h http.Handler, method, path, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, withSession(req))
	return rec
}

const vitaminD = `{"name":"Vitamine D","emoji":"💊","color":"amber","moment":"morning","unit":"pilules","step":1,"period":"day","target":2,"weekdays":127}`

func TestCreateHabit(t *testing.T) {
	var created []habit.Input
	h := habitsHandler(fakeStore{created: &created})

	rec := apiRequest(h, http.MethodPost, "/api/habits", vitaminD)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201 (body %q)", rec.Code, rec.Body.String())
	}
	var got habitJSON
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.ID != 7 || got.Name != "Vitamine D" || got.Target != 2 || got.Unit != "pilules" || got.ArchivedOn != nil {
		t.Errorf("habit = %+v", got)
	}
	if len(created) != 1 || created[0].Weekdays != habit.AllDays {
		t.Errorf("store received %+v", created)
	}
}

func TestCreateHabitRequiresSession(t *testing.T) {
	h := habitsHandler(fakeStore{})
	for _, req := range []*http.Request{
		httptest.NewRequest(http.MethodGet, "/api/habits", nil),
		httptest.NewRequest(http.MethodPost, "/api/habits", strings.NewReader(vitaminD)),
	} {
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if rec.Code != http.StatusUnauthorized {
			t.Errorf("%s without session = %d, want 401", req.Method, rec.Code)
		}
	}
}

func TestCreateHabitNameTaken(t *testing.T) {
	h := habitsHandler(fakeStore{createErr: habit.ErrNameTaken})
	rec := apiRequest(h, http.MethodPost, "/api/habits", vitaminD)
	if rec.Code != http.StatusConflict || errorCode(t, rec) != "name_taken" {
		t.Errorf("duplicate name = %d %q, want 409 name_taken", rec.Code, rec.Body.String())
	}
}

func TestCreateHabitInvalid(t *testing.T) {
	h := habitsHandler(fakeStore{})
	tests := map[string]string{
		"empty name":     strings.Replace(vitaminD, `"Vitamine D"`, `""`, 1),
		"unknown color":  strings.Replace(vitaminD, `"amber"`, `"mauve"`, 1),
		"unknown moment": strings.Replace(vitaminD, `"morning"`, `"night"`, 1),
		"target 0":       strings.Replace(vitaminD, `"target":2`, `"target":0`, 1),
		"weekdays wrap":  strings.Replace(vitaminD, `"weekdays":127`, `"weekdays":256`, 1),
		"missing fields": `{}`,
	}
	for name, body := range tests {
		t.Run(name, func(t *testing.T) {
			rec := apiRequest(h, http.MethodPost, "/api/habits", body)
			if rec.Code != http.StatusUnprocessableEntity || errorCode(t, rec) != "invalid_habit" {
				t.Errorf("status = %d %q, want 422 invalid_habit", rec.Code, rec.Body.String())
			}
		})
	}
	if rec := apiRequest(h, http.MethodPost, "/api/habits", "{nope"); rec.Code != http.StatusBadRequest {
		t.Errorf("invalid JSON = %d, want 400", rec.Code)
	}
}

func TestListHabitsEmptyListsAreArrays(t *testing.T) {
	rec := apiRequest(habitsHandler(fakeStore{}), http.MethodGet, "/api/habits", "")
	if got := strings.TrimSpace(rec.Body.String()); got != `{"active":[],"archived":[]}` {
		t.Errorf("body = %s", got)
	}
}

// Golden files (spec NF9): the JSON the real handlers produce for each response type,
// written next to the front, whose tests check it against the TypeScript types. A field
// renamed on one side only breaks `make test` (type check of the front).
func TestGoldenFiles(t *testing.T) {
	archivedOn := time.Date(2026, time.October, 12, 0, 0, 0, 0, time.UTC)
	active := []habit.Habit{
		{
			ID: 1, Name: "Vitamine D", Emoji: "💊", Color: "amber", Moment: habit.Morning, Position: 0,
			Unit: "pilules", Step: 1, Period: habit.PeriodDay, Target: 2, Weekdays: habit.AllDays,
		},
		{
			ID: 2, Name: "Course", Emoji: "🏃", Color: "green", Moment: habit.Evening, Position: 0,
			Step: 1, Period: habit.PeriodWeek, Target: 3, Weekdays: 0b0011111,
		},
	}
	archived := []habit.Habit{{
		ID: 3, Name: "Piano", Emoji: "🎹", Color: "violet", Moment: habit.Day, Position: 0,
		Unit: "min", Step: 5, Period: habit.PeriodDay, Target: 20, Weekdays: habit.AllDays, ArchivedOn: &archivedOn,
	}}
	h := habitsHandler(fakeStore{active: active, archived: archived})

	writeGolden(t, "habit", apiRequest(h, http.MethodPost, "/api/habits", vitaminD))
	writeGolden(t, "habit-list", apiRequest(h, http.MethodGet, "/api/habits", ""))

	// The Day golden: a morning habit done (2 / 2) and one pending, plus an evening one.
	logged := []habit.Logged{
		{Habit: active[0], Value: 2, Total: 2},
		{Habit: habit.Habit{
			ID: 4, Name: "Étirements", Emoji: "🧘", Color: "teal", Moment: habit.Morning, Position: 1,
			Step: 1, Period: habit.PeriodDay, Target: 1, Weekdays: habit.AllDays,
		}},
		{Habit: habit.Habit{
			ID: 5, Name: "Lecture", Emoji: "📖", Color: "blue", Moment: habit.Evening, Position: 0,
			Unit: "pages", Step: 5, Period: habit.PeriodDay, Target: 20, Weekdays: habit.AllDays,
		}, Value: 10, Total: 10},
		{Habit: habit.Habit{
			ID: 6, Name: "Sport", Emoji: "🏋️", Color: "red", Moment: habit.Day, Position: 0,
			Unit: "séances", Step: 1, Period: habit.PeriodYear, Target: 150, Weekdays: habit.AllDays,
		}, Value: 1, Total: 128},
	}
	dh := habitsHandler(fakeStore{logged: logged})
	writeGolden(t, "day", apiRequest(dh, http.MethodGet, "/api/days/2026-11-06", ""))
}

func writeGolden(t *testing.T, name string, rec *httptest.ResponseRecorder) {
	t.Helper()
	if rec.Code != http.StatusOK && rec.Code != http.StatusCreated {
		t.Fatalf("%s: status %d (%s)", name, rec.Code, rec.Body.String())
	}
	var out bytes.Buffer
	if err := json.Indent(&out, bytes.TrimSpace(rec.Body.Bytes()), "", "  "); err != nil {
		t.Fatal(err)
	}
	out.WriteByte('\n')
	dir := filepath.Join("..", "..", "web", "src", "golden")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, name+".json"), out.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
}
