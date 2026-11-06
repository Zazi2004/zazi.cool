package server

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"zazi.cool/habits/internal/habit"
)

// dayHabitJSON, dayJSON and weekDayJSON are the DayHabit, Day and Day.week of the spec (§7).
type dayHabitJSON struct {
	Habit    habitJSON `json:"habit"`
	Value    int       `json:"value"`
	Total    int       `json:"total"`
	Progress float64   `json:"progress"`
	Complete bool      `json:"complete"`
	Pace     *paceJSON `json:"pace"`
}

type paceJSON struct {
	Expected float64 `json:"expected"`
	Diff     int     `json:"diff"`
	Label    string  `json:"label"`
}

type weekDayJSON struct {
	Date     string  `json:"date"`
	Progress float64 `json:"progress"`
}

type groupJSON struct {
	Moment string         `json:"moment"`
	Habits []dayHabitJSON `json:"habits"`
}

type dayJSON struct {
	Date     string        `json:"date"`
	Editable bool          `json:"editable"`
	Progress float64       `json:"progress"`
	Done     int           `json:"done"`
	Total    int           `json:"total"`
	Complete bool          `json:"complete"`
	Streak   int           `json:"streak"`
	Week     []weekDayJSON `json:"week"`
	Groups   []groupJSON   `json:"groups"`
	Note     string        `json:"note"`
}

func toPaceJSON(p *habit.Pace) *paceJSON {
	if p == nil {
		return nil
	}
	return &paceJSON{Expected: p.Expected, Diff: p.Diff, Label: p.Label}
}

// buildDay computes the Day of the spec from what the store read.
func buildDay(day time.Time, editable bool, logged []habit.Logged, stats []habit.DayStat, note string, today time.Time) dayJSON {
	items := make([]habit.DayHabit, 0, len(logged))
	for _, l := range logged {
		items = append(items, habit.NewDayHabit(l, day))
	}
	done, total := habit.DayCounts(items)
	out := dayJSON{
		Date: habit.FormatDay(day), Editable: editable,
		Progress: habit.DayProgress(items), Done: done, Total: total, Complete: habit.DayComplete(items),
		Streak: habit.Streak(stats, today), Week: []weekDayJSON{}, Groups: []groupJSON{}, Note: note,
	}
	progress := make(map[time.Time]float64, len(stats))
	for _, st := range stats {
		progress[st.Day] = st.Progress()
	}
	for _, d := range habit.WeekDays(day) {
		out.Week = append(out.Week, weekDayJSON{Date: habit.FormatDay(d), Progress: progress[d]})
	}
	for _, g := range habit.GroupByMoment(items) {
		gj := groupJSON{Moment: string(g.Moment), Habits: make([]dayHabitJSON, 0, len(g.Habits))}
		for _, it := range g.Habits {
			gj.Habits = append(gj.Habits, dayHabitJSON{
				Habit: toHabitJSON(it.Habit), Value: it.Value, Total: it.Total,
				Progress: it.Progress, Complete: it.Complete, Pace: toPaceJSON(it.Pace),
			})
		}
		out.Groups = append(out.Groups, gj)
	}
	return out
}

// requestedDay parses the {date} of the path: the future is refused (422). When
// writing, a day outside the edit window is refused too (422), before anything is
// changed. It writes the error response and returns false on refusal.
func (h *handler) requestedDay(w http.ResponseWriter, r *http.Request, writing bool) (time.Time, bool) {
	day, err := habit.ParseDay(r.PathValue("date"))
	if err != nil {
		writeError(w, http.StatusUnprocessableEntity, "invalid_day", "Date invalide.")
		return time.Time{}, false
	}
	today := h.today()
	if day.After(today) {
		writeError(w, http.StatusUnprocessableEntity, "day_in_future", "Ce jour n'est pas encore arrivé.")
		return time.Time{}, false
	}
	if writing && !habit.Editable(day, today, h.cfg.EditWindowDays) {
		writeError(w, http.StatusUnprocessableEntity, "day_not_editable", "Ce jour est en lecture seule.")
		return time.Time{}, false
	}
	return day, true
}

func (h *handler) getDay(w http.ResponseWriter, r *http.Request) {
	day, ok := h.requestedDay(w, r, false)
	if !ok {
		return
	}
	h.writeDay(w, r, day, http.StatusOK)
}

func (h *handler) writeDay(w http.ResponseWriter, r *http.Request, day time.Time, status int) {
	ctx := r.Context()
	logged, err := h.cfg.Store.DayHabits(ctx, day)
	if err != nil {
		h.internalError(w, r, err)
		return
	}
	// One read feeds the week strip and the streak: from the earliest habit to the later
	// of the shown week's Sunday and today.
	week := habit.WeekDays(day)
	today := h.today()
	to := week[6]
	if today.After(to) {
		to = today
	}
	stats, err := h.cfg.Store.DayStats(ctx, week[0], to)
	if err != nil {
		h.internalError(w, r, err)
		return
	}
	note, err := h.cfg.Store.DayNote(ctx, day)
	if err != nil {
		h.internalError(w, r, err)
		return
	}
	writeJSON(w, status, buildDay(day, habit.Editable(day, today, h.cfg.EditWindowDays), logged, stats, note, today))
}

func (h *handler) putEntry(w http.ResponseWriter, r *http.Request) {
	day, ok := h.requestedDay(w, r, true)
	if !ok {
		return
	}
	id, err := strconv.ParseInt(r.PathValue("habitId"), 10, 64)
	if err != nil {
		writeError(w, http.StatusNotFound, "not_found", "Ressource introuvable.")
		return
	}
	// A pointer tells a missing value from 0, which deletes the entry.
	var req struct {
		Value *int `json:"value"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	if req.Value == nil {
		writeError(w, http.StatusUnprocessableEntity, "invalid_entry", "La valeur est obligatoire.")
		return
	}
	if err := habit.ValidateValue(*req.Value); err != nil {
		writeError(w, http.StatusUnprocessableEntity, "invalid_entry", err.Error())
		return
	}
	switch err := h.cfg.Store.SetEntry(r.Context(), id, day, *req.Value); {
	case errors.Is(err, habit.ErrNotFound):
		writeError(w, http.StatusNotFound, "not_found", "Cette habitude n'existe pas ce jour-là.")
		return
	case errors.Is(err, habit.ErrDayNotAllowed):
		writeError(w, http.StatusUnprocessableEntity, "day_not_allowed", "Cette habitude n'est pas prévue ce jour-là.")
		return
	case err != nil:
		h.internalError(w, r, err)
		return
	}
	h.writeDay(w, r, day, http.StatusOK)
}

func (h *handler) putNote(w http.ResponseWriter, r *http.Request) {
	day, ok := h.requestedDay(w, r, true)
	if !ok {
		return
	}
	var req struct {
		Text *string `json:"text"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	if req.Text == nil {
		writeError(w, http.StatusUnprocessableEntity, "invalid_note", "Le texte est obligatoire.")
		return
	}
	if err := habit.ValidateNote(*req.Text); err != nil {
		writeError(w, http.StatusUnprocessableEntity, "invalid_note", err.Error())
		return
	}
	if err := h.cfg.Store.SetNote(r.Context(), day, *req.Text); err != nil {
		h.internalError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
