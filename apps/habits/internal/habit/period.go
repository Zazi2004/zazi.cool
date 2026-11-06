package habit

import (
	"fmt"
	"math"
	"time"
)

// PeriodStart returns the first civil day of the period of d: the day itself, the Monday
// of its week, the first of its month or of its year.
func PeriodStart(p Period, d time.Time) time.Time {
	switch p {
	case PeriodWeek:
		return d.AddDate(0, 0, -Bit(d.Weekday()))
	case PeriodMonth:
		return time.Date(d.Year(), d.Month(), 1, 0, 0, 0, 0, time.UTC)
	case PeriodYear:
		return time.Date(d.Year(), time.January, 1, 0, 0, 0, 0, time.UTC)
	}
	return d
}

// PeriodDays returns how many days the period of d lasts (366 in a leap year).
func PeriodDays(p Period, d time.Time) int {
	start := PeriodStart(p, d)
	switch p {
	case PeriodWeek:
		return 7
	case PeriodMonth:
		return daysBetween(start, start.AddDate(0, 1, 0))
	case PeriodYear:
		return daysBetween(start, start.AddDate(1, 0, 0))
	}
	return 1
}

// daysBetween counts the civil days from a (included) to b (excluded); both are at
// midnight UTC, so the division is exact.
func daysBetween(a, b time.Time) int { return int(b.Sub(a).Hours() / 24) }

// Pace compares the total of a week, month or year habit with where it should be on d.
type Pace struct {
	// Expected is target × elapsed days (d included) ÷ days of the period.
	Expected float64
	// Diff is total − Expected, rounded to the nearest integer.
	Diff  int
	Label string
}

// NewPace computes the pace of a habit on d; nil for the `day` period, which has none.
func NewPace(p Period, d time.Time, total, target int) *Pace {
	if p == PeriodDay {
		return nil
	}
	elapsed := daysBetween(PeriodStart(p, d), d) + 1
	expected := float64(target) * float64(elapsed) / float64(PeriodDays(p, d))
	diff := int(math.Round(float64(total) - expected))
	pace := &Pace{Expected: expected, Diff: diff}
	switch {
	case Complete(total, target):
		pace.Label = "objectif atteint"
	case diff == 0:
		pace.Label = "dans le rythme"
	case diff > 0:
		pace.Label = fmt.Sprintf("%d d'avance", diff)
	default:
		pace.Label = fmt.Sprintf("%d de retard", -diff)
	}
	return pace
}
