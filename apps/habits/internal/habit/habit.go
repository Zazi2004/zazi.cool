// Package habit holds the domain types and the pure rules of the habit tracker:
// weekday masks, civil days and the edit window. It knows nothing about HTTP or SQL.
package habit

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

// MaxNameLength is the longest habit name, in characters (F1).
const MaxNameLength = 60

const dayLayout = "2006-01-02"

var (
	// ErrNotFound means the habit does not exist.
	ErrNotFound = errors.New("habit not found")
	// ErrNameTaken means another active habit already has this name.
	ErrNameTaken = errors.New("habit name already used by an active habit")
	// ErrNotScheduled means the habit does not apply to the requested day.
	ErrNotScheduled = errors.New("habit not scheduled on this day")
)

// Habit is a habit with the weekdays that apply today.
type Habit struct {
	ID         int64
	Name       string
	CreatedOn  time.Time
	ArchivedOn *time.Time
	Weekdays   Weekdays
}

// Archived reports whether the habit has been archived.
func (h Habit) Archived() bool { return h.ArchivedOn != nil }

// DayItem is a habit that applies to one day, with its state on that day.
type DayItem struct {
	HabitID int64
	Name    string
	Done    bool
}

// Weekdays is a 7-bit mask of the days a habit applies to: bit 0 is Monday and
// bit 6 is Sunday, which is also how the value is stored in the database.
type Weekdays uint8

// AllDays is the default schedule: every day of the week.
const AllDays Weekdays = 0b1111111

// Bit returns the position of d in the mask. time.Weekday counts from Sunday = 0,
// the mask from Monday = 0.
func Bit(d time.Weekday) int { return (int(d) + 6) % 7 }

// Has reports whether the mask includes d.
func (w Weekdays) Has(d time.Weekday) bool { return w&(1<<Bit(d)) != 0 }

// ParseWeekdays builds a mask from ISO weekday numbers, "1" (Monday) to "7" (Sunday).
// At least one day is required.
func ParseWeekdays(values []string) (Weekdays, error) {
	var w Weekdays
	for _, v := range values {
		n, err := strconv.Atoi(v)
		if err != nil || n < 1 || n > 7 {
			return 0, fmt.Errorf("invalid weekday %q", v)
		}
		w |= 1 << (n - 1)
	}
	if w == 0 {
		return 0, errors.New("at least one weekday is required")
	}
	return w, nil
}

// ValidateName trims the name and checks its length and characters (F1).
func ValidateName(name string) (string, error) {
	name = strings.TrimSpace(name)
	n := utf8.RuneCountInString(name)
	if n < 1 || n > MaxNameLength {
		return "", fmt.Errorf("name must be 1 to %d characters", MaxNameLength)
	}
	for _, r := range name {
		if unicode.IsControl(r) {
			return "", errors.New("name contains control characters")
		}
	}
	return name, nil
}

// ParseDay parses a strict YYYY-MM-DD date into a civil day (midnight UTC).
func ParseDay(s string) (time.Time, error) {
	return time.Parse(dayLayout, s)
}

// FormatDay renders a civil day as YYYY-MM-DD.
func FormatDay(d time.Time) string { return d.Format(dayLayout) }

// Today returns the civil day of now in loc (F6): the server's own time zone plays
// no part, so 23:30 UTC on 30 June is already 1 July in Paris.
func Today(now time.Time, loc *time.Location) time.Time {
	y, m, d := now.In(loc).Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}

// Editable reports whether day can be checked or unchecked: not in the future, and
// at most windowDays days before today (F5).
func Editable(day, today time.Time, windowDays int) bool {
	return !day.After(today) && !day.Before(today.AddDate(0, 0, -windowDays))
}
