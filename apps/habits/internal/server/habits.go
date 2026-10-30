package server

import (
	"errors"
	"net/http"
	"strconv"

	"zazi.cool/habits/internal/habit"
)

var isoWeekdayLabels = [...]string{"lun", "mar", "mer", "jeu", "ven", "sam", "dim"}

// dayBox is one weekday check box of the habit forms; Value is the ISO number (1 = Monday).
type dayBox struct {
	Value   int
	Label   string
	Checked bool
}

func weekdayBoxes(days habit.Weekdays) []dayBox {
	boxes := make([]dayBox, len(isoWeekdayLabels))
	for i, label := range isoWeekdayLabels {
		boxes[i] = dayBox{Value: i + 1, Label: label, Checked: days&(1<<i) != 0}
	}
	return boxes
}

type habitRow struct {
	ID   int64
	Name string
	Days []dayBox
}

type habitsView struct {
	Active, Archived []habitRow
	// NewDays is the check box set of the creation form (every day checked).
	NewDays []dayBox
	// Error is shown above the forms; NewName keeps what was typed in the creation form.
	Error   string
	NewName string
}

// renderHabits shows the management page with the given status (200, or an error status
// when a form was refused and the page is shown again with its message).
func (h *handler) renderHabits(w http.ResponseWriter, r *http.Request, status int, msg, newName string) {
	habits, err := h.cfg.Store.ListHabits(r.Context(), h.today())
	if err != nil {
		h.internalError(w, r, err)
		return
	}
	page := habitsView{NewDays: weekdayBoxes(habit.AllDays), Error: msg, NewName: newName}
	for _, hb := range habits {
		row := habitRow{ID: hb.ID, Name: hb.Name, Days: weekdayBoxes(hb.Weekdays)}
		if hb.Archived() {
			page.Archived = append(page.Archived, row)
		} else {
			page.Active = append(page.Active, row)
		}
	}
	h.render(w, r, status, "habits", page)
}

func (h *handler) habitsIndex(w http.ResponseWriter, r *http.Request) {
	h.renderHabits(w, r, http.StatusOK, "", "")
}

// createHabit adds a habit (F1). With no weekday ticked, it applies every day.
func (h *handler) createHabit(w http.ResponseWriter, r *http.Request) {
	if !parseForm(w, r) {
		return
	}
	name, err := habit.ValidateName(r.PostForm.Get("name"))
	if err != nil {
		h.renderHabits(w, r, http.StatusUnprocessableEntity, "Le nom doit faire de 1 à 60 caractères, sans retour à la ligne.", r.PostForm.Get("name"))
		return
	}
	days := habit.AllDays
	if values := r.PostForm["weekdays"]; len(values) > 0 {
		if days, err = habit.ParseWeekdays(values); err != nil {
			h.renderHabits(w, r, http.StatusUnprocessableEntity, "Jours invalides.", name)
			return
		}
	}
	err = h.cfg.Store.CreateHabit(r.Context(), name, days, h.today())
	if errors.Is(err, habit.ErrNameTaken) {
		h.renderHabits(w, r, http.StatusConflict, "Une habitude active porte déjà ce nom.", name)
		return
	}
	if err != nil {
		h.internalError(w, r, err)
		return
	}
	http.Redirect(w, r, "/habits", http.StatusSeeOther)
}

// updateHabit renames a habit and changes its weekdays from today on (F2).
func (h *handler) updateHabit(w http.ResponseWriter, r *http.Request) {
	id, ok := habitID(w, r)
	if !ok || !parseForm(w, r) {
		return
	}
	name, err := habit.ValidateName(r.PostForm.Get("name"))
	if err != nil {
		h.renderHabits(w, r, http.StatusUnprocessableEntity, "Le nom doit faire de 1 à 60 caractères, sans retour à la ligne.", "")
		return
	}
	days, err := habit.ParseWeekdays(r.PostForm["weekdays"])
	if err != nil {
		h.renderHabits(w, r, http.StatusUnprocessableEntity, "Choisissez au moins un jour.", "")
		return
	}
	err = h.cfg.Store.UpdateHabit(r.Context(), id, name, days, h.today())
	h.habitResult(w, r, err)
}

func (h *handler) archiveHabit(w http.ResponseWriter, r *http.Request) {
	if id, ok := habitID(w, r); ok {
		h.habitResult(w, r, h.cfg.Store.ArchiveHabit(r.Context(), id, h.today()))
	}
}

func (h *handler) restoreHabit(w http.ResponseWriter, r *http.Request) {
	if id, ok := habitID(w, r); ok {
		h.habitResult(w, r, h.cfg.Store.RestoreHabit(r.Context(), id))
	}
}

// habitResult turns the outcome of a habit change into a response.
func (h *handler) habitResult(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, habit.ErrNotFound):
		http.Error(w, "Habitude introuvable.", http.StatusNotFound)
	case errors.Is(err, habit.ErrNameTaken):
		h.renderHabits(w, r, http.StatusConflict, "Une habitude active porte déjà ce nom.", "")
	case err != nil:
		h.internalError(w, r, err)
	default:
		http.Redirect(w, r, "/habits", http.StatusSeeOther)
	}
}

// habitID reads the {id} of the path; a malformed one is a habit that does not exist.
func habitID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "Habitude introuvable.", http.StatusNotFound)
		return 0, false
	}
	return id, true
}
