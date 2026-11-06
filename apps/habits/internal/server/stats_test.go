package server

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"zazi.cool/habits/internal/habit"
)

func statsDay(s string) time.Time {
	d, err := habit.ParseDay(s)
	if err != nil {
		panic(err)
	}
	return d
}

// statsHistories: a weekly habit switched to monthly (interrupted week), a daily one
// and a yearly one; today is Friday 6 November 2026.
func statsHistories() []habit.History {
	return []habit.History{
		{
			Habit: habit.Habit{
				ID: 1, Name: "Vitamine D", Emoji: "💊", Color: "amber", Moment: habit.Morning,
				Unit: "pilules", Step: 1, Period: habit.PeriodDay, Target: 2, Weekdays: habit.AllDays,
			},
			Rules: []habit.RuleVersion{{ValidFrom: statsDay("2026-11-01"), Rule: habit.Rule{Period: habit.PeriodDay, Target: 2, Weekdays: habit.AllDays}}},
			Entries: map[time.Time]int{
				statsDay("2026-11-01"): 2, statsDay("2026-11-02"): 2, statsDay("2026-11-03"): 1,
				statsDay("2026-11-04"): 2, statsDay("2026-11-05"): 2, statsDay("2026-11-06"): 1,
			},
		},
		{
			Habit: habit.Habit{
				ID: 2, Name: "Course", Emoji: "🏃", Color: "green", Moment: habit.Evening,
				Step: 1, Period: habit.PeriodMonth, Target: 10, Weekdays: habit.AllDays,
			},
			Rules: []habit.RuleVersion{
				{ValidFrom: statsDay("2026-10-12"), Rule: habit.Rule{Period: habit.PeriodWeek, Target: 3, Weekdays: habit.AllDays}},
				{ValidFrom: statsDay("2026-11-01"), Rule: habit.Rule{Period: habit.PeriodMonth, Target: 10, Weekdays: habit.AllDays}},
			},
			Entries: map[time.Time]int{
				statsDay("2026-10-13"): 3, statsDay("2026-10-20"): 2, statsDay("2026-10-27"): 3,
				statsDay("2026-11-03"): 4,
			},
		},
		{
			Habit: habit.Habit{
				ID: 3, Name: "Sport", Emoji: "🏋️", Color: "red", Moment: habit.Day,
				Unit: "séances", Step: 1, Period: habit.PeriodYear, Target: 150, Weekdays: habit.AllDays,
			},
			Rules:   []habit.RuleVersion{{ValidFrom: statsDay("2026-01-01"), Rule: habit.Rule{Period: habit.PeriodYear, Target: 150, Weekdays: habit.AllDays}}},
			Entries: map[time.Time]int{statsDay("2026-03-01"): 127, statsDay("2026-11-06"): 1},
		},
	}
}

func TestStatsOverview(t *testing.T) {
	rec := apiRequest(habitsHandler(fakeStore{histories: statsHistories()}), http.MethodGet, "/api/stats", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d (%s)", rec.Code, rec.Body.String())
	}
	var got struct {
		Habits []summaryJSON `json:"habits"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil || len(got.Habits) != 3 {
		t.Fatalf("body = %s (err %v)", rec.Body.String(), err)
	}
	vit, course, sport := got.Habits[0], got.Habits[1], got.Habits[2]
	if vit.Total != 1 || vit.Complete || vit.Streak == nil || *vit.Streak != 2 {
		t.Errorf("daily summary = %+v streak %v", vit, vit.Streak)
	}
	if course.Total != 4 || course.Streak == nil || *course.Streak != 0 {
		t.Errorf("monthly summary = %+v", course)
	}
	if sport.Total != 128 || sport.Streak != nil || sport.Pace == nil || sport.Pace.Label != "1 d'avance" {
		t.Errorf("yearly summary = %+v: no streak and 1 d'avance expected", sport)
	}
}

func TestStatsEmptyOverviewIsAnArray(t *testing.T) {
	rec := apiRequest(habitsHandler(fakeStore{}), http.MethodGet, "/api/stats", "")
	if got := strings.TrimSpace(rec.Body.String()); got != `{"habits":[]}` {
		t.Errorf("body = %s", got)
	}
}

func TestHabitStats(t *testing.T) {
	h := habitsHandler(fakeStore{histories: statsHistories()})
	get := func(path string) statsJSON {
		t.Helper()
		rec := apiRequest(h, http.MethodGet, path, "")
		if rec.Code != http.StatusOK {
			t.Fatalf("%s = %d (%s)", path, rec.Code, rec.Body.String())
		}
		var s statsJSON
		if err := json.Unmarshal(rec.Body.Bytes(), &s); err != nil {
			t.Fatal(err)
		}
		return s
	}

	// Weekly then monthly: the week of 26 October is cut on the 31st (CA5).
	course := get("/api/stats/habits/2")
	if course.Year != 2026 || len(course.Days) != 365 || course.BestStreak == nil {
		t.Fatalf("course = year %d, %d days", course.Year, len(course.Days))
	}
	var interrupted []periodJSON
	for _, p := range course.Periods {
		if p.Status == "interrupted" {
			interrupted = append(interrupted, p)
		}
	}
	if len(interrupted) != 1 || interrupted[0].Start != "2026-10-26" || interrupted[0].End != "2026-10-31" {
		t.Errorf("interrupted periods = %+v", interrupted)
	}
	if course.Periods[0].Start != "2026-11-01" || course.Periods[0].Status != "current" {
		t.Errorf("periods are newest first, got %+v", course.Periods[0])
	}

	// A daily habit lists no periods; the heatmap carries the levels.
	vit := get("/api/stats/habits/1?year=2026")
	if len(vit.Periods) != 0 || vit.Periods == nil {
		t.Errorf("daily periods = %#v, want an empty list", vit.Periods)
	}
	if lvl := vit.Days[304]; lvl.Date != "2026-11-01" || lvl.Level == nil || *lvl.Level != 4 {
		t.Errorf("1 November = %+v, want level 4", lvl)
	}
	if vit.Days[0].Level != nil {
		t.Errorf("1 January is before the habit: level = %v, want nil", *vit.Days[0].Level)
	}
	if vit.Rate30 == nil || vit.RateYear == nil {
		t.Errorf("rates = %v, %v", vit.Rate30, vit.RateYear)
	}

	// No streak, best streak nor rate for a yearly habit still short of its target.
	sport := get("/api/stats/habits/3")
	if sport.Streak != nil || sport.BestStreak != nil || sport.RateYear != nil {
		t.Errorf("yearly stats = streak %v best %v rate %v", sport.Streak, sport.BestStreak, sport.RateYear)
	}

	// An earlier year has no days of this habit.
	past := get("/api/stats/habits/1?year=2025")
	if len(past.Days) != 365 || past.Days[10].Level != nil || past.RateYear != nil {
		t.Errorf("2025 = %d days, rate %v", len(past.Days), past.RateYear)
	}
}

func TestHabitStatsErrors(t *testing.T) {
	h := habitsHandler(fakeStore{histories: statsHistories()})
	tests := []struct {
		path   string
		status int
		code   string
	}{
		{"/api/stats/habits/99", 404, "not_found"},
		{"/api/stats/habits/abc", 404, "not_found"},
		{"/api/stats/habits/1?year=abc", 422, "invalid_year"},
		{"/api/stats/habits/1?year=2027", 422, "invalid_year"},
		{"/api/stats/habits/1?year=1999", 422, "invalid_year"},
	}
	for _, tt := range tests {
		rec := apiRequest(h, http.MethodGet, tt.path, "")
		if rec.Code != tt.status || errorCode(t, rec) != tt.code {
			t.Errorf("%s = %d %s, want %d %s", tt.path, rec.Code, rec.Body.String(), tt.status, tt.code)
		}
	}
}

func TestStatsRequireSession(t *testing.T) {
	h := habitsHandler(fakeStore{})
	for _, path := range []string{"/api/stats", "/api/stats/habits/1"} {
		if rec := get(t, h, http.MethodGet, path); rec.Code != http.StatusUnauthorized {
			t.Errorf("%s without session = %d, want 401", path, rec.Code)
		}
	}
}
