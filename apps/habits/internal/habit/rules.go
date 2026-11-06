package habit

import (
	"slices"
	"time"
)

// Rule is the versioned part of a habit: what changes how the past is counted.
type Rule struct {
	Period   Period
	Target   int
	Weekdays Weekdays
}

// RuleVersion is a Rule effective from a civil day on.
type RuleVersion struct {
	ValidFrom time.Time
	Rule
}

// RuleChange says what to write to apply a new rule on a day.
type RuleChange struct {
	// None is true when the rule in force that day already is the new one: nothing to write.
	None bool
	// From is the first day of the new version: the start of the current period, taken
	// with the new period. Every stored version from this day on is replaced.
	From time.Time
	// Insert is false when the version before From already equals the rule, so the
	// replaced versions are simply dropped (an edit undone in the same period).
	Insert bool
}

// PlanRuleChange decides how rules (ascending by ValidFrom) change when next becomes the
// rule on today. A version for a past period is never touched (spec §4): the past is
// still counted with the rule that was in force then.
func PlanRuleChange(rules []RuleVersion, next Rule, today time.Time) RuleChange {
	if i := lastBefore(rules, today.AddDate(0, 0, 1)); i >= 0 && rules[i].Rule == next {
		return RuleChange{None: true}
	}
	from := PeriodStart(next.Period, today)
	i := lastBefore(rules, from)
	return RuleChange{From: from, Insert: i < 0 || rules[i].Rule != next}
}

// lastBefore returns the index of the last rule that starts before day, or -1.
func lastBefore(rules []RuleVersion, day time.Time) int {
	i := slices.IndexFunc(rules, func(v RuleVersion) bool { return !v.ValidFrom.Before(day) })
	if i < 0 {
		return len(rules) - 1
	}
	return i - 1
}
