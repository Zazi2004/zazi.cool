package habit

import (
	"math"
	"strings"
	"testing"
	"time"
)

func TestProgress(t *testing.T) {
	tests := []struct {
		total, target int
		want          float64
		complete      bool
	}{
		{0, 2, 0, false},
		{1, 2, 0.5, false},
		{2, 2, 1, true},
		{3, 2, 1, true},
		{0, 1, 0, false},
		{1, 1, 1, true},
	}
	for _, tt := range tests {
		if got := Progress(tt.total, tt.target); got != tt.want {
			t.Errorf("Progress(%d, %d) = %v, want %v", tt.total, tt.target, got, tt.want)
		}
		if got := Complete(tt.total, tt.target); got != tt.complete {
			t.Errorf("Complete(%d, %d) = %v, want %v", tt.total, tt.target, got, tt.complete)
		}
	}
}

func logged(id int64, moment Moment, pos int, period Period, target, value int) Logged {
	return Logged{Habit: Habit{ID: id, Moment: moment, Position: pos, Period: period, Target: target}, Value: value}
}

func TestDayProgress(t *testing.T) {
	tests := []struct {
		name          string
		items         []Logged
		want          float64
		done, total   int
		dayIsComplete bool
	}{
		{"none", nil, 0, 0, 0, false},
		{"mean of day habits", []Logged{
			logged(1, Morning, 0, PeriodDay, 2, 1),
			logged(2, Morning, 1, PeriodDay, 1, 1),
		}, 0.75, 1, 2, false},
		{"exceeding caps at 1", []Logged{logged(1, Morning, 0, PeriodDay, 2, 3)}, 1, 1, 1, true},
		{"period habits are ignored", []Logged{
			logged(1, Morning, 0, PeriodDay, 1, 1),
			logged(2, Day, 0, PeriodWeek, 3, 0),
		}, 1, 1, 1, true},
		{"only period habits", []Logged{logged(2, Day, 0, PeriodWeek, 3, 3)}, 0, 0, 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var items []DayHabit
			for _, l := range tt.items {
				items = append(items, NewDayHabit(l))
			}
			if got := DayProgress(items); math.Abs(got-tt.want) > 1e-9 {
				t.Errorf("DayProgress = %v, want %v", got, tt.want)
			}
			done, total := DayCounts(items)
			if done != tt.done || total != tt.total {
				t.Errorf("DayCounts = %d/%d, want %d/%d", done, total, tt.done, tt.total)
			}
			if got := DayComplete(items); got != tt.dayIsComplete {
				t.Errorf("DayComplete = %v, want %v", got, tt.dayIsComplete)
			}
		})
	}
}

func TestGroupByMoment(t *testing.T) {
	items := []DayHabit{
		NewDayHabit(logged(1, Evening, 0, PeriodDay, 1, 0)),
		NewDayHabit(logged(2, Morning, 0, PeriodDay, 2, 2)), // complete: goes last
		NewDayHabit(logged(3, Morning, 1, PeriodDay, 1, 0)),
		NewDayHabit(logged(4, Morning, 2, PeriodDay, 1, 0)),
	}
	groups := GroupByMoment(items)
	if len(groups) != 2 || groups[0].Moment != Morning || groups[1].Moment != Evening {
		t.Fatalf("groups = %+v (the empty Day moment must be skipped)", groups)
	}
	var ids []int64
	for _, h := range groups[0].Habits {
		ids = append(ids, h.Habit.ID)
	}
	if len(ids) != 3 || ids[0] != 3 || ids[1] != 4 || ids[2] != 2 {
		t.Errorf("morning order = %v, want [3 4 2]", ids)
	}
}

func TestValidateValue(t *testing.T) {
	for _, v := range []int{0, 1, MaxEntryValue} {
		if err := ValidateValue(v); err != nil {
			t.Errorf("ValidateValue(%d) = %v", v, err)
		}
	}
	for _, v := range []int{-1, MaxEntryValue + 1} {
		if err := ValidateValue(v); err == nil {
			t.Errorf("ValidateValue(%d) accepted", v)
		}
	}
}

func TestEditable(t *testing.T) {
	today := time.Date(2026, time.November, 6, 0, 0, 0, 0, time.UTC)
	tests := []struct {
		name   string
		offset int
		want   bool
	}{
		{"tomorrow", 1, false},
		{"today", 0, true},
		{"today - 6", -6, true},
		{"today - 7", -7, false},
		{"long ago", -90, false},
	}
	for _, tt := range tests {
		if got := Editable(today.AddDate(0, 0, tt.offset), today, 7); got != tt.want {
			t.Errorf("%s: Editable = %v, want %v", tt.name, got, tt.want)
		}
	}
}

func TestWeekDays(t *testing.T) {
	// 2026-11-06 is a Friday; its week runs from Monday 2 to Sunday 8.
	days := WeekDays(time.Date(2026, time.November, 6, 0, 0, 0, 0, time.UTC))
	if len(days) != 7 || FormatDay(days[0]) != "2026-11-02" || FormatDay(days[6]) != "2026-11-08" {
		t.Errorf("week = %v", days)
	}
	// A Sunday belongs to the week that ends with it.
	if got := WeekDays(days[6]); FormatDay(got[0]) != "2026-11-02" {
		t.Errorf("Sunday's week starts %s", FormatDay(got[0]))
	}
}

func TestValidateNote(t *testing.T) {
	if err := ValidateNote(strings.Repeat("é", MaxNoteLength)); err != nil {
		t.Errorf("1000 characters: %v", err)
	}
	if err := ValidateNote(strings.Repeat("é", MaxNoteLength+1)); err == nil {
		t.Error("1001 characters accepted")
	}
	if err := ValidateNote(""); err != nil {
		t.Errorf("empty: %v", err)
	}
}

func TestStreak(t *testing.T) {
	today := time.Date(2026, time.November, 6, 0, 0, 0, 0, time.UTC)
	// stats builds one entry per day: kinds are 'c' complete, 'i' incomplete, '-' no habit,
	// the last character being today.
	stats := func(kinds string) []DayStat {
		var out []DayStat
		for i, k := range kinds {
			s := DayStat{Day: today.AddDate(0, 0, i+1-len(kinds))}
			switch k {
			case 'c':
				s.Total, s.Done, s.ProgressSum = 1, 1, 1
			case 'i':
				s.Total = 1
			}
			out = append(out, s)
		}
		return out
	}
	tests := []struct {
		name  string
		kinds string
		want  int
	}{
		{"twelve until yesterday, today incomplete", strings.Repeat("c", 12) + "i", 12},
		{"today complete", strings.Repeat("c", 12) + "c", 13},
		{"a day without habit is skipped", "cc-cc", 4},
		{"an incomplete day ends it", "cccicc", 2},
		{"today without habit", "cc-", 2},
		{"nothing", "iii", 0},
	}
	for _, tt := range tests {
		if got := Streak(stats(tt.kinds), today); got != tt.want {
			t.Errorf("%s: Streak = %d, want %d", tt.name, got, tt.want)
		}
	}
	if got := Streak(nil, today); got != 0 {
		t.Errorf("no stats: Streak = %d", got)
	}
}
