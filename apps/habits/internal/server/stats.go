package server

import (
	"net/http"
	"strconv"
	"time"

	"zazi.cool/habits/internal/habit"
)

const monthLayout = "2006-01"

// streakHorizon is how far back the week view looks for the streaks. The history is
// small (a few thousand rows over the years), so one wide read beats a second query.
var streakHorizon = time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC)

var shortWeekdays = [...]string{"lun", "mar", "mer", "jeu", "ven", "sam", "dim"}

// redirectToCurrent sends /week, /month and /year to the current period.
func (h *handler) redirectToCurrent(prefix string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		today := h.today()
		var target string
		switch prefix {
		case "/week/":
			target = habit.FormatDay(today)
		case "/month/":
			target = today.Format(monthLayout)
		default:
			target = strconv.Itoa(today.Year())
		}
		http.Redirect(w, r, prefix+target, http.StatusSeeOther)
	})
}

// habitRate is one row of the per-habit rates of a period.
type habitRate struct {
	Name   string
	Rate   string
	Done   int
	Due    int
	Streak int
}

// level maps a day's rate to a class the stylesheet can colour (T6): "none" when
// nothing was due, else 0 to 4 by quarters.
func level(t habit.Tally) string {
	pct, ok := t.Percent()
	switch {
	case !ok:
		return "none"
	case pct == 0:
		return "0"
	case pct <= 25:
		return "1"
	case pct <= 50:
		return "2"
	case pct <= 75:
		return "3"
	default:
		return "4"
	}
}

// applicable keeps the habits concerned by at least one day of [from, to].
func applicable(hs []habit.History, from, to time.Time) []habit.History {
	var out []habit.History
	for _, h := range hs {
		for d := from; !d.After(to); d = d.AddDate(0, 0, 1) {
			if h.Applies(d) {
				out = append(out, h)
				break
			}
		}
	}
	return out
}

func rates(hs []habit.History, from, to, today time.Time) (rows []habitRate, total habit.Tally) {
	for _, h := range hs {
		t := h.Tally(from, to, today)
		total = total.Add(t)
		rows = append(rows, habitRate{Name: h.Name, Rate: t.String(), Done: t.Done, Due: t.Due, Streak: h.Streak(today)})
	}
	return rows, total
}

type weekCell struct {
	Date  string
	Label string
	Class string
}

type weekRow struct {
	habitRate
	Cells []weekCell
}

type weekPage struct {
	Label              string
	PrevHref, NextHref string
	Days               []weekCell
	Rows               []weekRow
	Total              string
}

// week shows the Monday-to-Sunday grid of the week containing the date (F7).
func (h *handler) week(w http.ResponseWriter, r *http.Request) {
	date, err := habit.ParseDay(r.PathValue("date"))
	if err != nil {
		http.Error(w, "Date invalide (format AAAA-MM-JJ).", http.StatusBadRequest)
		return
	}
	today := h.today()
	monday := habit.Monday(date)
	sunday := monday.AddDate(0, 0, 6)

	from := streakHorizon
	if monday.Before(from) {
		from = monday
	}
	to := sunday
	if today.After(to) {
		to = today // the streaks end today, whichever week is shown
	}
	hist, err := h.cfg.Store.History(r.Context(), from, to)
	if err != nil {
		h.internalError(w, r, err)
		return
	}
	hist = applicable(hist, monday, sunday)

	page := weekPage{
		Label:    "Semaine du " + frenchDate(monday) + " au " + frenchDate(sunday),
		PrevHref: "/week/" + habit.FormatDay(monday.AddDate(0, 0, -7)),
	}
	if next := monday.AddDate(0, 0, 7); !next.After(today) {
		page.NextHref = "/week/" + habit.FormatDay(next)
	}
	for i := range 7 {
		d := monday.AddDate(0, 0, i)
		page.Days = append(page.Days, weekCell{
			Date: habit.FormatDay(d), Label: shortWeekdays[i] + " " + strconv.Itoa(d.Day()),
		})
	}
	rows, total := rates(hist, monday, sunday, today)
	for i, hh := range hist {
		row := weekRow{habitRate: rows[i]}
		for d := monday; !d.After(sunday); d = d.AddDate(0, 0, 1) {
			c := weekCell{Date: habit.FormatDay(d)}
			switch hh.StateOn(d, today) {
			case habit.StateDone:
				c.Label, c.Class = "fait", "done"
			case habit.StateMissed:
				c.Label, c.Class = "pas fait", "missed"
			case habit.StateFuture:
				c.Label, c.Class = "à venir", "future"
			default:
				c.Label, c.Class = "non concerné", "na"
			}
			row.Cells = append(row.Cells, c)
		}
		page.Rows = append(page.Rows, row)
	}
	page.Total = total.String()
	h.render(w, r, http.StatusOK, "week", page)
}

// calCell is one day of a calendar grid. A blank cell pads the week rows.
type calCell struct {
	Blank bool
	Day   int
	Date  string
	Rate  string
	Level string
	Href  string
}

// calendar lays [from, to] out in rows of Monday to Sunday. With links, each day
// points to its day page.
func calendar(hs []habit.History, from, to, today time.Time, links bool) [][]calCell {
	var weeks [][]calCell
	row := make([]calCell, 0, 7)
	for range habit.Bit(from.Weekday()) {
		row = append(row, calCell{Blank: true})
	}
	for d := from; !d.After(to); d = d.AddDate(0, 0, 1) {
		t := habit.DayTally(hs, d, today)
		c := calCell{Day: d.Day(), Date: frenchDate(d), Rate: t.String(), Level: level(t)}
		if links {
			c.Href = dayHref(d, today)
		}
		row = append(row, c)
		if len(row) == 7 {
			weeks = append(weeks, row)
			row = make([]calCell, 0, 7)
		}
	}
	if len(row) > 0 {
		for len(row) < 7 {
			row = append(row, calCell{Blank: true})
		}
		weeks = append(weeks, row)
	}
	return weeks
}

type monthPage struct {
	Label              string
	PrevHref, NextHref string
	Weekdays           [7]string
	Weeks              [][]calCell
	Rows               []habitRate
	Total              string
}

// month shows the calendar of a month, each day coloured by its rate (F8).
func (h *handler) month(w http.ResponseWriter, r *http.Request) {
	first, err := time.Parse(monthLayout, r.PathValue("month"))
	if err != nil {
		http.Error(w, "Mois invalide (format AAAA-MM).", http.StatusBadRequest)
		return
	}
	today := h.today()
	last := first.AddDate(0, 1, -1)
	hist, err := h.cfg.Store.History(r.Context(), first, last)
	if err != nil {
		h.internalError(w, r, err)
		return
	}
	hist = applicable(hist, first, last)

	page := monthPage{
		Label:    frenchMonths[first.Month()-1] + " " + strconv.Itoa(first.Year()),
		PrevHref: "/month/" + first.AddDate(0, -1, 0).Format(monthLayout),
		Weekdays: shortWeekdays,
		Weeks:    calendar(hist, first, last, today, true),
	}
	if next := first.AddDate(0, 1, 0); !next.After(today) {
		page.NextHref = "/month/" + next.Format(monthLayout)
	}
	rows, total := rates(hist, first, last, today)
	page.Rows, page.Total = rows, total.String()
	h.render(w, r, http.StatusOK, "month", page)
}

type monthRate struct {
	Label string
	Href  string
	Rate  string
}

type yearPage struct {
	Label              string
	PrevHref, NextHref string
	Months             []monthRate
	Weekdays           [7]string
	Weeks              [][]calCell
	Rows               []habitRate
	Total              string
}

// year shows the rate of each month, a heatmap of the days and the rate per habit (F9).
func (h *handler) year(w http.ResponseWriter, r *http.Request) {
	y, err := time.Parse("2006", r.PathValue("year"))
	if err != nil {
		http.Error(w, "Année invalide (format AAAA).", http.StatusBadRequest)
		return
	}
	today := h.today()
	first := y
	last := time.Date(y.Year(), time.December, 31, 0, 0, 0, 0, time.UTC)
	hist, err := h.cfg.Store.History(r.Context(), first, last)
	if err != nil {
		h.internalError(w, r, err)
		return
	}
	hist = applicable(hist, first, last)

	page := yearPage{
		Label:    strconv.Itoa(y.Year()),
		PrevHref: "/year/" + strconv.Itoa(y.Year()-1),
		Weekdays: shortWeekdays,
		Weeks:    calendar(hist, first, last, today, false),
	}
	if y.Year() < today.Year() {
		page.NextHref = "/year/" + strconv.Itoa(y.Year()+1)
	}
	for m := first; m.Year() == y.Year(); m = m.AddDate(0, 1, 0) {
		var t habit.Tally
		for _, hh := range hist {
			t = t.Add(hh.Tally(m, m.AddDate(0, 1, -1), today))
		}
		page.Months = append(page.Months, monthRate{
			Label: frenchMonths[m.Month()-1], Href: "/month/" + m.Format(monthLayout), Rate: t.String(),
		})
	}
	rows, total := rates(hist, first, last, today)
	page.Rows, page.Total = rows, total.String()
	h.render(w, r, http.StatusOK, "year", page)
}
