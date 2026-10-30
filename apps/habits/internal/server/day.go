package server

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"zazi.cool/habits/internal/habit"
)

var (
	frenchWeekdays = [...]string{"dimanche", "lundi", "mardi", "mercredi", "jeudi", "vendredi", "samedi"}
	frenchMonths   = [...]string{"janvier", "février", "mars", "avril", "mai", "juin", "juillet",
		"août", "septembre", "octobre", "novembre", "décembre"}
)

// frenchDate renders a day like "vendredi 30 octobre 2026".
func frenchDate(d time.Time) string {
	return frenchWeekdays[d.Weekday()] + " " + strconv.Itoa(d.Day()) + " " +
		frenchMonths[d.Month()-1] + " " + strconv.Itoa(d.Year())
}

func (h *handler) today() time.Time {
	return habit.Today(h.cfg.Now(), h.cfg.Location)
}

type dayPage struct {
	Label string
	// Date is the YYYY-MM-DD form used in URLs.
	Date  string
	Items []habit.DayItem
	// Editable is false for future days and days older than the edit window.
	Editable bool
	// PrevHref and NextHref are empty when there is nowhere to go.
	PrevHref, NextHref string
}

// dayHref is the URL of a day; today lives at "/".
func dayHref(day, today time.Time) string {
	if day.Equal(today) {
		return "/"
	}
	return "/day/" + habit.FormatDay(day)
}

// home shows today (F4).
func (h *handler) home(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	h.showDay(w, r, h.today())
}

// day shows any day, past or future; only the check boxes depend on the edit window.
func (h *handler) day(w http.ResponseWriter, r *http.Request) {
	day, err := habit.ParseDay(r.PathValue("date"))
	if err != nil {
		http.Error(w, "Date invalide (format AAAA-MM-JJ).", http.StatusBadRequest)
		return
	}
	h.showDay(w, r, day)
}

func (h *handler) showDay(w http.ResponseWriter, r *http.Request, day time.Time) {
	today := h.today()
	items, err := h.cfg.Store.DayHabits(r.Context(), day)
	if err != nil {
		h.internalError(w, r, err)
		return
	}
	page := dayPage{
		Label:    frenchDate(day),
		Date:     habit.FormatDay(day),
		Items:    items,
		Editable: habit.Editable(day, today, h.cfg.EditWindowDays),
	}
	// Previous days are offered as long as they are still editable; "next" stops at today.
	if prev := day.AddDate(0, 0, -1); habit.Editable(prev, today, h.cfg.EditWindowDays) {
		page.PrevHref = dayHref(prev, today)
	}
	if day.Before(today) {
		page.NextHref = dayHref(day.AddDate(0, 0, 1), today)
	}
	h.render(w, r, http.StatusOK, "day", page)
}

type toggleResult struct {
	HabitID int64  `json:"habit_id"`
	Day     string `json:"day"`
	Done    bool   `json:"done"`
}

// toggle checks or unchecks one habit on one day (F4, F5). With "Accept:
// application/json" it answers with the new state, otherwise it redirects back.
func (h *handler) toggle(w http.ResponseWriter, r *http.Request) {
	day, err := habit.ParseDay(r.PathValue("date"))
	if err != nil {
		http.Error(w, "Date invalide (format AAAA-MM-JJ).", http.StatusBadRequest)
		return
	}
	if !parseForm(w, r) {
		return
	}
	habitID, err := strconv.ParseInt(r.PostForm.Get("habit_id"), 10, 64)
	done := r.PostForm.Get("done")
	if err != nil || (done != "0" && done != "1") {
		http.Error(w, "habit_id ou done invalide.", http.StatusUnprocessableEntity)
		return
	}
	today := h.today()
	if !habit.Editable(day, today, h.cfg.EditWindowDays) {
		http.Error(w, "Ce jour n'est pas modifiable.", http.StatusUnprocessableEntity)
		return
	}

	err = h.cfg.Store.SetDone(r.Context(), habitID, day, done == "1")
	switch {
	case errors.Is(err, habit.ErrNotFound):
		http.Error(w, "Habitude introuvable.", http.StatusNotFound)
		return
	case errors.Is(err, habit.ErrNotScheduled):
		http.Error(w, "Cette habitude ne concerne pas ce jour.", http.StatusUnprocessableEntity)
		return
	case err != nil:
		h.internalError(w, r, err)
		return
	}

	if strings.Contains(r.Header.Get("Accept"), "application/json") {
		w.Header().Set("Content-Type", "application/json")
		res := toggleResult{HabitID: habitID, Day: habit.FormatDay(day), Done: done == "1"}
		if err := json.NewEncoder(w).Encode(res); err != nil {
			slog.Warn("write response", "path", r.URL.Path, "error", err)
		}
		return
	}
	http.Redirect(w, r, dayHref(day, today), http.StatusSeeOther)
}
