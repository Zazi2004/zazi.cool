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
	Expected int    `json:"expected"`
	Diff     int    `json:"diff"`
	Label    string `json:"label"`
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

// buildDay computes the Day of the spec from what the store read. Streak, week strip,
// pace and note belong to later tickets: they are neutral for now.
func buildDay(day time.Time, logged []habit.Logged) dayJSON {
	items := make([]habit.DayHabit, 0, len(logged))
	for _, l := range logged {
		items = append(items, habit.NewDayHabit(l))
	}
	done, total := habit.DayCounts(items)
	out := dayJSON{
		Date: habit.FormatDay(day), Editable: true,
		Progress: habit.DayProgress(items), Done: done, Total: total, Complete: habit.DayComplete(items),
		Week: []weekDayJSON{}, Groups: []groupJSON{},
	}
	for _, g := range habit.GroupByMoment(items) {
		gj := groupJSON{Moment: string(g.Moment), Habits: make([]dayHabitJSON, 0, len(g.Habits))}
		for _, it := range g.Habits {
			gj.Habits = append(gj.Habits, dayHabitJSON{
				Habit: toHabitJSON(it.Habit), Value: it.Value, Total: it.Total,
				Progress: it.Progress, Complete: it.Complete,
			})
		}
		out.Groups = append(out.Groups, gj)
	}
	return out
}

// requestedDay parses the {date} of the path. Only today is available for now; it
// writes the error response and returns false otherwise.
func (h *handler) requestedDay(w http.ResponseWriter, r *http.Request) (time.Time, bool) {
	day, err := habit.ParseDay(r.PathValue("date"))
	if err != nil {
		writeError(w, http.StatusUnprocessableEntity, "invalid_day", "Date invalide.")
		return time.Time{}, false
	}
	if !day.Equal(h.today()) {
		writeError(w, http.StatusUnprocessableEntity, "day_not_editable", "Seul le jour courant est disponible pour l'instant.")
		return time.Time{}, false
	}
	return day, true
}

func (h *handler) getDay(w http.ResponseWriter, r *http.Request) {
	day, ok := h.requestedDay(w, r)
	if !ok {
		return
	}
	h.writeDay(w, r, day, http.StatusOK)
}

func (h *handler) writeDay(w http.ResponseWriter, r *http.Request, day time.Time, status int) {
	logged, err := h.cfg.Store.DayHabits(r.Context(), day)
	if err != nil {
		h.internalError(w, r, err)
		return
	}
	writeJSON(w, status, buildDay(day, logged))
}

func (h *handler) putEntry(w http.ResponseWriter, r *http.Request) {
	day, ok := h.requestedDay(w, r)
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
