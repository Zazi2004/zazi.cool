package server

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"zazi.cool/habits/internal/habit"
)

func loggedHabit(id int64, name string, moment habit.Moment, pos, target, value int) habit.Logged {
	return habit.Logged{
		Habit: habit.Habit{
			ID: id, Name: name, Emoji: "💊", Color: "amber", Moment: moment, Position: pos,
			Step: 1, Period: habit.PeriodDay, Target: target, Weekdays: habit.AllDays,
		},
		Value: value, Total: value,
	}
}

func TestGetDay(t *testing.T) {
	h := habitsHandler(fakeStore{logged: []habit.Logged{
		loggedHabit(1, "Vitamine D", habit.Morning, 0, 2, 2),
		loggedHabit(2, "Étirements", habit.Morning, 1, 1, 0),
	}})
	rec := apiRequest(h, http.MethodGet, "/api/days/2026-11-06", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d (%s)", rec.Code, rec.Body.String())
	}
	var day dayJSON
	if err := json.Unmarshal(rec.Body.Bytes(), &day); err != nil {
		t.Fatal(err)
	}
	if day.Progress != 0.5 || day.Done != 1 || day.Total != 2 || day.Complete {
		t.Errorf("day = %+v, want progress 0.5, 1/2, not complete", day)
	}
	if len(day.Groups) != 1 || day.Groups[0].Moment != "morning" {
		t.Fatalf("groups = %+v", day.Groups)
	}
	// The completed habit goes below the pending one.
	if g := day.Groups[0].Habits; g[0].Habit.ID != 2 || g[1].Habit.ID != 1 || !g[1].Complete {
		t.Errorf("order = %+v", g)
	}
}

func TestGetDayEmptyListsAreArrays(t *testing.T) {
	rec := apiRequest(habitsHandler(fakeStore{}), http.MethodGet, "/api/days/2026-11-06", "")
	for _, want := range []string{`"groups":[]`, `"progress":0`} {
		if !strings.Contains(rec.Body.String(), want) {
			t.Errorf("body %s lacks %s", rec.Body.String(), want)
		}
	}
}

func TestGetDayRange(t *testing.T) {
	h := habitsHandler(fakeStore{})
	tests := []struct {
		date     string
		want     int
		editable string
	}{
		{"2026-11-06", http.StatusOK, `"editable":true`},
		{"2026-10-31", http.StatusOK, `"editable":true`},  // today - 6
		{"2026-10-30", http.StatusOK, `"editable":false`}, // today - 7: read only
		{"2026-11-07", http.StatusUnprocessableEntity, ""},
		{"nope", http.StatusUnprocessableEntity, ""},
	}
	for _, tt := range tests {
		rec := apiRequest(h, http.MethodGet, "/api/days/"+tt.date, "")
		if rec.Code != tt.want || !strings.Contains(rec.Body.String(), tt.editable) {
			t.Errorf("GET %s = %d %s, want %d with %q", tt.date, rec.Code, rec.Body.String(), tt.want, tt.editable)
		}
	}
}

func TestGetDayStreakWeekAndNote(t *testing.T) {
	d := func(s string) time.Time { v, _ := habit.ParseDay(s); return v }
	var stats []habit.DayStat
	// 12 complete days up to yesterday; the week strip shows Friday 6 at half.
	for i := 12; i >= 1; i-- {
		stats = append(stats, habit.DayStat{Day: d("2026-11-06").AddDate(0, 0, -i), Total: 1, Done: 1, ProgressSum: 1})
	}
	stats = append(stats, habit.DayStat{Day: d("2026-11-06"), Total: 2, Done: 1, ProgressSum: 1})
	h := habitsHandler(fakeStore{stats: stats, note: "Bonne journée"})

	rec := apiRequest(h, http.MethodGet, "/api/days/2026-11-06", "")
	var day dayJSON
	if err := json.Unmarshal(rec.Body.Bytes(), &day); err != nil {
		t.Fatal(err)
	}
	if day.Streak != 12 || day.Note != "Bonne journée" {
		t.Errorf("streak = %d, note = %q, want 12 and the note", day.Streak, day.Note)
	}
	if len(day.Week) != 7 || day.Week[0].Date != "2026-11-02" || day.Week[6].Date != "2026-11-08" {
		t.Fatalf("week = %+v", day.Week)
	}
	if day.Week[3].Progress != 1 || day.Week[4].Progress != 0.5 || day.Week[5].Progress != 0 {
		t.Errorf("week progress = %+v", day.Week)
	}
}

func TestPutEntryInsideWindow(t *testing.T) {
	var calls []entryCall
	h := habitsHandler(fakeStore{entries: &calls})
	rec := apiRequest(h, http.MethodPut, "/api/days/2026-10-31/entries/1", `{"value":1}`) // today - 6
	if rec.Code != http.StatusOK || len(calls) != 1 {
		t.Errorf("status = %d, calls = %+v, want 200 and one call", rec.Code, calls)
	}
}

func TestPutNote(t *testing.T) {
	var notes []noteCall
	h := habitsHandler(fakeStore{notes: &notes})
	rec := apiRequest(h, http.MethodPut, "/api/days/2026-11-06/note", `{"text":"Fatigué mais content"}`)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d (%s)", rec.Code, rec.Body.String())
	}
	if len(notes) != 1 || notes[0].text != "Fatigué mais content" || habit.FormatDay(notes[0].day) != "2026-11-06" {
		t.Errorf("notes = %+v", notes)
	}
	// An empty text is valid: it deletes the note.
	if rec := apiRequest(h, http.MethodPut, "/api/days/2026-11-06/note", `{"text":""}`); rec.Code != http.StatusNoContent {
		t.Errorf("empty note = %d", rec.Code)
	}
}

func TestPutNoteRefusals(t *testing.T) {
	tests := []struct {
		name, path, body string
		want             int
	}{
		{"1001 characters", "/api/days/2026-11-06/note", `{"text":"` + strings.Repeat("a", 1001) + `"}`, http.StatusUnprocessableEntity},
		{"missing text", "/api/days/2026-11-06/note", `{}`, http.StatusUnprocessableEntity},
		{"tomorrow", "/api/days/2026-11-07/note", `{"text":"x"}`, http.StatusUnprocessableEntity},
		{"today - 7", "/api/days/2026-10-30/note", `{"text":"x"}`, http.StatusUnprocessableEntity},
		{"invalid json", "/api/days/2026-11-06/note", `{`, http.StatusBadRequest},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var notes []noteCall
			rec := apiRequest(habitsHandler(fakeStore{notes: &notes}), http.MethodPut, tt.path, tt.body)
			if rec.Code != tt.want || len(notes) != 0 {
				t.Errorf("status = %d, saved = %+v, want %d and nothing saved", rec.Code, notes, tt.want)
			}
		})
	}
}

func TestPutEntry(t *testing.T) {
	var calls []entryCall
	h := habitsHandler(fakeStore{entries: &calls, logged: []habit.Logged{loggedHabit(1, "Vitamine D", habit.Morning, 0, 2, 3)}})

	rec := apiRequest(h, http.MethodPut, "/api/days/2026-11-06/entries/1", `{"value":3}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d (%s)", rec.Code, rec.Body.String())
	}
	if len(calls) != 1 || calls[0].habitID != 1 || calls[0].value != 3 || habit.FormatDay(calls[0].day) != "2026-11-06" {
		t.Errorf("store calls = %+v", calls)
	}
	// 3 / 2: the target can be exceeded and the habit stays complete.
	if !strings.Contains(rec.Body.String(), `"total":3`) || !strings.Contains(rec.Body.String(), `"complete":true`) {
		t.Errorf("body = %s", rec.Body.String())
	}
}

func TestPutEntryErrors(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		body     string
		entryErr error
		want     int
	}{
		{"missing value", "/api/days/2026-11-06/entries/1", `{}`, nil, http.StatusUnprocessableEntity},
		{"negative", "/api/days/2026-11-06/entries/1", `{"value":-1}`, nil, http.StatusUnprocessableEntity},
		{"too large", "/api/days/2026-11-06/entries/1", `{"value":100001}`, nil, http.StatusUnprocessableEntity},
		{"invalid json", "/api/days/2026-11-06/entries/1", `{`, nil, http.StatusBadRequest},
		{"tomorrow", "/api/days/2026-11-07/entries/1", `{"value":1}`, nil, http.StatusUnprocessableEntity},
		{"today - 7", "/api/days/2026-10-30/entries/1", `{"value":1}`, nil, http.StatusUnprocessableEntity},
		{"bad id", "/api/days/2026-11-06/entries/x", `{"value":1}`, nil, http.StatusNotFound},
		{"unknown habit", "/api/days/2026-11-06/entries/1", `{"value":1}`, habit.ErrNotFound, http.StatusNotFound},
		{"weekday excluded", "/api/days/2026-11-06/entries/1", `{"value":1}`, habit.ErrDayNotAllowed, http.StatusUnprocessableEntity},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var calls []entryCall
			h := habitsHandler(fakeStore{entries: &calls, entryErr: tt.entryErr})
			rec := apiRequest(h, http.MethodPut, tt.path, tt.body)
			if rec.Code != tt.want {
				t.Errorf("status = %d, want %d (%s)", rec.Code, tt.want, rec.Body.String())
			}
			if tt.entryErr == nil && tt.want != http.StatusOK && len(calls) != 0 {
				t.Errorf("store was called despite the refusal: %+v", calls)
			}
		})
	}
}
