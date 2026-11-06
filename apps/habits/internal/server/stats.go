package server

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"zazi.cool/habits/internal/habit"
)

// summaryJSON is the HabitSummary of the spec (§7); statsJSON, its HabitStats, embeds it
// (encoding/json flattens an embedded struct into the parent object).
type summaryJSON struct {
	Habit    habitJSON `json:"habit"`
	Total    int       `json:"total"`
	Progress float64   `json:"progress"`
	Complete bool      `json:"complete"`
	Pace     *paceJSON `json:"pace"`
	Streak   *int      `json:"streak"`
}

type heatDayJSON struct {
	Date  string `json:"date"`
	Value int    `json:"value"`
	Level *int   `json:"level"`
}

type periodJSON struct {
	Start  string `json:"start"`
	End    string `json:"end"`
	Total  int    `json:"total"`
	Target int    `json:"target"`
	Status string `json:"status"`
}

type statsJSON struct {
	summaryJSON
	Year       int           `json:"year"`
	BestStreak *int          `json:"bestStreak"`
	Rate30     *float64      `json:"rate30"`
	RateYear   *float64      `json:"rateYear"`
	Days       []heatDayJSON `json:"days"`
	Periods    []periodJSON  `json:"periods"`
}

func toSummaryJSON(hist habit.History, today time.Time) summaryJSON {
	s := hist.Summary(today)
	return summaryJSON{
		Habit: toHabitJSON(hist.Habit), Total: s.Total, Progress: s.Progress, Complete: s.Complete,
		Pace: toPaceJSON(s.Pace), Streak: s.Streak,
	}
}

// toStatsJSON computes the HabitStats of the year. Periods come newest first, the
// order the front lists them in; the `day` period has none (the heatmap covers it).
func toStatsJSON(hist habit.History, year int, today time.Time) statsJSON {
	out := statsJSON{
		summaryJSON: toSummaryJSON(hist, today), Year: year,
		Days: []heatDayJSON{}, Periods: []periodJSON{},
	}
	periods := hist.Periods(today)
	if hist.Habit.Period != habit.PeriodYear {
		_, best := habit.Streaks(periods)
		out.BestStreak = &best
	}
	asOf := today
	if hist.Habit.ArchivedOn != nil && hist.Habit.ArchivedOn.Before(today.AddDate(0, 0, 1)) {
		asOf = hist.Habit.ArchivedOn.AddDate(0, 0, -1)
	}
	first := time.Date(year, time.January, 1, 0, 0, 0, 0, time.UTC)
	last := time.Date(year, time.December, 31, 0, 0, 0, 0, time.UTC)
	out.Rate30 = habit.Rate(periods, asOf.AddDate(0, 0, -29), asOf)
	out.RateYear = habit.Rate(periods, first, last)

	for _, c := range hist.Heatmap(year) {
		cell := heatDayJSON{Date: habit.FormatDay(c.Day), Value: c.Value}
		if c.Allowed {
			level := c.Level
			cell.Level = &level
		}
		out.Days = append(out.Days, cell)
	}
	for i := len(periods) - 1; i >= 0; i-- {
		p := periods[i]
		if hist.Habit.Period == habit.PeriodDay || p.End.Before(first) || p.Start.After(last) {
			continue
		}
		out.Periods = append(out.Periods, periodJSON{
			Start: habit.FormatDay(p.Start), End: habit.FormatDay(p.End),
			Total: p.Total, Target: p.Target, Status: string(p.Status),
		})
	}
	return out
}

func (h *handler) getStats(w http.ResponseWriter, r *http.Request) {
	today := h.today()
	histories, err := h.cfg.Store.ActiveHistories(r.Context(), today)
	if err != nil {
		h.internalError(w, r, err)
		return
	}
	out := struct {
		Habits []summaryJSON `json:"habits"`
	}{Habits: make([]summaryJSON, 0, len(histories))}
	for _, hist := range histories {
		out.Habits = append(out.Habits, toSummaryJSON(hist, today))
	}
	writeJSON(w, http.StatusOK, out)
}

func (h *handler) getHabitStats(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusNotFound, "not_found", "Ressource introuvable.")
		return
	}
	today := h.today()
	year := today.Year()
	if q := r.URL.Query().Get("year"); q != "" {
		year, err = strconv.Atoi(q)
		if err != nil || year < 2000 || year > today.Year() {
			writeError(w, http.StatusUnprocessableEntity, "invalid_year", "Année invalide.")
			return
		}
	}
	hist, err := h.cfg.Store.HabitHistory(r.Context(), id, today)
	if errors.Is(err, habit.ErrNotFound) {
		writeError(w, http.StatusNotFound, "not_found", "Cette habitude n'existe pas.")
		return
	}
	if err != nil {
		h.internalError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, toStatsJSON(hist, year, today))
}
