package server

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"zazi.cool/habits/internal/habit"
)

// statsHistory has "Sport" (Monday, Wednesday, Friday) done on Monday 2026-06-22 only,
// and "<script>x" every day. fixedNow makes today Wednesday 2026-07-01.
func statsHistory() []habit.History {
	day := func(s string) time.Time { d, _ := habit.ParseDay(s); return d }
	created := day("2026-06-01")
	return []habit.History{
		{
			ID: 1, Name: "Sport", CreatedOn: created,
			Schedules: []habit.ScheduleVersion{{ValidFrom: created, Weekdays: 0b0010101}},
			Done:      map[time.Time]bool{day("2026-06-22"): true},
		},
		{
			ID: 2, Name: "<script>x", CreatedOn: created,
			Schedules: []habit.ScheduleVersion{{ValidFrom: created, Weekdays: habit.AllDays}},
			Done:      map[time.Time]bool{},
		},
	}
}

func TestWeekView(t *testing.T) {
	m := &memStore{history: statsHistory()}
	rec := do(newHabitsHandler(t, m), http.MethodGet, "/week/2026-06-24", nil, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	body := rec.Body.String()
	// CA7: Sport is "non concerné" on the four other days, and 1 of 3 due days is done.
	if n := strings.Count(body, `class="na">non concerné`); n != 4 {
		t.Errorf("non concerné cells = %d, want 4", n)
	}
	for _, want := range []string{"Semaine du lundi 22 juin 2026 au dimanche 28 juin 2026", "<td>33\u00a0%</td>", "&lt;script&gt;x"} {
		if !strings.Contains(body, want) {
			t.Errorf("page lacks %q", want)
		}
	}
	if strings.Contains(body, "<script>x") {
		t.Error("habit name is not escaped")
	}
	// The streaks end today (1 July), which is after the displayed week.
	if got := m.histAsk[0][1]; got != "2026-07-01" {
		t.Errorf("history read up to %s, want 2026-07-01", got)
	}
	if !strings.Contains(body, `href="/week/2026-06-15"`) || !strings.Contains(body, `href="/week/2026-06-29"`) {
		t.Error("missing previous or next week link")
	}
}

func TestWeekViewHasNoNextBeyondCurrentWeek(t *testing.T) {
	rec := do(newHabitsHandler(t, &memStore{}), http.MethodGet, "/week/2026-07-01", nil, nil)
	if strings.Contains(rec.Body.String(), `rel="next"`) {
		t.Error("current week links to the future")
	}
}

// CA8: 2 of 3 habits done on a day is rendered at 67 %.
func TestMonthViewColoursDays(t *testing.T) {
	day, _ := habit.ParseDay("2026-06-10")
	mk := func(id int64, done bool) habit.History {
		created, _ := habit.ParseDay("2026-06-01")
		h := habit.History{ID: id, Name: "h", CreatedOn: created,
			Schedules: []habit.ScheduleVersion{{ValidFrom: created, Weekdays: habit.AllDays}}, Done: map[time.Time]bool{}}
		h.Done[day] = done
		return h
	}
	m := &memStore{history: []habit.History{mk(1, true), mk(2, true), mk(3, false)}}
	rec := do(newHabitsHandler(t, m), http.MethodGet, "/month/2026-06", nil, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, `<a href="/day/2026-06-10"><span class="sr">mercredi 10 juin 2026 : </span>10 <small>67 %</small>`) {
		t.Errorf("10 June not rendered at 67 %%:\n%s", body)
	}
	if m.histAsk[0] != [2]string{"2026-06-01", "2026-06-30"} {
		t.Errorf("history range = %v", m.histAsk[0])
	}
}

func TestYearView(t *testing.T) {
	m := &memStore{history: statsHistory()}
	rec := do(newHabitsHandler(t, m), http.MethodGet, "/year/2026", nil, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	body := rec.Body.String()
	if n := strings.Count(body, `<li><a href="/month/2026-`); n != 12 {
		t.Errorf("months = %d, want 12", n)
	}
	// 365 days in the heatmap (one cell per day, each with a screen-reader label).
	if n := strings.Count(body, `<span class="sr">`); n != 365 {
		t.Errorf("heatmap cells = %d, want 365", n)
	}
	if len(m.histAsk) != 1 {
		t.Errorf("history reads = %d, want 1 (two SQL queries at most)", len(m.histAsk))
	}
}

func TestStatsInvalidPeriods(t *testing.T) {
	h := newHabitsHandler(t, &memStore{})
	for _, path := range []string{"/week/nope", "/week/2026-13-01", "/month/2026-6", "/month/2026-13", "/year/26", "/year/abcd"} {
		if rec := do(h, http.MethodGet, path, nil, nil); rec.Code != http.StatusBadRequest {
			t.Errorf("%s = %d, want 400", path, rec.Code)
		}
	}
}

func TestStatsRedirectsToCurrentPeriod(t *testing.T) {
	h := newHabitsHandler(t, &memStore{})
	for path, want := range map[string]string{"/week": "/week/2026-07-01", "/month": "/month/2026-07", "/year": "/year/2026"} {
		rec := do(h, http.MethodGet, path, nil, nil)
		if rec.Code != http.StatusSeeOther || rec.Header().Get("Location") != want {
			t.Errorf("%s = %d %q, want 303 %q", path, rec.Code, rec.Header().Get("Location"), want)
		}
	}
}

func TestStatsRequireSession(t *testing.T) {
	h := NewHandler(Config{Store: fakeStore{}, Auth: fakeAuth{}, BaseURL: testBaseURL, LoginLimit: 10})
	for _, path := range []string{"/week/2026-06-24", "/month/2026-06", "/year/2026"} {
		rec := get(t, h, http.MethodGet, path)
		if rec.Code != http.StatusSeeOther || rec.Header().Get("Location") != "/login" {
			t.Errorf("%s = %d %q, want 303 /login", path, rec.Code, rec.Header().Get("Location"))
		}
	}
}
