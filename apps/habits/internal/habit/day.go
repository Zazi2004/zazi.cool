package habit

import (
	"errors"
	"slices"
)

// MaxEntryValue bounds one day's value, far above any realistic count.
const MaxEntryValue = 100000

var (
	// ErrNotFound means no habit applies on the day: unknown, archived or not yet created.
	ErrNotFound = errors.New("habit not found on this day")
	// ErrDayNotAllowed means the habit's weekdays exclude the day.
	ErrDayNotAllowed = errors.New("habit not allowed on this weekday")
)

// Logged is a habit with what was entered for it on a day.
type Logged struct {
	Habit Habit
	Value int
}

// DayHabit is a habit on a day with its computed progress.
type DayHabit struct {
	Habit    Habit
	Value    int
	Total    int
	Progress float64
	Complete bool
}

// MomentGroup is the habits of one moment, in display order.
type MomentGroup struct {
	Moment Moment
	Habits []DayHabit
}

// Progress is the share of the target reached, capped at 1 (the target may be exceeded).
func Progress(total, target int) float64 {
	if target <= 0 {
		return 0
	}
	return min(1, float64(total)/float64(target))
}

// Complete reports whether the target is reached.
func Complete(total, target int) bool { return target > 0 && total >= target }

// NewDayHabit computes the progress of a habit from its day value. Only the `day`
// period is handled for now: its total is the value of the day.
func NewDayHabit(l Logged) DayHabit {
	total := l.Value
	return DayHabit{
		Habit: l.Habit, Value: l.Value, Total: total,
		Progress: Progress(total, l.Habit.Target),
		Complete: Complete(total, l.Habit.Target),
	}
}

// DayProgress is the mean progress of the `day` habits (the big ring); 0 without any.
func DayProgress(items []DayHabit) float64 {
	var sum float64
	var n int
	for _, it := range items {
		if it.Habit.Period != PeriodDay {
			continue
		}
		sum += it.Progress
		n++
	}
	if n == 0 {
		return 0
	}
	return sum / float64(n)
}

// DayCounts returns how many `day` habits are complete and how many there are.
func DayCounts(items []DayHabit) (done, total int) {
	for _, it := range items {
		if it.Habit.Period != PeriodDay {
			continue
		}
		total++
		if it.Complete {
			done++
		}
	}
	return done, total
}

// DayComplete reports whether every `day` habit is complete; false without any.
func DayComplete(items []DayHabit) bool {
	done, total := DayCounts(items)
	return total > 0 && done == total
}

// GroupByMoment groups habits by moment in display order, skipping empty moments. Within
// a group the manual order is kept and completed habits go last.
func GroupByMoment(items []DayHabit) []MomentGroup {
	groups := []MomentGroup{}
	for _, m := range Moments {
		var in []DayHabit
		for _, it := range items {
			if it.Habit.Moment == m {
				in = append(in, it)
			}
		}
		if len(in) == 0 {
			continue
		}
		slices.SortStableFunc(in, func(a, b DayHabit) int {
			return a.Habit.Position - b.Habit.Position
		})
		slices.SortStableFunc(in, func(a, b DayHabit) int {
			return boolInt(a.Complete) - boolInt(b.Complete)
		})
		groups = append(groups, MomentGroup{Moment: m, Habits: in})
	}
	return groups
}

func boolInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

// ValidateValue checks a value to record: 0 deletes the entry.
func ValidateValue(v int) error {
	if v < 0 || v > MaxEntryValue {
		return invalid("La valeur doit être comprise entre 0 et 100 000.")
	}
	return nil
}
