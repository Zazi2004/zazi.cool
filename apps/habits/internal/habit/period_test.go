package habit

import (
	"math"
	"testing"
	"time"
)

func date(s string) time.Time {
	d, err := ParseDay(s)
	if err != nil {
		panic(err)
	}
	return d
}

func TestPeriodStartAndDays(t *testing.T) {
	tests := []struct {
		name   string
		period Period
		day    string
		start  string
		days   int
	}{
		{"day", PeriodDay, "2026-11-06", "2026-11-06", 1},
		{"week from a Wednesday", PeriodWeek, "2026-11-04", "2026-11-02", 7},
		{"week on its Monday", PeriodWeek, "2026-11-02", "2026-11-02", 7},
		{"week on its Sunday", PeriodWeek, "2026-11-08", "2026-11-02", 7},
		{"week across two months", PeriodWeek, "2026-12-01", "2026-11-30", 7},
		{"week across two years", PeriodWeek, "2027-01-01", "2026-12-28", 7},
		{"month of 30 days", PeriodMonth, "2026-11-06", "2026-11-01", 30},
		{"february", PeriodMonth, "2027-02-15", "2027-02-01", 28},
		{"leap february", PeriodMonth, "2028-02-29", "2028-02-01", 29},
		{"year", PeriodYear, "2026-11-06", "2026-01-01", 365},
		{"leap year", PeriodYear, "2028-03-01", "2028-01-01", 366},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := date(tt.day)
			if got := FormatDay(PeriodStart(tt.period, d)); got != tt.start {
				t.Errorf("PeriodStart = %s, want %s", got, tt.start)
			}
			if got := PeriodDays(tt.period, d); got != tt.days {
				t.Errorf("PeriodDays = %d, want %d", got, tt.days)
			}
		})
	}
}

func TestNewPace(t *testing.T) {
	tests := []struct {
		name          string
		period        Period
		day           string
		total, target int
		expected      float64
		diff          int
		label         string
	}{
		{"CA2: year ahead", PeriodYear, "2026-11-06", 128, 150, 127.4, 1, "1 d'avance"},
		{"year behind", PeriodYear, "2026-11-06", 120, 150, 127.4, -7, "7 de retard"},
		{"year on pace", PeriodYear, "2026-11-06", 127, 150, 127.4, 0, "dans le rythme"},
		{"leap year uses 366 days", PeriodYear, "2028-12-31", 100, 150, 150, -50, "50 de retard"},
		{"week, first day", PeriodWeek, "2026-11-02", 0, 7, 1, -1, "1 de retard"},
		{"week reached", PeriodWeek, "2026-11-04", 3, 3, 1.285714285714, 2, "objectif atteint"},
		{"week exceeded", PeriodWeek, "2026-11-04", 4, 3, 1.285714285714, 3, "objectif atteint"},
		{"month", PeriodMonth, "2026-11-15", 10, 20, 10, 0, "dans le rythme"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewPace(tt.period, date(tt.day), tt.total, tt.target)
			if p == nil {
				t.Fatal("pace = nil")
			}
			if math.Abs(p.Expected-tt.expected) > 5e-3 || p.Diff != tt.diff || p.Label != tt.label {
				t.Errorf("pace = %+v, want %v / %d / %q", *p, tt.expected, tt.diff, tt.label)
			}
		})
	}
	if p := NewPace(PeriodDay, date("2026-11-06"), 1, 2); p != nil {
		t.Errorf("day period has a pace: %+v", p)
	}
}
