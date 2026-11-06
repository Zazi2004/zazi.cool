package habit

import (
	"math"
	"testing"
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
