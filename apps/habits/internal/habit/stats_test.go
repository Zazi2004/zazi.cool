package habit

import (
	"testing"
	"time"
)

func d(t *testing.T, s string) time.Time {
	t.Helper()
	day, err := ParseDay(s)
	if err != nil {
		t.Fatal(err)
	}
	return day
}

func doneOn(t *testing.T, days ...string) map[time.Time]bool {
	m := map[time.Time]bool{}
	for _, s := range days {
		m[d(t, s)] = true
	}
	return m
}

const (
	mon Weekdays = 1 << iota
	tue
	wed
	thu
	fri
	sat
	sun
)

// 2026-10-26 is a Monday.
func TestApplies(t *testing.T) {
	archived := d(t, "2026-10-29")
	h := History{
		CreatedOn: d(t, "2026-10-10"),
		Schedules: []ScheduleVersion{
			{d(t, "2026-10-10"), AllDays},
			{d(t, "2026-10-26"), mon},
		},
		ArchivedOn: &archived,
	}
	tests := []struct {
		day  string
		want bool
	}{
		{"2026-10-09", false}, // before creation (CA9)
		{"2026-10-10", true},
		{"2026-10-25", true},  // Sunday, old schedule
		{"2026-10-26", true},  // Monday, new schedule
		{"2026-10-27", false}, // Tuesday, new schedule
		{"2026-11-02", false}, // Monday but archived
	}
	for _, tt := range tests {
		if got := h.Applies(d(t, tt.day)); got != tt.want {
			t.Errorf("Applies(%s) = %v, want %v", tt.day, got, tt.want)
		}
	}
}

// CA5: restricting the days from today leaves the past week unchanged.
func TestTallyKeepsPastSchedule(t *testing.T) {
	h := History{
		CreatedOn: d(t, "2026-10-01"),
		Schedules: []ScheduleVersion{{d(t, "2026-10-01"), AllDays}, {d(t, "2026-10-26"), mon}},
		Done:      doneOn(t, "2026-10-19", "2026-10-20", "2026-10-21"),
	}
	today := d(t, "2026-10-28")
	got := h.Tally(d(t, "2026-10-19"), d(t, "2026-10-25"), today)
	if want := (Tally{3, 7}); got != want {
		t.Errorf("past week = %+v, want %+v", got, want)
	}
}

// CA7: a mon/wed/fri habit counts only those days.
func TestTallyOnlyScheduledDays(t *testing.T) {
	h := History{
		CreatedOn: d(t, "2026-01-01"),
		Schedules: []ScheduleVersion{{d(t, "2026-01-01"), mon | wed | fri}},
		Done:      doneOn(t, "2026-10-26", "2026-10-28"),
	}
	today := d(t, "2026-11-10")
	from, to := d(t, "2026-10-26"), d(t, "2026-11-01")
	if got, want := h.Tally(from, to, today), (Tally{2, 3}); got != want {
		t.Errorf("tally = %+v, want %+v", got, want)
	}
	if got := h.StateOn(d(t, "2026-10-27"), today); got != NotApplicable {
		t.Errorf("Tuesday = %v, want NotApplicable", got)
	}
}

// CA9: nothing counts before creation, and future days never lower the rate.
func TestTallyIgnoresBeforeCreationAndFuture(t *testing.T) {
	h := History{
		CreatedOn: d(t, "2026-10-10"),
		Schedules: []ScheduleVersion{{d(t, "2026-10-10"), AllDays}},
		Done:      doneOn(t, "2026-10-10", "2026-10-11"),
	}
	today := d(t, "2026-10-12")
	got := h.Tally(d(t, "2026-10-01"), d(t, "2026-10-31"), today)
	if want := (Tally{2, 3}); got != want {
		t.Errorf("tally = %+v, want %+v", got, want)
	}
	if got := h.StateOn(d(t, "2026-10-13"), today); got != StateFuture {
		t.Errorf("future day = %v, want StateFuture", got)
	}
}

// CA8: 2 of 3 habits done is 67 %.
func TestDayTallyRounds(t *testing.T) {
	day := d(t, "2026-10-26")
	mk := func(done bool) History {
		h := History{CreatedOn: day, Schedules: []ScheduleVersion{{day, AllDays}}, Done: map[time.Time]bool{}}
		h.Done[day] = done
		return h
	}
	got := DayTally([]History{mk(true), mk(true), mk(false)}, day, day)
	if pct, ok := got.Percent(); !ok || pct != 67 {
		t.Errorf("percent = %d, %v; want 67", pct, ok)
	}
	if s := got.String(); s != "67 %" {
		t.Errorf("String = %q", s)
	}
	if s := (Tally{}).String(); s != "—" {
		t.Errorf("empty tally = %q", s)
	}
}

// CA10 and friends.
func TestStreak(t *testing.T) {
	// 2026-10-28 is a Wednesday.
	today := d(t, "2026-10-28")
	daily := func(done ...string) History {
		return History{CreatedOn: d(t, "2026-09-01"), Schedules: []ScheduleVersion{{d(t, "2026-09-01"), AllDays}}, Done: doneOn(t, done...)}
	}
	tests := []struct {
		name string
		h    History
		want int
	}{
		{"three days including today", daily("2026-10-26", "2026-10-27", "2026-10-28"), 3},
		{"today not checked yet keeps yesterday's streak", daily("2026-10-25", "2026-10-26", "2026-10-27"), 3},
		{"a missed due day resets", daily("2026-10-26", "2026-10-28"), 1},
		{"yesterday missed and today unchecked", daily("2026-10-25"), 0},
		{"stops at creation", History{CreatedOn: d(t, "2026-10-27"), Schedules: []ScheduleVersion{{d(t, "2026-10-27"), AllDays}}, Done: doneOn(t, "2026-10-27", "2026-10-28")}, 2},
		{
			"days not scheduled do not break it",
			History{
				CreatedOn: d(t, "2026-09-01"),
				Schedules: []ScheduleVersion{{d(t, "2026-09-01"), mon | wed | fri}},
				Done:      doneOn(t, "2026-10-23", "2026-10-26", "2026-10-28"),
			}, 3,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.h.Streak(today); got != tt.want {
				t.Errorf("streak = %d, want %d", got, tt.want)
			}
		})
	}

	archived := d(t, "2026-10-28")
	h := daily("2026-10-26", "2026-10-27")
	h.ArchivedOn = &archived
	if got := h.Streak(today); got != 0 {
		t.Errorf("archived streak = %d, want 0", got)
	}
}

func TestMonday(t *testing.T) {
	for in, want := range map[string]string{"2026-10-26": "2026-10-26", "2026-10-28": "2026-10-26", "2026-11-01": "2026-10-26"} {
		if got := FormatDay(Monday(d(t, in))); got != want {
			t.Errorf("Monday(%s) = %s, want %s", in, got, want)
		}
	}
}
