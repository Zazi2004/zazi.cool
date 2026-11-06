package server

import (
	"errors"
	"net/http"
	"slices"
	"strconv"

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
	// RuleStarts gives, for each period, the day a change of period, target or weekdays
	// would take effect if saved today (the start of the current period). The form shows it
	// before saving; the rule stays computed in Go (NF5).
	RuleStarts map[string]string `json:"ruleStarts"`
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
	today := h.today()
	starts := make(map[string]string, len(habit.Periods))
	for _, p := range habit.Periods {
		starts[string(p)] = habit.FormatDay(habit.PeriodStart(p, today))
	}
	writeJSON(w, http.StatusOK, habitListJSON{
		Active: toHabitsJSON(active), Archived: toHabitsJSON(archived), RuleStarts: starts,
	})
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

// habitPatchJSON is the partial HabitInput of PATCH /api/habits/{id}: an absent field
// keeps its value.
type habitPatchJSON struct {
	Name     *string `json:"name"`
	Emoji    *string `json:"emoji"`
	Color    *string `json:"color"`
	Moment   *string `json:"moment"`
	Unit     *string `json:"unit"`
	Step     *int    `json:"step"`
	Period   *string `json:"period"`
	Target   *int    `json:"target"`
	Weekdays *int    `json:"weekdays"`
}

func (p habitPatchJSON) toPatch() habit.Patch {
	out := habit.Patch{Name: p.Name, Emoji: p.Emoji, Unit: p.Unit, Step: p.Step, Target: p.Target}
	if p.Color != nil {
		c := habit.Color(*p.Color)
		out.Color = &c
	}
	if p.Moment != nil {
		m := habit.Moment(*p.Moment)
		out.Moment = &m
	}
	if p.Period != nil {
		per := habit.Period(*p.Period)
		out.Period = &per
	}
	if p.Weekdays != nil {
		wd := weekdaysFrom(*p.Weekdays)
		out.Weekdays = &wd
	}
	return out
}

// habitID reads the {id} of the path; it answers 404 itself when it is not a number.
func habitID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id < 1 {
		apiNotFound(w, r)
		return 0, false
	}
	return id, true
}

// writeHabitResult answers a habit-changing request: the habit, or the error of the
// store mapped to its HTTP status.
func (h *handler) writeHabitResult(w http.ResponseWriter, r *http.Request, out habit.Habit, err error) {
	var ve *habit.ValidationError
	switch {
	case errors.As(err, &ve):
		writeError(w, http.StatusUnprocessableEntity, "invalid_habit", ve.Message)
	case errors.Is(err, habit.ErrNotFound):
		writeError(w, http.StatusNotFound, "not_found", "Cette habitude n'existe pas ou est archivée.")
	case errors.Is(err, habit.ErrNameTaken):
		writeError(w, http.StatusConflict, "name_taken", "Une habitude active porte déjà ce nom.")
	case err != nil:
		h.internalError(w, r, err)
	default:
		writeJSON(w, http.StatusOK, toHabitJSON(out))
	}
}

func (h *handler) updateHabit(w http.ResponseWriter, r *http.Request) {
	id, ok := habitID(w, r)
	if !ok {
		return
	}
	var req habitPatchJSON
	if !decodeJSON(w, r, &req) {
		return
	}
	out, err := h.cfg.Store.UpdateHabit(r.Context(), id, req.toPatch(), h.today())
	h.writeHabitResult(w, r, out, err)
}

func (h *handler) archiveHabit(w http.ResponseWriter, r *http.Request) {
	id, ok := habitID(w, r)
	if !ok {
		return
	}
	out, err := h.cfg.Store.ArchiveHabit(r.Context(), id, h.today())
	h.writeHabitResult(w, r, out, err)
}

func (h *handler) restoreHabit(w http.ResponseWriter, r *http.Request) {
	id, ok := habitID(w, r)
	if !ok {
		return
	}
	out, err := h.cfg.Store.RestoreHabit(r.Context(), id, h.today())
	h.writeHabitResult(w, r, out, err)
}

func (h *handler) reorderHabits(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Moment string  `json:"moment"`
		IDs    []int64 `json:"ids"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	moment := habit.Moment(req.Moment)
	if !slices.Contains(habit.Moments, moment) {
		writeError(w, http.StatusUnprocessableEntity, "invalid_order", "Moment inconnu.")
		return
	}
	err := h.cfg.Store.ReorderHabits(r.Context(), moment, req.IDs, h.today())
	var ve *habit.ValidationError
	switch {
	case errors.As(err, &ve):
		writeError(w, http.StatusUnprocessableEntity, "invalid_order", ve.Message)
	case err != nil:
		h.internalError(w, r, err)
	default:
		w.WriteHeader(http.StatusNoContent)
	}
}
