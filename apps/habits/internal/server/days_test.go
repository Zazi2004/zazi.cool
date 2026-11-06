package server

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"zazi.cool/habits/internal/habit"
)

func loggedHabit(id int64, name string, moment habit.Moment, pos, target, value int) habit.Logged {
	return habit.Logged{
		Habit: habit.Habit{
			ID: id, Name: name, Emoji: "💊", Color: "amber", Moment: moment, Position: pos,
			Step: 1, Period: habit.PeriodDay, Target: target, Weekdays: habit.AllDays,
		},
		Value: value,
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
	for _, want := range []string{`"groups":[]`, `"week":[]`, `"progress":0`} {
		if !strings.Contains(rec.Body.String(), want) {
			t.Errorf("body %s lacks %s", rec.Body.String(), want)
		}
	}
}

func TestDayOnlyToday(t *testing.T) {
	h := habitsHandler(fakeStore{})
	for _, date := range []string{"2026-11-05", "2026-11-07", "nope"} {
		rec := apiRequest(h, http.MethodGet, "/api/days/"+date, "")
		if rec.Code != http.StatusUnprocessableEntity {
			t.Errorf("GET %s = %d, want 422", date, rec.Code)
		}
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
		{"other day", "/api/days/2026-11-05/entries/1", `{"value":1}`, nil, http.StatusUnprocessableEntity},
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
		})
	}
}
