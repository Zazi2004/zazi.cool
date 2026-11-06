package habit

import (
	"math"
	"testing"
	"time"
)

// entries builds a history map from day → value pairs.
func entries(pairs ...any) map[time.Time]int {
	m := map[time.Time]int{}
	for i := 0; i < len(pairs); i += 2 {
		m[d(pairs[i].(string))] = pairs[i+1].(int)
	}
	return m
}

var (
	week3 = Rule{PeriodWeek, 3, AllDays}
	today = d("2026-11-06") // a Friday
)

func statuses(ps []PeriodStat) []Status {
	out := make([]Status, len(ps))
	for i, p := range ps {
		out[i] = p.Status
	}
	return out
}

func equalStatuses(a, b []Status) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// CA3: weeks of 3 outings; the ongoing week at 1 / 3 does not break the streak, a past
// week at 2 / 3 resets it.
func TestWeeklyStreak(t *testing.T) {
	rules := []RuleVersion{{d("2026-09-21"), week3}}
	tests := []struct {
		name          string
		entries       map[time.Time]int
		current, best int
	}{
		{
			"ongoing week below target keeps the streak",
			entries("2026-09-21", 3, "2026-09-28", 3, "2026-10-05", 3, "2026-10-12", 2,
				"2026-10-19", 3, "2026-10-26", 3, "2026-11-02", 1),
			2, 3,
		},
		{
			"ongoing week reached counts",
			entries("2026-09-21", 3, "2026-10-26", 3, "2026-11-02", 3),
			2, 2,
		},
		{
			"a past week at 2 / 3 resets to 0",
			entries("2026-09-21", 3, "2026-10-26", 2, "2026-10-27", 0),
			0, 1,
		},
		{"nothing at all", nil, 0, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := History{Rules: rules, Entries: tt.entries}
			current, best := Streaks(h.Periods(today))
			if current != tt.current || best != tt.best {
				t.Errorf("streaks = %d, %d, want %d, %d (%v)", current, best, tt.current, tt.best, statuses(h.Periods(today)))
			}
		})
	}
}

// CA5: week → month on 6 November: the week of 26 October is cut on the 31st and is
// interrupted, whatever was entered; it counts neither in the streak nor in the rate.
func TestInterruptedPeriod(t *testing.T) {
	h := History{
		Rules: []RuleVersion{
			{d("2026-10-12"), week3},
			{d("2026-11-01"), Rule{PeriodMonth, 10, AllDays}},
		},
		Entries: entries("2026-10-12", 3, "2026-10-19", 3, "2026-10-26", 3, "2026-11-03", 2),
	}
	periods := h.Periods(today)
	want := []Status{StatusSuccess, StatusSuccess, StatusInterrupted, StatusCurrent}
	if got := statuses(periods); !equalStatuses(got, want) {
		t.Fatalf("statuses = %v, want %v", got, want)
	}
	cut := periods[2]
	if cut.Start != d("2026-10-26") || cut.End != d("2026-10-31") || cut.Total != 3 {
		t.Errorf("interrupted period = %+v, want 26 to 31 October", cut)
	}
	if current, best := Streaks(periods); current != 2 || best != 2 {
		t.Errorf("streaks = %d, %d, want 2, 2", current, best)
	}
	rate := Rate(periods, d("2026-10-01"), today)
	if rate == nil || *rate != 1 {
		t.Errorf("rate = %v, want 1 (2 of 2, interrupted and ongoing left out)", rate)
	}
}

// A period change cuts only the old period it falls in; earlier ones keep their status.
func TestPeriodChangeCutsOnlyTheLastPeriod(t *testing.T) {
	h := History{
		Rules: []RuleVersion{
			{d("2026-10-05"), week3},
			{d("2026-11-01"), Rule{PeriodMonth, 10, AllDays}},
		},
		Entries: entries("2026-10-26", 1),
	}
	// The week of 26 October ends on Sunday 1 November, the day the month starts: cut.
	// The week of 19 October ends on the 25th: untouched.
	periods := h.Periods(today)
	if periods[3].Status != StatusInterrupted || periods[2].Status != StatusMissed {
		t.Errorf("statuses = %v", statuses(periods))
	}
}

// CA4: raising the target on a Thursday leaves the past weeks on 3.
func TestTargetChangeKeepsThePast(t *testing.T) {
	h := History{
		Rules: []RuleVersion{
			{d("2026-10-19"), week3},
			{d("2026-11-02"), Rule{PeriodWeek, 4, AllDays}},
		},
		Entries: entries("2026-10-19", 3, "2026-10-26", 3, "2026-11-02", 3),
	}
	periods := h.Periods(today)
	want := []Status{StatusSuccess, StatusSuccess, StatusCurrent}
	if got := statuses(periods); !equalStatuses(got, want) {
		t.Fatalf("statuses = %v, want %v", got, want)
	}
	if periods[2].Target != 4 || periods[1].Target != 3 {
		t.Errorf("targets = %d, %d, want 3 then 4", periods[1].Target, periods[2].Target)
	}
}

func TestDayHabitSkipsForbiddenWeekdays(t *testing.T) {
	// Monday to Friday; created on Wednesday 28 October.
	h := History{
		Rules: []RuleVersion{{d("2026-10-28"), Rule{PeriodDay, 1, 0b0011111}}},
		Entries: entries("2026-10-28", 1, "2026-10-29", 1, "2026-10-30", 1, "2026-11-02", 1,
			"2026-11-04", 1, "2026-11-05", 1, "2026-10-31", 1),
	}
	periods := h.Periods(today)
	want := []Status{
		StatusSuccess, StatusSuccess, StatusSuccess, // Wed, Thu, Fri
		StatusSuccess, StatusMissed, StatusSuccess, StatusSuccess, // Mon, Tue, Wed, Thu
		StatusCurrent, // today, Friday
	}
	if got := statuses(periods); !equalStatuses(got, want) {
		t.Fatalf("statuses = %v, want %v", got, want)
	}
	if current, best := Streaks(periods); current != 2 || best != 4 {
		t.Errorf("streaks = %d, %d, want 2, 4 (the weekend is skipped)", current, best)
	}
}

func TestRate(t *testing.T) {
	h := History{
		Rules: []RuleVersion{{d("2026-09-21"), week3}},
		Entries: entries("2026-09-21", 3, "2026-09-28", 3, "2026-10-05", 3, "2026-10-12", 2,
			"2026-10-19", 3, "2026-10-26", 3, "2026-11-02", 1),
	}
	periods := h.Periods(today)
	got := Rate(periods, d("2026-01-01"), d("2026-12-31"))
	if got == nil || math.Abs(*got-5.0/6) > 1e-9 {
		t.Errorf("rate = %v, want 5/6 (ongoing week left out)", got)
	}
	if got := Rate(periods, d("2025-01-01"), d("2025-12-31")); got != nil {
		t.Errorf("rate of a year without periods = %v, want nil", *got)
	}
	// The last 30 days only reach the weeks that overlap them.
	if got := Rate(periods, today.AddDate(0, 0, -29), today); got == nil || *got != 0.75 {
		t.Errorf("rate30 = %v, want 3 of 4", got)
	}
}

func TestSummary(t *testing.T) {
	year := History{
		Rules:   []RuleVersion{{d("2026-01-01"), Rule{PeriodYear, 150, AllDays}}},
		Entries: entries("2026-03-01", 100, "2026-11-05", 27, "2026-11-06", 1),
	}
	s := year.Summary(today)
	if s.Total != 128 || s.Complete || s.Streak != nil || s.Pace == nil || s.Pace.Label != "1 d'avance" {
		t.Errorf("year summary = %+v (pace %+v)", s, s.Pace)
	}

	week := History{Rules: []RuleVersion{{d("2026-10-26"), week3}}, Entries: entries("2026-10-26", 3, "2026-11-02", 1)}
	s = week.Summary(today)
	if s.Total != 1 || s.Streak == nil || *s.Streak != 1 {
		t.Errorf("week summary = %+v", s)
	}
}

func TestArchivedHabitStopsAtArchiveDay(t *testing.T) {
	archived := d("2026-11-04")
	h := History{
		Habit:   Habit{ArchivedOn: &archived},
		Rules:   []RuleVersion{{d("2026-11-01"), Rule{PeriodDay, 1, AllDays}}},
		Entries: entries("2026-11-01", 1, "2026-11-02", 1, "2026-11-03", 1),
	}
	periods := h.Periods(today)
	if len(periods) != 3 || periods[2].End != d("2026-11-03") {
		t.Fatalf("periods = %+v, want 1 to 3 November", periods)
	}
	if current, _ := Streaks(periods); current != 3 {
		t.Errorf("streak = %d, want 3", current)
	}
}

func TestHeatmap(t *testing.T) {
	// Monday to Friday, 2 a day, created on 2026-11-02.
	h := History{
		Rules:   []RuleVersion{{d("2026-11-02"), Rule{PeriodDay, 2, 0b0011111}}},
		Entries: entries("2026-11-02", 2, "2026-11-03", 1, "2026-11-04", 3),
	}
	cells := h.Heatmap(2026)
	if len(cells) != 365 {
		t.Fatalf("%d cells, want 365", len(cells))
	}
	at := func(s string) Cell { return cells[d(s).YearDay()-1] }
	tests := []struct {
		day     string
		allowed bool
		level   int
	}{
		{"2026-10-30", false, 0}, // before the habit existed
		{"2026-11-02", true, 4},
		{"2026-11-03", true, 2}, // 1 of 2
		{"2026-11-04", true, 4}, // exceeding caps at 4
		{"2026-11-05", true, 0},
		{"2026-11-07", false, 0}, // Saturday
	}
	for _, tt := range tests {
		if c := at(tt.day); c.Allowed != tt.allowed || c.Level != tt.level {
			t.Errorf("%s = %+v, want allowed %v level %d", tt.day, c, tt.allowed, tt.level)
		}
	}
	if len((History{Rules: h.Rules}).Heatmap(2028)) != 366 {
		t.Error("2028 should have 366 cells")
	}
}

func TestHeatmapLevelOfPeriodHabitIsRelativeToThePeak(t *testing.T) {
	h := History{
		Rules:   []RuleVersion{{d("2026-01-01"), Rule{PeriodYear, 150, AllDays}}},
		Entries: entries("2026-03-01", 4, "2026-03-02", 1, "2026-03-03", 2),
	}
	cells := h.Heatmap(2026)
	got := []int{cells[59].Level, cells[60].Level, cells[61].Level, cells[62].Level}
	want := []int{4, 1, 2, 0}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("levels = %v, want %v", got, want)
		}
	}
}
