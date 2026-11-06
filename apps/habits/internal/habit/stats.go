package habit

import (
	"math"
	"time"
)

// History is everything the statistics of one habit are computed from: its versioned
// rules and every entry ever recorded. The rules are ascending by ValidFrom and the
// first one starts on the day the habit was created.
type History struct {
	// Habit carries the fixed attributes and the rule in force today.
	Habit   Habit
	Rules   []RuleVersion
	Entries map[time.Time]int
}

// Status is how a period ended up.
type Status string

const (
	// StatusSuccess: the target is reached (also for the current period once it is).
	StatusSuccess Status = "success"
	// StatusMissed: a closed period whose target was not reached.
	StatusMissed Status = "missed"
	// StatusCurrent: the ongoing period, target not reached yet.
	StatusCurrent Status = "current"
	// StatusInterrupted: cut short by a change of period (D8); neither a success nor a miss.
	StatusInterrupted Status = "interrupted"
)

// PeriodStat is one period of a habit, counted with the rule of its first day.
type PeriodStat struct {
	Start, End time.Time
	Total      int
	Target     int
	Status     Status
}

// asOf is the last day the habit exists on: today, or the day before it was archived.
func (h History) asOf(today time.Time) time.Time {
	if h.Habit.ArchivedOn != nil && h.Habit.ArchivedOn.Before(today.AddDate(0, 0, 1)) {
		return h.Habit.ArchivedOn.AddDate(0, 0, -1)
	}
	return today
}

// ruleOn returns the rule in force on day; false before the habit was created.
func (h History) ruleOn(day time.Time) (Rule, bool) {
	i := lastBefore(h.Rules, day.AddDate(0, 0, 1))
	if i < 0 {
		return Rule{}, false
	}
	return h.Rules[i].Rule, true
}

// sum adds the entries from first to last, both included.
func (h History) sum(first, last time.Time) int {
	total := 0
	for d := first; !d.After(last); d = d.AddDate(0, 0, 1) {
		total += h.Entries[d]
	}
	return total
}

// Periods lists, oldest first, every period of the habit up to today. A period is
// evaluated with the rule of its first day; the last period of a rule that a change of
// period replaced is cut the day before the new rule and is interrupted. For the `day`
// period, a period is a day the weekdays allow.
func (h History) Periods(today time.Time) []PeriodStat {
	asOf := h.asOf(today)
	var out []PeriodStat
	for i, v := range h.Rules {
		limit := asOf.AddDate(0, 0, 1)
		hasNext := i+1 < len(h.Rules)
		if hasNext && h.Rules[i+1].ValidFrom.Before(limit) {
			limit = h.Rules[i+1].ValidFrom
		}
		for start := PeriodStart(v.Period, v.ValidFrom); start.Before(limit); {
			naturalEnd := PeriodEnd(v.Period, start)
			if v.Period != PeriodDay || v.Weekdays.Has(start.Weekday()) {
				p := PeriodStat{Start: start, End: naturalEnd, Target: v.Target}
				if hasNext && !naturalEnd.Before(h.Rules[i+1].ValidFrom) {
					p.End, p.Status = h.Rules[i+1].ValidFrom.AddDate(0, 0, -1), StatusInterrupted
				}
				p.Total = h.sum(p.Start, p.End)
				if p.Status == "" {
					p.Status = outcome(p, asOf)
				}
				out = append(out, p)
			}
			start = naturalEnd.AddDate(0, 0, 1)
		}
	}
	return out
}

// outcome decides the status of a period that was not cut short.
func outcome(p PeriodStat, asOf time.Time) Status {
	switch {
	case Complete(p.Total, p.Target):
		return StatusSuccess
	case !p.End.Before(asOf):
		return StatusCurrent
	}
	return StatusMissed
}

// Streaks returns the run of successful periods that ends now (the ongoing period
// does not break it while it is not done, an interrupted one is skipped) and the
// longest run ever. periods are oldest first.
func Streaks(periods []PeriodStat) (current, best int) {
	run := 0
	for _, p := range periods {
		switch p.Status {
		case StatusSuccess:
			run++
			best = max(best, run)
		case StatusMissed:
			run = 0
		}
	}
	for i := len(periods) - 1; i >= 0; i-- {
		switch periods[i].Status {
		case StatusSuccess:
			current++
		case StatusMissed:
			return current, best
		}
	}
	return current, best
}

// Rate is the share of successful periods among those that count between from and to
// (the periods overlapping that interval): closed ones, plus the ongoing one when it
// is already a success. Interrupted periods do not count. nil when none counts.
func Rate(periods []PeriodStat, from, to time.Time) *float64 {
	var done, counted int
	for _, p := range periods {
		if p.End.Before(from) || p.Start.After(to) {
			continue
		}
		switch p.Status {
		case StatusSuccess:
			done++
			counted++
		case StatusMissed:
			counted++
		}
	}
	if counted == 0 {
		return nil
	}
	rate := float64(done) / float64(counted)
	return &rate
}

// Summary is the state of a habit in its current period.
type Summary struct {
	Total    int
	Progress float64
	Complete bool
	// Pace is nil for the `day` period.
	Pace *Pace
	// Streak is nil for the `year` period, which has none.
	Streak *int
}

// Summary computes the current period of the habit as of today, and its streak.
func (h History) Summary(today time.Time) Summary {
	asOf := h.asOf(today)
	rule, ok := h.ruleOn(asOf)
	if !ok {
		return Summary{}
	}
	total := h.sum(PeriodStart(rule.Period, asOf), asOf)
	s := Summary{
		Total: total, Progress: Progress(total, rule.Target), Complete: Complete(total, rule.Target),
		Pace: NewPace(rule.Period, asOf, total, rule.Target),
	}
	if rule.Period != PeriodYear {
		current, _ := Streaks(h.Periods(today))
		s.Streak = &current
	}
	return s
}

// Cell is one day of the heatmap.
type Cell struct {
	Day   time.Time
	Value int
	// Level is 0 to 4; Allowed is false when the habit does not apply that day.
	Level   int
	Allowed bool
}

// Heatmap returns every day of the year. The level is the day's progress for a `day`
// habit, and for the others the value relative to the highest value of the year, in
// quarters; 0 only when nothing was entered.
func (h History) Heatmap(year int) []Cell {
	first := time.Date(year, time.January, 1, 0, 0, 0, 0, time.UTC)
	last := time.Date(year, time.December, 31, 0, 0, 0, 0, time.UTC)
	peak := 0
	for d := first; !d.After(last); d = d.AddDate(0, 0, 1) {
		peak = max(peak, h.Entries[d])
	}
	var cells []Cell
	for d := first; !d.After(last); d = d.AddDate(0, 0, 1) {
		rule, ok := h.ruleOn(d)
		archived := h.Habit.ArchivedOn != nil && !d.Before(*h.Habit.ArchivedOn)
		if !ok || archived || !rule.Weekdays.Has(d.Weekday()) {
			cells = append(cells, Cell{Day: d})
			continue
		}
		value := h.Entries[d]
		share := float64(value) / float64(max(peak, 1))
		if rule.Period == PeriodDay {
			share = Progress(value, rule.Target)
		}
		cells = append(cells, Cell{Day: d, Value: value, Level: level(share), Allowed: true})
	}
	return cells
}

// level maps a share in [0, 1] to 0..4: 0 only for nothing, then quarters.
func level(share float64) int {
	if share <= 0 {
		return 0
	}
	return min(4, max(1, int(math.Ceil(share*4))))
}
