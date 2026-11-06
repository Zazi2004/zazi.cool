package store

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"testing"
	"time"

	"zazi.cool/habits/internal/habit"
)

func ptr[T any](v T) *T { return &v }

// rules lists the stored versions of a habit as "AAAA-MM-JJ period target".
func rules(t *testing.T, s *Store, id int64) []string {
	t.Helper()
	rows, err := s.pool.Query(context.Background(),
		`SELECT valid_from, period, target FROM habit_rules WHERE habit_id = $1 ORDER BY valid_from`, id)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var (
			from   time.Time
			period string
			target int
		)
		if err := rows.Scan(&from, &period, &target); err != nil {
			t.Fatal(err)
		}
		out = append(out, fmt.Sprintf("%s %s %d", habit.FormatDay(from), period, target))
	}
	return out
}

func weekly(name string, target int) habit.Input {
	in := input(name, habit.Evening)
	in.Unit, in.Period, in.Target = "sorties", habit.PeriodWeek, target
	return in
}

// CA4: the new target starts at the current week; the past keeps its rule.
func TestUpdateHabitVersionsRule(t *testing.T) {
	ctx := context.Background()
	s := newStore(t)
	created, err := s.CreateHabit(ctx, weekly("Course", 3), day("2026-10-01"))
	if err != nil {
		t.Fatal(err)
	}
	thursday := day("2026-11-05")
	got, err := s.UpdateHabit(ctx, created.ID, habit.Patch{Target: ptr(4)}, thursday)
	if err != nil {
		t.Fatal(err)
	}
	if got.Target != 4 || got.Name != "Course" {
		t.Errorf("updated = %+v", got)
	}
	want := []string{"2026-10-01 week 3", "2026-11-02 week 4"}
	if got := rules(t, s, created.ID); !slices.Equal(got, want) {
		t.Fatalf("rules = %v, want %v", got, want)
	}

	// The past week is still read with 3, the current one with 4.
	for _, c := range []struct {
		day    string
		target int
	}{{"2026-11-01", 3}, {"2026-11-02", 4}, {"2026-11-05", 4}} {
		active, _, err := s.ListHabits(ctx, day(c.day))
		if err != nil || len(active) != 1 || active[0].Target != c.target {
			t.Errorf("on %s: %+v, %v; want target %d", c.day, active, err, c.target)
		}
	}
}

// Two edits in the same period leave one version; undoing the first leaves none.
func TestUpdateHabitSamePeriodReplacesVersion(t *testing.T) {
	ctx := context.Background()
	s := newStore(t)
	created, _ := s.CreateHabit(ctx, weekly("Course", 3), day("2026-10-01"))

	for _, target := range []int{4, 5} {
		if _, err := s.UpdateHabit(ctx, created.ID, habit.Patch{Target: ptr(target)}, day("2026-11-05")); err != nil {
			t.Fatal(err)
		}
	}
	if got, want := rules(t, s, created.ID), []string{"2026-10-01 week 3", "2026-11-02 week 5"}; !slices.Equal(got, want) {
		t.Errorf("rules = %v, want %v", got, want)
	}

	if _, err := s.UpdateHabit(ctx, created.ID, habit.Patch{Target: ptr(3)}, day("2026-11-06")); err != nil {
		t.Fatal(err)
	}
	if got, want := rules(t, s, created.ID), []string{"2026-10-01 week 3"}; !slices.Equal(got, want) {
		t.Errorf("after undo, rules = %v, want %v", got, want)
	}
}

// CA5: week to month on 6 November starts the version on 1 November, so the week of
// 26 to 31 October is the last one of the old rule, cut the day before.
func TestUpdateHabitPeriodChange(t *testing.T) {
	ctx := context.Background()
	s := newStore(t)
	created, _ := s.CreateHabit(ctx, weekly("Course", 3), day("2026-10-01"))
	if _, err := s.UpdateHabit(ctx, created.ID, habit.Patch{Period: ptr(habit.PeriodMonth)}, day("2026-11-06")); err != nil {
		t.Fatal(err)
	}
	if got, want := rules(t, s, created.ID), []string{"2026-10-01 week 3", "2026-11-01 month 3"}; !slices.Equal(got, want) {
		t.Errorf("rules = %v, want %v", got, want)
	}
}

func TestUpdateHabitUnversionedFieldsWriteNoRule(t *testing.T) {
	ctx := context.Background()
	s := newStore(t)
	created, _ := s.CreateHabit(ctx, input("Vitamine D", habit.Morning), day("2026-11-01"))
	other, _ := s.CreateHabit(ctx, input("Lire", habit.Evening), day("2026-11-01"))

	got, err := s.UpdateHabit(ctx, created.ID, habit.Patch{
		Name: ptr("Vitamine D3"), Emoji: ptr("☀️"), Color: ptr(habit.Color("blue")),
		Moment: ptr(habit.Evening), Unit: ptr("gélules"), Step: ptr(2),
	}, day("2026-11-06"))
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != "Vitamine D3" || got.Moment != habit.Evening || got.Unit != "gélules" || got.Step != 2 {
		t.Errorf("updated = %+v", got)
	}
	if got.Position != other.Position+1 {
		t.Errorf("position = %d, want last of the evening (%d)", got.Position, other.Position+1)
	}
	if rs := rules(t, s, created.ID); len(rs) != 1 {
		t.Errorf("rules = %v, want the original version only", rs)
	}
}

func TestUpdateHabitRefusals(t *testing.T) {
	ctx := context.Background()
	s := newStore(t)
	today := day("2026-11-06")
	a, _ := s.CreateHabit(ctx, input("Vitamine D", habit.Morning), today)
	b, _ := s.CreateHabit(ctx, input("Lire", habit.Morning), today)

	var ve *habit.ValidationError
	if _, err := s.UpdateHabit(ctx, a.ID, habit.Patch{Target: ptr(0)}, today); !errors.As(err, &ve) {
		t.Errorf("target 0: %v, want a validation error", err)
	}
	if _, err := s.UpdateHabit(ctx, a.ID, habit.Patch{Name: ptr("Lire")}, today); !errors.Is(err, habit.ErrNameTaken) {
		t.Errorf("name taken: %v, want ErrNameTaken", err)
	}
	if _, err := s.UpdateHabit(ctx, 999, habit.Patch{}, today); !errors.Is(err, habit.ErrNotFound) {
		t.Errorf("unknown: %v, want ErrNotFound", err)
	}
	if _, err := s.ArchiveHabit(ctx, b.ID, today); err != nil {
		t.Fatal(err)
	}
	if _, err := s.UpdateHabit(ctx, b.ID, habit.Patch{Name: ptr("Piano")}, today); !errors.Is(err, habit.ErrNotFound) {
		t.Errorf("archived: %v, want ErrNotFound", err)
	}
	// A refused edit changes nothing.
	active, _, _ := s.ListHabits(ctx, today)
	if len(active) != 1 || active[0].Name != "Vitamine D" || active[0].Target != 2 {
		t.Errorf("active = %+v", active)
	}
}

// Archiving removes the habit from Today from that day on and keeps its history;
// restoring brings it back, last in its moment, with its entries intact.
func TestArchiveAndRestore(t *testing.T) {
	ctx := context.Background()
	s := newStore(t)
	created, _ := s.CreateHabit(ctx, input("Vitamine D", habit.Morning), day("2026-11-01"))
	if _, err := s.CreateHabit(ctx, input("Lire", habit.Morning), day("2026-11-01")); err != nil {
		t.Fatal(err)
	}
	if err := s.SetEntry(ctx, created.ID, day("2026-11-04"), 2); err != nil {
		t.Fatal(err)
	}
	today := day("2026-11-06")

	archived, err := s.ArchiveHabit(ctx, created.ID, today)
	if err != nil || archived.ArchivedOn == nil {
		t.Fatalf("archive = %+v, %v", archived, err)
	}
	if again, err := s.ArchiveHabit(ctx, created.ID, day("2026-11-07")); err != nil || !again.ArchivedOn.Equal(today) {
		t.Errorf("archiving twice = %+v, %v; want the first date kept", again, err)
	}
	if logged, _ := s.DayHabits(ctx, today); len(logged) != 1 || logged[0].Habit.Name != "Lire" {
		t.Errorf("today after archive = %+v", logged)
	}
	if logged, _ := s.DayHabits(ctx, day("2026-11-04")); len(logged) != 2 || logged[0].Value != 2 {
		t.Errorf("past day after archive = %+v, want the habit and its entry", logged)
	}
	active, archivedList, _ := s.ListHabits(ctx, today)
	if len(active) != 1 || len(archivedList) != 1 {
		t.Errorf("active = %d, archived = %d, want 1 and 1", len(active), len(archivedList))
	}

	restored, err := s.RestoreHabit(ctx, created.ID, today)
	if err != nil || restored.ArchivedOn != nil || restored.Position != 2 {
		t.Fatalf("restore = %+v, %v; want active, position 2", restored, err)
	}
	if logged, _ := s.DayHabits(ctx, today); len(logged) != 2 || logged[1].Habit.Name != "Vitamine D" {
		t.Errorf("today after restore = %+v", logged)
	}
	if logged, _ := s.DayHabits(ctx, day("2026-11-04")); logged[1].Value != 2 {
		t.Errorf("entry lost after restore: %+v", logged)
	}
}

func TestRestoreNameTaken(t *testing.T) {
	ctx := context.Background()
	s := newStore(t)
	today := day("2026-11-06")
	old, _ := s.CreateHabit(ctx, input("Lire", habit.Morning), today)
	if _, err := s.ArchiveHabit(ctx, old.ID, today); err != nil {
		t.Fatal(err)
	}
	if _, err := s.CreateHabit(ctx, input("Lire", habit.Morning), today); err != nil {
		t.Fatal(err)
	}
	if _, err := s.RestoreHabit(ctx, old.ID, today); !errors.Is(err, habit.ErrNameTaken) {
		t.Errorf("restore = %v, want ErrNameTaken", err)
	}
	if _, err := s.RestoreHabit(ctx, 999, today); !errors.Is(err, habit.ErrNotFound) {
		t.Errorf("unknown = %v, want ErrNotFound", err)
	}
}

// The order chosen is the order of the Today screen.
func TestReorderHabits(t *testing.T) {
	ctx := context.Background()
	s := newStore(t)
	today := day("2026-11-06")
	var ids []int64
	for _, name := range []string{"A", "B", "C"} {
		h, err := s.CreateHabit(ctx, input(name, habit.Morning), today)
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, h.ID)
	}
	evening, _ := s.CreateHabit(ctx, input("Soir", habit.Evening), today)

	if err := s.ReorderHabits(ctx, habit.Morning, []int64{ids[2], ids[0], ids[1]}, today); err != nil {
		t.Fatal(err)
	}
	logged, err := s.DayHabits(ctx, today)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, l := range logged {
		names = append(names, l.Habit.Name)
	}
	if want := []string{"C", "A", "B", "Soir"}; !slices.Equal(names, want) {
		t.Errorf("Today order = %v, want %v", names, want)
	}

	// An incomplete list, or a habit of another moment, changes nothing.
	var ve *habit.ValidationError
	for _, bad := range [][]int64{{ids[0], ids[1]}, {ids[0], ids[1], evening.ID}} {
		if err := s.ReorderHabits(ctx, habit.Morning, bad, today); !errors.As(err, &ve) {
			t.Errorf("order %v = %v, want a validation error", bad, err)
		}
	}
	logged, _ = s.DayHabits(ctx, today)
	if logged[0].Habit.Name != "C" {
		t.Errorf("order changed by a refused request: %+v", logged)
	}
}
