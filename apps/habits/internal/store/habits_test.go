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

func newStore(t *testing.T) *Store {
	t.Helper()
	pool := dbtest.New(t)
	if err := migrate.Up(context.Background(), pool, migrations.FS); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return New(pool)
}

func day(s string) time.Time {
	d, err := habit.ParseDay(s)
	if err != nil {
		panic(err)
	}
	return d
}

func input(name string, moment habit.Moment) habit.Input {
	return habit.Input{
		Name: name, Emoji: "💊", Color: "amber", Moment: moment, Unit: "pilules", Step: 1,
		Period: habit.PeriodDay, Target: 2, Weekdays: habit.AllDays,
	}
}

func TestCreateAndListHabits(t *testing.T) {
	ctx := context.Background()
	s := newStore(t)
	today := day("2026-11-06")

	created, err := s.CreateHabit(ctx, input("Vitamine D", habit.Morning), today)
	if err != nil {
		t.Fatal(err)
	}
	if created.ID == 0 || created.Position != 0 {
		t.Errorf("created = %+v, want an id and position 0", created)
	}

	active, archived, err := s.ListHabits(ctx, today)
	if err != nil {
		t.Fatal(err)
	}
	if len(archived) != 0 || len(active) != 1 {
		t.Fatalf("active = %d, archived = %d, want 1 and 0", len(active), len(archived))
	}
	got := active[0]
	if got.ID != created.ID || got.Name != "Vitamine D" || got.Emoji != "💊" || got.Color != "amber" ||
		got.Moment != habit.Morning || got.Unit != "pilules" || got.Step != 1 ||
		got.Period != habit.PeriodDay || got.Target != 2 || got.Weekdays != habit.AllDays || got.ArchivedOn != nil {
		t.Errorf("listed = %+v", got)
	}
}

// The first rule starts on the day of creation (spec §4): before it, the habit has no rule.
func TestFirstRuleStartsOnCreationDay(t *testing.T) {
	ctx := context.Background()
	s := newStore(t)
	if _, err := s.CreateHabit(ctx, input("Lire", habit.Day), day("2026-11-06")); err != nil {
		t.Fatal(err)
	}
	var validFrom time.Time
	if err := s.pool.QueryRow(ctx, `SELECT valid_from FROM habit_rules`).Scan(&validFrom); err != nil {
		t.Fatal(err)
	}
	if got := habit.FormatDay(validFrom); got != "2026-11-06" {
		t.Errorf("valid_from = %s, want 2026-11-06", got)
	}
}

func TestListHabitsOrder(t *testing.T) {
	ctx := context.Background()
	s := newStore(t)
	today := day("2026-11-06")
	for _, h := range []struct {
		name   string
		moment habit.Moment
	}{
		{"Soir 1", habit.Evening}, {"Matin 1", habit.Morning}, {"Journée 1", habit.Day},
		{"Matin 2", habit.Morning}, {"Soir 2", habit.Evening},
	} {
		if _, err := s.CreateHabit(ctx, input(h.name, h.moment), today); err != nil {
			t.Fatal(err)
		}
	}
	active, _, err := s.ListHabits(ctx, today)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, h := range active {
		names = append(names, h.Name)
	}
	want := []string{"Matin 1", "Matin 2", "Journée 1", "Soir 1", "Soir 2"}
	if len(names) != len(want) {
		t.Fatalf("names = %v, want %v", names, want)
	}
	for i := range want {
		if names[i] != want[i] {
			t.Fatalf("names = %v, want %v", names, want)
		}
	}
	if active[1].Position != 1 {
		t.Errorf("second morning habit has position %d, want 1", active[1].Position)
	}
}

func TestCreateHabitNameTaken(t *testing.T) {
	ctx := context.Background()
	s := newStore(t)
	today := day("2026-11-06")
	if _, err := s.CreateHabit(ctx, input("Vitamine D", habit.Morning), today); err != nil {
		t.Fatal(err)
	}
	_, err := s.CreateHabit(ctx, input("Vitamine D", habit.Evening), today)
	if !errors.Is(err, habit.ErrNameTaken) {
		t.Fatalf("second create error = %v, want ErrNameTaken", err)
	}
	// The failed create left nothing behind (habit and rule are in one transaction).
	var n int
	if err := s.pool.QueryRow(ctx, `SELECT count(*) FROM habit_rules`).Scan(&n); err != nil || n != 1 {
		t.Errorf("rules = %d, %v; want 1", n, err)
	}

	// An archived habit does not hold its name.
	if _, err := s.pool.Exec(ctx, `UPDATE habits SET archived_on = $1`, today); err != nil {
		t.Fatal(err)
	}
	if _, err := s.CreateHabit(ctx, input("Vitamine D", habit.Morning), today); err != nil {
		t.Errorf("create with the name of an archived habit: %v", err)
	}
	active, archived, err := s.ListHabits(ctx, today)
	if err != nil || len(active) != 1 || len(archived) != 1 {
		t.Errorf("active = %d, archived = %d, err = %v; want 1 and 1", len(active), len(archived), err)
	}
}
