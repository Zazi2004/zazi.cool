package habit

import (
	"sort"
	"strconv"
	"time"
)

// ScheduleVersion is the weekday mask in force from ValidFrom on.
type ScheduleVersion struct {
	ValidFrom time.Time
	Weekdays  Weekdays
}

// History is everything the statistics need to know about one habit: when it
// existed, which weekdays applied over time (NF5) and the days it was done.
type History struct {
	ID         int64
	Name       string
	CreatedOn  time.Time
	ArchivedOn *time.Time
	// Schedules is sorted by ValidFrom, oldest first.
	Schedules []ScheduleVersion
	// Done holds the civil days (midnight UTC) on which the habit was checked.
	Done map[time.Time]bool
}

// State is how one day counts for one habit.
type State int

const (
	// NotApplicable: the habit did not exist, was archived or was not scheduled.
	NotApplicable State = iota
	// StateDone: the habit applied and was checked.
	StateDone
	// StateMissed: the habit applied, the day is over or ongoing, and it was not checked.
	StateMissed
	// StateFuture: the habit applies but the day has not come yet.
	StateFuture
)

// Applies reports whether the habit was concerned by day: it existed, was not yet
// archived, and the schedule in force that day included its weekday (F10).
func (h History) Applies(day time.Time) bool {
	if day.Before(h.CreatedOn) || (h.ArchivedOn != nil && !day.Before(*h.ArchivedOn)) {
		return false
	}
	// Latest version that started on or before day.
	i := sort.Search(len(h.Schedules), func(i int) bool { return h.Schedules[i].ValidFrom.After(day) })
	return i > 0 && h.Schedules[i-1].Weekdays.Has(day.Weekday())
}

// StateOn classifies day for this habit, given the current civil day.
func (h History) StateOn(day, today time.Time) State {
	switch {
	case !h.Applies(day):
		return NotApplicable
	case day.After(today):
		return StateFuture
	case h.Done[day]:
		return StateDone
	default:
		return StateMissed
	}
}

// Tally counts done days out of the days that were due.
type Tally struct {
	Done, Due int
}

// Add returns the sum of two tallies.
func (t Tally) Add(o Tally) Tally { return Tally{t.Done + o.Done, t.Due + o.Due} }

// Percent is the success rate rounded to the nearest percent; ok is false when
// nothing was due, so there is no rate to show.
func (t Tally) Percent() (pct int, ok bool) {
	if t.Due == 0 {
		return 0, false
	}
	return (t.Done*100 + t.Due/2) / t.Due, true
}

// String renders the rate as "67 %", or "—" when nothing was due.
func (t Tally) String() string {
	pct, ok := t.Percent()
	if !ok {
		return "—"
	}
	return strconv.Itoa(pct) + " %"
}

// Tally counts the days of [from, to] that were due, up to today included: future
// days and days before the habit existed do not lower the rate (F10).
func (h History) Tally(from, to, today time.Time) Tally {
	if to.After(today) {
		to = today
	}
	var t Tally
	for d := from; !d.After(to); d = d.AddDate(0, 0, 1) {
		switch h.StateOn(d, today) {
		case StateDone:
			t.Done++
			t.Due++
		case StateMissed:
			t.Due++
		}
	}
	return t
}

// DayTally counts, across habits, how many were due on day and how many were done.
func DayTally(hs []History, day, today time.Time) Tally {
	var t Tally
	for _, h := range hs {
		t = t.Add(h.Tally(day, day, today))
	}
	return t
}

// Streak is the number of consecutive due days checked, ending today (F11). Days the
// habit does not apply to are skipped, not counted and do not break the streak; an
// unchecked today is not a miss yet, so the streak of yesterday stands. An archived
// habit has no streak.
func (h History) Streak(today time.Time) int {
	if h.ArchivedOn != nil && !today.Before(*h.ArchivedOn) {
		return 0
	}
	day := today
	if h.Applies(day) && !h.Done[day] {
		day = day.AddDate(0, 0, -1)
	}
	n := 0
	for ; !day.Before(h.CreatedOn); day = day.AddDate(0, 0, -1) {
		if !h.Applies(day) {
			continue
		}
		if !h.Done[day] {
			break
		}
		n++
	}
	return n
}

// Monday returns the Monday of the week containing day.
func Monday(day time.Time) time.Time {
	return day.AddDate(0, 0, -Bit(day.Weekday()))
}
