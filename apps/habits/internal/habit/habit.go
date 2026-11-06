// Package habit holds the domain types and the pure rules of the habit tracker:
// the habit model and its validation, weekday masks and civil days. It knows nothing
// about HTTP or SQL.
package habit

import (
	"errors"
	"slices"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

const (
	// MaxNameLength is the longest habit name, in characters.
	MaxNameLength = 60
	// MaxUnitLength is the longest unit, in characters.
	MaxUnitLength = 20
	// MaxTarget is the highest target (and step).
	MaxTarget = 10000
	// maxEmojiRunes bounds one emoji: a flag, a family or a skin-toned sequence is
	// several code points joined into one grapheme.
	maxEmojiRunes = 10
)

const dayLayout = "2006-01-02"

var (
	// ErrNameTaken means another active habit already has this name.
	ErrNameTaken = errors.New("habit name already used by an active habit")
)

// ValidationError is an input the user can fix; Message is shown to them (in French).
type ValidationError struct {
	Message string
}

func (e *ValidationError) Error() string { return e.Message }

// Moment is the part of the day a habit belongs to.
type Moment string

const (
	Morning Moment = "morning"
	Day     Moment = "day"
	Evening Moment = "evening"
)

// Moments lists the moments in display order.
var Moments = []Moment{Morning, Day, Evening}

// Period is the span a habit's target applies to.
type Period string

const (
	PeriodDay   Period = "day"
	PeriodWeek  Period = "week"
	PeriodMonth Period = "month"
	PeriodYear  Period = "year"
)

// Periods lists the known periods.
var Periods = []Period{PeriodDay, PeriodWeek, PeriodMonth, PeriodYear}

// Color is a key of the 8-color palette; the front maps it to actual colors.
type Color string

// Colors lists the palette.
var Colors = []Color{"amber", "violet", "blue", "red", "green", "teal", "slate", "pink"}

// Habit is a habit with the rule in force on the day it was read.
type Habit struct {
	ID         int64
	Name       string
	Emoji      string
	Color      Color
	Moment     Moment
	Position   int
	Unit       string
	Step       int
	Period     Period
	Target     int
	Weekdays   Weekdays
	ArchivedOn *time.Time
}

// Input is what the user provides to create a habit.
type Input struct {
	Name     string
	Emoji    string
	Color    Color
	Moment   Moment
	Unit     string
	Step     int
	Period   Period
	Target   int
	Weekdays Weekdays
}

// Validate checks the input and returns it normalized (name and unit trimmed).
func (in Input) Validate() (Input, error) {
	var err error
	if in.Name, err = ValidateName(in.Name); err != nil {
		return Input{}, err
	}
	if !validEmoji(in.Emoji) {
		return Input{}, invalid("Choisissez un emoji.")
	}
	if !slices.Contains(Colors, in.Color) {
		return Input{}, invalid("Couleur inconnue.")
	}
	if !slices.Contains(Moments, in.Moment) {
		return Input{}, invalid("Moment inconnu.")
	}
	if !slices.Contains(Periods, in.Period) {
		return Input{}, invalid("Période inconnue.")
	}
	if in.Target < 1 || in.Target > MaxTarget {
		return Input{}, invalid("La cible doit être comprise entre 1 et 10 000.")
	}
	if in.Step < 1 || in.Step > MaxTarget {
		return Input{}, invalid("Le pas doit être compris entre 1 et 10 000.")
	}
	in.Unit = strings.TrimSpace(in.Unit)
	if utf8.RuneCountInString(in.Unit) > MaxUnitLength || hasControl(in.Unit) {
		return Input{}, invalid("L'unité doit faire 20 caractères au plus.")
	}
	if in.Weekdays == 0 || in.Weekdays > AllDays {
		return Input{}, invalid("Choisissez au moins un jour.")
	}
	return in, nil
}

func invalid(message string) error { return &ValidationError{Message: message} }

// ValidateName trims the name and checks its length and characters.
func ValidateName(name string) (string, error) {
	name = strings.TrimSpace(name)
	n := utf8.RuneCountInString(name)
	if n < 1 || n > MaxNameLength {
		return "", invalid("Le nom doit faire entre 1 et 60 caractères.")
	}
	if hasControl(name) {
		return "", invalid("Le nom contient des caractères interdits.")
	}
	return name, nil
}

// validEmoji accepts a short run of symbols. Counting graphemes exactly needs a
// Unicode segmentation table the standard library does not have, so this rejects
// what is clearly not an emoji (letters, spaces, control characters) and bounds the length.
func validEmoji(s string) bool {
	n := utf8.RuneCountInString(s)
	if n < 1 || n > maxEmojiRunes {
		return false
	}
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsSpace(r) || unicode.IsControl(r) {
			return false
		}
	}
	return true
}

func hasControl(s string) bool {
	for _, r := range s {
		if unicode.IsControl(r) {
			return true
		}
	}
	return false
}

// Weekdays is a 7-bit mask of the days a habit applies to: bit 0 is Monday and
// bit 6 is Sunday, which is also how the value is stored in the database.
type Weekdays uint8

// AllDays is the default rule: every day of the week.
const AllDays Weekdays = 0b1111111

// Bit returns the position of d in the mask. time.Weekday counts from Sunday = 0,
// the mask from Monday = 0.
func Bit(d time.Weekday) int { return (int(d) + 6) % 7 }

// Has reports whether the mask includes d.
func (w Weekdays) Has(d time.Weekday) bool { return w&(1<<Bit(d)) != 0 }

// ParseDay parses a strict YYYY-MM-DD date into a civil day (midnight UTC).
func ParseDay(s string) (time.Time, error) {
	return time.Parse(dayLayout, s)
}

// FormatDay renders a civil day as YYYY-MM-DD.
func FormatDay(d time.Time) string { return d.Format(dayLayout) }

// Today returns the civil day of now in loc: the server's own time zone plays
// no part, so 23:30 UTC on 30 June is already 1 July in Paris.
func Today(now time.Time, loc *time.Location) time.Time {
	y, m, d := now.In(loc).Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}
