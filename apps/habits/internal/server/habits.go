package server

import (
	"errors"
	"net/http"

	"zazi.cool/habits/internal/habit"
)

// habitJSON is the Habit of the spec (§7). Its field names are the contract with the
// front: the golden files check them against the TypeScript types.
type habitJSON struct {
	ID         int64   `json:"id"`
	Name       string  `json:"name"`
	Emoji      string  `json:"emoji"`
	Color      string  `json:"color"`
	Moment     string  `json:"moment"`
	Position   int     `json:"position"`
	Unit       string  `json:"unit"`
	Step       int     `json:"step"`
	Period     string  `json:"period"`
	Target     int     `json:"target"`
	Weekdays   int     `json:"weekdays"`
	ArchivedOn *string `json:"archivedOn"`
}

func toHabitJSON(h habit.Habit) habitJSON {
	out := habitJSON{
		ID: h.ID, Name: h.Name, Emoji: h.Emoji, Color: string(h.Color), Moment: string(h.Moment),
		Position: h.Position, Unit: h.Unit, Step: h.Step, Period: string(h.Period),
		Target: h.Target, Weekdays: int(h.Weekdays),
	}
	if h.ArchivedOn != nil {
		day := habit.FormatDay(*h.ArchivedOn)
		out.ArchivedOn = &day
	}
	return out
}

// habitListJSON is the answer of GET /api/habits.
type habitListJSON struct {
	Active   []habitJSON `json:"active"`
	Archived []habitJSON `json:"archived"`
}

func toHabitsJSON(habits []habit.Habit) []habitJSON {
	// Never nil: an empty list must be [] in JSON, not null.
	out := make([]habitJSON, 0, len(habits))
	for _, h := range habits {
		out = append(out, toHabitJSON(h))
	}
	return out
}

func (h *handler) listHabits(w http.ResponseWriter, r *http.Request) {
	active, archived, err := h.cfg.Store.ListHabits(r.Context(), h.today())
	if err != nil {
		h.internalError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, habitListJSON{Active: toHabitsJSON(active), Archived: toHabitsJSON(archived)})
}

// habitInputJSON is the HabitInput of the spec (§7).
type habitInputJSON struct {
	Name     string `json:"name"`
	Emoji    string `json:"emoji"`
	Color    string `json:"color"`
	Moment   string `json:"moment"`
	Unit     string `json:"unit"`
	Step     int    `json:"step"`
	Period   string `json:"period"`
	Target   int    `json:"target"`
	Weekdays int    `json:"weekdays"`
}

func (h *handler) createHabit(w http.ResponseWriter, r *http.Request) {
	var req habitInputJSON
	if !decodeJSON(w, r, &req) {
		return
	}
	in, err := habit.Input{
		Name: req.Name, Emoji: req.Emoji, Color: habit.Color(req.Color), Moment: habit.Moment(req.Moment),
		Unit: req.Unit, Step: req.Step, Period: habit.Period(req.Period), Target: req.Target,
		// A mask outside 0..255 would wrap in the uint8 conversion: map it to an invalid one.
		Weekdays: weekdaysFrom(req.Weekdays),
	}.Validate()
	if err != nil {
		var ve *habit.ValidationError
		if errors.As(err, &ve) {
			writeError(w, http.StatusUnprocessableEntity, "invalid_habit", ve.Message)
			return
		}
		h.internalError(w, r, err)
		return
	}
	created, err := h.cfg.Store.CreateHabit(r.Context(), in, h.today())
	if errors.Is(err, habit.ErrNameTaken) {
		writeError(w, http.StatusConflict, "name_taken", "Une habitude active porte déjà ce nom.")
		return
	}
	if err != nil {
		h.internalError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, toHabitJSON(created))
}

func weekdaysFrom(mask int) habit.Weekdays {
	if mask < 0 || mask > int(habit.AllDays) {
		return 0
	}
	return habit.Weekdays(mask)
}
