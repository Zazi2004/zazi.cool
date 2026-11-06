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
		Now:            func() time.Time { return time.Date(2026, time.November, 6, 10, 0, 0, 0, time.UTC) },
		EditWindowDays: 7,
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

func TestUpdateHabit(t *testing.T) {
	var patches []habit.Patch
	h := habitsHandler(fakeStore{patches: &patches})
	rec := apiRequest(h, http.MethodPatch, "/api/habits/3", `{"target":4}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d (%s)", rec.Code, rec.Body.String())
	}
	if len(patches) != 1 || patches[0].Target == nil || *patches[0].Target != 4 || patches[0].Name != nil {
		t.Errorf("patch = %+v, want only the target", patches)
	}
	var got habitJSON
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil || got.ID != 3 || got.Target != 4 {
		t.Errorf("body = %s", rec.Body.String())
	}
}

func TestHabitEditErrors(t *testing.T) {
	tests := []struct {
		name         string
		err          error
		method, path string
		body         string
		status       int
		code         string
	}{
		{"update unknown", habit.ErrNotFound, http.MethodPatch, "/api/habits/3", `{}`, 404, "not_found"},
		{"update invalid", &habit.ValidationError{Message: "Cible"}, http.MethodPatch, "/api/habits/3", `{}`, 422, "invalid_habit"},
		{"update name taken", habit.ErrNameTaken, http.MethodPatch, "/api/habits/3", `{}`, 409, "name_taken"},
		{"restore name taken", habit.ErrNameTaken, http.MethodPost, "/api/habits/3/restore", "", 409, "name_taken"},
		{"archive unknown", habit.ErrNotFound, http.MethodPost, "/api/habits/3/archive", "", 404, "not_found"},
		{"bad id", nil, http.MethodPatch, "/api/habits/abc", `{}`, 404, "not_found"},
		{"order invalid", &habit.ValidationError{Message: "Ordre"}, http.MethodPut, "/api/habits/order", `{"moment":"day","ids":[1]}`, 422, "invalid_order"},
		{"order unknown moment", nil, http.MethodPut, "/api/habits/order", `{"moment":"night","ids":[1]}`, 422, "invalid_order"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := apiRequest(habitsHandler(fakeStore{editErr: tt.err}), tt.method, tt.path, tt.body)
			if rec.Code != tt.status || errorCode(t, rec) != tt.code {
				t.Errorf("= %d %s, want %d %s", rec.Code, rec.Body.String(), tt.status, tt.code)
			}
		})
	}
}

func TestArchiveAndRestoreHabit(t *testing.T) {
	h := habitsHandler(fakeStore{})
	var archived habitJSON
	rec := apiRequest(h, http.MethodPost, "/api/habits/3/archive", "")
	if err := json.Unmarshal(rec.Body.Bytes(), &archived); err != nil || rec.Code != 200 ||
		archived.ArchivedOn == nil || *archived.ArchivedOn != "2026-11-06" {
		t.Errorf("archive = %d %s", rec.Code, rec.Body.String())
	}
	var restored habitJSON
	rec = apiRequest(h, http.MethodPost, "/api/habits/3/restore", "")
	if err := json.Unmarshal(rec.Body.Bytes(), &restored); err != nil || rec.Code != 200 || restored.ArchivedOn != nil {
		t.Errorf("restore = %d %s", rec.Code, rec.Body.String())
	}
}

func TestReorderHabits(t *testing.T) {
	var orders []orderCall
	h := habitsHandler(fakeStore{orders: &orders})
	rec := apiRequest(h, http.MethodPut, "/api/habits/order", `{"moment":"morning","ids":[3,1,2]}`)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d (%s)", rec.Code, rec.Body.String())
	}
	if len(orders) != 1 || orders[0].moment != habit.Morning || len(orders[0].ids) != 3 || orders[0].ids[0] != 3 {
		t.Errorf("orders = %+v", orders)
	}
}

func TestEditRoutesRequireSession(t *testing.T) {
	h := habitsHandler(fakeStore{})
	for _, r := range [][2]string{
		{http.MethodPatch, "/api/habits/3"}, {http.MethodPost, "/api/habits/3/archive"},
		{http.MethodPost, "/api/habits/3/restore"}, {http.MethodPut, "/api/habits/order"},
	} {
		req := httptest.NewRequest(r[0], r[1], strings.NewReader(`{}`))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if rec.Code != http.StatusUnauthorized {
			t.Errorf("%s %s without session = %d, want 401", r[0], r[1], rec.Code)
		}
	}
}

func TestListHabitsEmptyListsAreArrays(t *testing.T) {
	rec := apiRequest(habitsHandler(fakeStore{}), http.MethodGet, "/api/habits", "")
	if got := strings.TrimSpace(rec.Body.String()); got != `{"active":[],"archived":[],"ruleStarts":{"day":"2026-11-06","month":"2026-11-01","week":"2026-11-02","year":"2026-01-01"}}` {
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
	var stats []habit.DayStat
	for _, d := range []string{"2026-11-03", "2026-11-04", "2026-11-05"} {
		day, _ := habit.ParseDay(d)
		stats = append(stats, habit.DayStat{Day: day, Total: 3, Done: 3, ProgressSum: 3})
	}
	today, _ := habit.ParseDay("2026-11-06")
	stats = append(stats, habit.DayStat{Day: today, Total: 3, Done: 1, ProgressSum: 1.5})
	dh := habitsHandler(fakeStore{logged: logged, stats: stats, note: "Bonne journée"})
	writeGolden(t, "day", apiRequest(dh, http.MethodGet, "/api/days/2026-11-06", ""))

	// The stats goldens: the weekly habit switched to monthly shows every status.
	sh := habitsHandler(fakeStore{histories: statsHistories()})
	writeGolden(t, "stats", apiRequest(sh, http.MethodGet, "/api/stats", ""))
	writeGolden(t, "habit-stats", apiRequest(sh, http.MethodGet, "/api/stats/habits/2", ""))
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
