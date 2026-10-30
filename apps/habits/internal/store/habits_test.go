package store

import (
	"context"
	"errors"
	"testing"
	"time"

	"zazi.cool/habits/internal/dbtest"
	"zazi.cool/habits/internal/habit"
	"zazi.cool/habits/internal/migrate"
	"zazi.cool/habits/migrations"
)

func day(t *testing.T, s string) time.Time {
	t.Helper()
	d, err := habit.ParseDay(s)
	if err != nil {
		t.Fatal(err)
	}
	return d
}

func newStore(t *testing.T) *Store {
	t.Helper()
	pool := dbtest.New(t)
	if err := migrate.Up(context.Background(), pool, migrations.FS); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return New(pool)
}

func names(items []habit.DayItem) []string {
	var out []string
	for _, it := range items {
		out = append(out, it.Name)
	}
	return out
}

func idOf(t *testing.T, s *Store, name string) int64 {
	t.Helper()
	habits, err := s.ListHabits(context.Background(), day(t, "2030-01-01"))
	if err != nil {
		t.Fatal(err)
	}
	for _, h := range habits {
		if h.Name == name && !h.Archived() {
			return h.ID
		}
	}
	t.Fatalf("no active habit %q", name)
	return 0
}

// CA1: a second active habit with the same name is refused.
func TestCreateHabitRefusesDuplicateActiveName(t *testing.T) {
	ctx := context.Background()
	s := newStore(t)
	today := day(t, "2026-10-30")

	if err := s.CreateHabit(ctx, "Lire", habit.AllDays, today); err != nil {
		t.Fatal(err)
	}
	if err := s.CreateHabit(ctx, "Lire", habit.AllDays, today); !errors.Is(err, habit.ErrNameTaken) {
		t.Errorf("duplicate create = %v, want ErrNameTaken", err)
	}

	// Once archived, the name is free again; restoring the old one then clashes.
	oldID := idOf(t, s, "Lire")
	if err := s.ArchiveHabit(ctx, oldID, today); err != nil {
		t.Fatal(err)
	}
	if err := s.CreateHabit(ctx, "Lire", habit.AllDays, today); err != nil {
		t.Errorf("create after archive: %v", err)
	}
	if err := s.RestoreHabit(ctx, oldID); !errors.Is(err, habit.ErrNameTaken) {
		t.Errorf("restore over a taken name = %v, want ErrNameTaken", err)
	}
}

// CA2: a checked habit stays checked; unchecking removes the completion row.
func TestSetDoneRoundTrip(t *testing.T) {
	ctx := context.Background()
	s := newStore(t)
	today := day(t, "2026-10-30")
	if err := s.CreateHabit(ctx, "Lire", habit.AllDays, today); err != nil {
		t.Fatal(err)
	}
	id := idOf(t, s, "Lire")

	items, err := s.DayHabits(ctx, today)
	if err != nil || len(items) != 1 || items[0].Done {
		t.Fatalf("before: %v, %v", items, err)
	}
	// Twice: checking is idempotent.
	for range 2 {
		if err := s.SetDone(ctx, id, today, true); err != nil {
			t.Fatal(err)
		}
	}
	if items, _ = s.DayHabits(ctx, today); !items[0].Done {
		t.Error("after check: not done")
	}

	if err := s.SetDone(ctx, id, today, false); err != nil {
		t.Fatal(err)
	}
	if items, _ = s.DayHabits(ctx, today); items[0].Done {
		t.Error("after uncheck: still done")
	}
	var rows int
	if err := s.pool.QueryRow(ctx, `SELECT count(*) FROM completions`).Scan(&rows); err != nil || rows != 0 {
		t.Errorf("completions rows = %d, %v; want 0", rows, err)
	}
}

func TestSetDoneErrors(t *testing.T) {
	ctx := context.Background()
	s := newStore(t)
	created := day(t, "2026-10-10")
	// 2026-10-30 is a Friday.
	mondays, _ := habit.ParseWeekdays([]string{"1"})
	if err := s.CreateHabit(ctx, "Sport", mondays, created); err != nil {
		t.Fatal(err)
	}
	id := idOf(t, s, "Sport")

	if err := s.SetDone(ctx, 999, created, true); !errors.Is(err, habit.ErrNotFound) {
		t.Errorf("unknown habit = %v, want ErrNotFound", err)
	}
	// A Tuesday: the habit does not apply.
	if err := s.SetDone(ctx, id, day(t, "2026-10-13"), true); !errors.Is(err, habit.ErrNotScheduled) {
		t.Errorf("unscheduled day = %v, want ErrNotScheduled", err)
	}
	// A Monday before the creation day.
	if err := s.SetDone(ctx, id, day(t, "2026-10-05"), true); !errors.Is(err, habit.ErrNotScheduled) {
		t.Errorf("before creation = %v, want ErrNotScheduled", err)
	}
	if err := s.SetDone(ctx, id, day(t, "2026-10-12"), true); err != nil {
		t.Errorf("scheduled Monday: %v", err)
	}
}

// CA5: restricting a daily habit to Mondays from today leaves past days unchanged.
func TestScheduleChangeKeepsHistory(t *testing.T) {
	ctx := context.Background()
	s := newStore(t)
	if err := s.CreateHabit(ctx, "Lire", habit.AllDays, day(t, "2026-10-01")); err != nil {
		t.Fatal(err)
	}
	id := idOf(t, s, "Lire")
	today := day(t, "2026-10-30") // a Friday
	mondays, _ := habit.ParseWeekdays([]string{"1"})

	if err := s.UpdateHabit(ctx, id, "Lire", mondays, today); err != nil {
		t.Fatal(err)
	}

	for d, want := range map[string]bool{
		"2026-10-29": true,  // Thursday before the change
		"2026-10-30": false, // Friday, the day of the change
		"2026-11-02": true,  // next Monday
		"2026-11-03": false,
	} {
		items, err := s.DayHabits(ctx, day(t, d))
		if err != nil {
			t.Fatal(err)
		}
		if got := len(items) == 1; got != want {
			t.Errorf("habit applies on %s = %v, want %v", d, got, want)
		}
	}

	// A second edit the same day replaces that day's version instead of failing.
	if err := s.UpdateHabit(ctx, id, "Lire", habit.AllDays, today); err != nil {
		t.Fatal(err)
	}
	if items, _ := s.DayHabits(ctx, today); len(items) != 1 {
		t.Error("habit should apply today again")
	}
}

func TestUpdateHabit(t *testing.T) {
	ctx := context.Background()
	s := newStore(t)
	today := day(t, "2026-10-30")
	for _, n := range []string{"Lire", "Sport"} {
		if err := s.CreateHabit(ctx, n, habit.AllDays, today); err != nil {
			t.Fatal(err)
		}
	}
	id := idOf(t, s, "Lire")

	if err := s.UpdateHabit(ctx, id, "Sport", habit.AllDays, today); !errors.Is(err, habit.ErrNameTaken) {
		t.Errorf("rename to a taken name = %v, want ErrNameTaken", err)
	}
	if err := s.UpdateHabit(ctx, 999, "X", habit.AllDays, today); !errors.Is(err, habit.ErrNotFound) {
		t.Errorf("unknown habit = %v, want ErrNotFound", err)
	}
	if err := s.UpdateHabit(ctx, id, "Méditer", habit.AllDays, today); err != nil {
		t.Fatal(err)
	}
	items, _ := s.DayHabits(ctx, today)
	if got := names(items); len(got) != 2 || got[0] != "Méditer" || got[1] != "Sport" {
		t.Errorf("names = %v, want [Méditer Sport]", got)
	}
}

// CA6: an archived habit leaves today and later days but keeps its past; restoring
// brings it back.
func TestArchiveAndRestore(t *testing.T) {
	ctx := context.Background()
	s := newStore(t)
	if err := s.CreateHabit(ctx, "Lire", habit.AllDays, day(t, "2026-10-01")); err != nil {
		t.Fatal(err)
	}
	id := idOf(t, s, "Lire")
	past, today := day(t, "2026-10-20"), day(t, "2026-10-30")
	if err := s.SetDone(ctx, id, past, true); err != nil {
		t.Fatal(err)
	}

	if err := s.ArchiveHabit(ctx, id, today); err != nil {
		t.Fatal(err)
	}
	if err := s.ArchiveHabit(ctx, id, day(t, "2026-11-05")); err != nil { // keeps the first date
		t.Fatal(err)
	}
	if items, _ := s.DayHabits(ctx, today); len(items) != 0 {
		t.Errorf("today after archive: %v, want none", names(items))
	}
	items, _ := s.DayHabits(ctx, past)
	if len(items) != 1 || !items[0].Done {
		t.Errorf("past day after archive: %v, want the habit, done", items)
	}

	habits, err := s.ListHabits(ctx, today)
	if err != nil || len(habits) != 1 || !habits[0].Archived() {
		t.Fatalf("ListHabits = %v, %v; want one archived habit", habits, err)
	}

	if err := s.RestoreHabit(ctx, id); err != nil {
		t.Fatal(err)
	}
	if items, _ := s.DayHabits(ctx, today); len(items) != 1 {
		t.Error("restored habit missing today")
	}
	for _, err := range []error{s.ArchiveHabit(ctx, 999, today), s.RestoreHabit(ctx, 999)} {
		if !errors.Is(err, habit.ErrNotFound) {
			t.Errorf("unknown habit = %v, want ErrNotFound", err)
		}
	}
}

// History feeds the statistics: schedule versions stay, completions are limited to
// the range, and a habit archived before the range is left out.
func TestHistory(t *testing.T) {
	ctx := context.Background()
	s := newStore(t)
	created := day(t, "2026-10-01")
	if err := s.CreateHabit(ctx, "Lire", habit.AllDays, created); err != nil {
		t.Fatal(err)
	}
	if err := s.CreateHabit(ctx, "Ancienne", habit.AllDays, created); err != nil {
		t.Fatal(err)
	}
	lire, old := idOf(t, s, "Lire"), idOf(t, s, "Ancienne")
	mondays, _ := habit.ParseWeekdays([]string{"1"})
	if err := s.UpdateHabit(ctx, lire, "Lire", mondays, day(t, "2026-10-26")); err != nil {
		t.Fatal(err)
	}
	for _, d := range []string{"2026-10-05", "2026-10-20", "2026-10-26"} {
		if err := s.SetDone(ctx, lire, day(t, d), true); err != nil {
			t.Fatal(err)
		}
	}
	if err := s.ArchiveHabit(ctx, old, day(t, "2026-10-10")); err != nil {
		t.Fatal(err)
	}

	got, err := s.History(ctx, day(t, "2026-10-15"), day(t, "2026-10-31"))
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Name != "Lire" {
		t.Fatalf("history = %+v, want only Lire", got)
	}
	h := got[0]
	if len(h.Schedules) != 2 || h.Schedules[1].Weekdays != mondays {
		t.Errorf("schedules = %+v, want 2 versions ending on Mondays", h.Schedules)
	}
	if len(h.Done) != 2 || !h.Done[day(t, "2026-10-20")] || !h.Done[day(t, "2026-10-26")] {
		t.Errorf("done = %v, want 20 and 26 October only", h.Done)
	}
	if h.Applies(day(t, "2026-10-27")) || !h.Applies(day(t, "2026-10-21")) {
		t.Error("Applies does not follow the schedule versions")
	}

	// Archived on the 10th: still listed for a range that covers days before it.
	got, err = s.History(ctx, day(t, "2026-10-01"), day(t, "2026-10-31"))
	if err != nil || len(got) != 2 {
		t.Errorf("full range = %d habits, %v; want 2", len(got), err)
	}
}
