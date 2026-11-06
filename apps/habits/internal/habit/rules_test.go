package habit

import (
	"testing"
	"time"
)

func d(s string) time.Time {
	day, err := ParseDay(s)
	if err != nil {
		panic(err)
	}
	return day
}

func TestPlanRuleChange(t *testing.T) {
	week3 := Rule{PeriodWeek, 3, AllDays}
	week4 := Rule{PeriodWeek, 4, AllDays}
	month3 := Rule{PeriodMonth, 3, AllDays}
	tests := []struct {
		name  string
		rules []RuleVersion
		next  Rule
		today string
		want  RuleChange
	}{
		{
			// CA4: Thursday 5 November, the week starts on Monday 2 November.
			name:  "target change starts at the current week",
			rules: []RuleVersion{{d("2026-10-01"), week3}},
			next:  week4, today: "2026-11-05",
			want: RuleChange{From: d("2026-11-02"), Insert: true},
		},
		{
			// CA5: week to month on 6 November: the version starts on 1 November.
			name:  "period change starts at the new period",
			rules: []RuleVersion{{d("2026-10-01"), week3}},
			next:  month3, today: "2026-11-06",
			want: RuleChange{From: d("2026-11-01"), Insert: true},
		},
		{
			name:  "second edit in the same period replaces the version",
			rules: []RuleVersion{{d("2026-10-01"), week3}, {d("2026-11-02"), week4}},
			next:  Rule{PeriodWeek, 5, AllDays}, today: "2026-11-06",
			want: RuleChange{From: d("2026-11-02"), Insert: true},
		},
		{
			name:  "edit undone in the same period drops the version",
			rules: []RuleVersion{{d("2026-10-01"), week3}, {d("2026-11-02"), week4}},
			next:  week3, today: "2026-11-06",
			want: RuleChange{From: d("2026-11-02"), Insert: false},
		},
		{
			name:  "same rule changes nothing",
			rules: []RuleVersion{{d("2026-10-01"), week3}, {d("2026-11-02"), week4}},
			next:  week4, today: "2026-11-06",
			want: RuleChange{None: true},
		},
		{
			name:  "habit created this week keeps a single version",
			rules: []RuleVersion{{d("2026-11-04"), week3}},
			next:  week4, today: "2026-11-05",
			want: RuleChange{From: d("2026-11-02"), Insert: true},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := PlanRuleChange(tt.rules, tt.next, d(tt.today)); got != tt.want {
				t.Errorf("PlanRuleChange = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestPatchApply(t *testing.T) {
	h := Habit{Name: "Course", Emoji: "🏃", Color: "green", Moment: Evening, Step: 1, Period: PeriodWeek, Target: 3, Weekdays: AllDays}
	target := 4
	name := "Footing"
	in := Patch{Target: &target, Name: &name}.Apply(h)
	if in.Target != 4 || in.Name != "Footing" || in.Period != PeriodWeek || in.Emoji != "🏃" {
		t.Errorf("Apply = %+v", in)
	}
}

func TestValidateOrder(t *testing.T) {
	current := []int64{1, 2, 3}
	tests := map[string]struct {
		ids []int64
		ok  bool
	}{
		"reordered": {[]int64{3, 1, 2}, true},
		"missing":   {[]int64{1, 2}, false},
		"extra":     {[]int64{1, 2, 3, 4}, false},
		"unknown":   {[]int64{1, 2, 9}, false},
		"duplicate": {[]int64{1, 1, 2}, false},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			if err := ValidateOrder(current, tt.ids); (err == nil) != tt.ok {
				t.Errorf("ValidateOrder(%v) = %v, want ok=%v", tt.ids, err, tt.ok)
			}
		})
	}
}
