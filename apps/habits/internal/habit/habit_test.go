package habit

import (
	"strings"
	"testing"
	"time"
)

func TestWeekdays(t *testing.T) {
	w, err := ParseWeekdays([]string{"1", "3", "5"})
	if err != nil {
		t.Fatal(err)
	}
	if w != 0b0010101 {
		t.Errorf("mask = %07b, want 0010101", w)
	}
	want := map[time.Weekday]bool{
		time.Monday: true, time.Tuesday: false, time.Wednesday: true, time.Thursday: false,
		time.Friday: true, time.Saturday: false, time.Sunday: false,
	}
	for d, has := range want {
		if w.Has(d) != has {
			t.Errorf("Has(%v) = %v, want %v", d, w.Has(d), has)
		}
	}
	if sunday, _ := ParseWeekdays([]string{"7"}); !sunday.Has(time.Sunday) || sunday != 0b1000000 {
		t.Errorf("Sunday mask = %07b", sunday)
	}
}

func TestParseWeekdaysRejects(t *testing.T) {
	for _, values := range [][]string{nil, {"0"}, {"8"}, {"x"}, {""}} {
		if _, err := ParseWeekdays(values); err == nil {
			t.Errorf("ParseWeekdays(%q): want an error", values)
		}
	}
}

func TestValidateName(t *testing.T) {
	tests := []struct {
		in, want string
		ok       bool
	}{
		{"Lire", "Lire", true},
		{"  Lire  ", "Lire", true},
		{"<script>", "<script>", true},
		{"", "", false},
		{"   ", "", false},
		{strings.Repeat("é", 60), strings.Repeat("é", 60), true},
		{strings.Repeat("é", 61), "", false},
		{"a\nb", "", false},
	}
	for _, tt := range tests {
		got, err := ValidateName(tt.in)
		if (err == nil) != tt.ok || got != tt.want {
			t.Errorf("ValidateName(%q) = %q, %v; want %q, ok=%v", tt.in, got, err, tt.want, tt.ok)
		}
	}
}

func TestParseDay(t *testing.T) {
	if _, err := ParseDay("2026-10-30"); err != nil {
		t.Errorf("valid day: %v", err)
	}
	for _, s := range []string{"", "2026-1-5", "2026-02-30", "30/10/2026", "x"} {
		if _, err := ParseDay(s); err == nil {
			t.Errorf("ParseDay(%q): want an error", s)
		}
	}
}

// CA4: 23:30 UTC on 30 June is 1 July in Paris.
func TestTodayUsesTheConfiguredZone(t *testing.T) {
	paris, err := time.LoadLocation("Europe/Paris")
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, time.June, 30, 23, 30, 0, 0, time.UTC)
	if got := FormatDay(Today(now, paris)); got != "2026-07-01" {
		t.Errorf("Today = %s, want 2026-07-01", got)
	}
	if got := FormatDay(Today(now, time.UTC)); got != "2026-06-30" {
		t.Errorf("Today in UTC = %s, want 2026-06-30", got)
	}
}

// CA3: the window is inclusive; the day after it and the future are refused.
func TestEditable(t *testing.T) {
	today, _ := ParseDay("2026-10-30")
	tests := []struct {
		day  string
		want bool
	}{
		{"2026-10-30", true},
		{"2026-10-27", true},
		{"2026-10-16", true},  // today - 14
		{"2026-10-15", false}, // today - 15
		{"2026-10-31", false},
	}
	for _, tt := range tests {
		day, _ := ParseDay(tt.day)
		if got := Editable(day, today, 14); got != tt.want {
			t.Errorf("Editable(%s) = %v, want %v", tt.day, got, tt.want)
		}
	}
}
