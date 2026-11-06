package store

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"zazi.cool/habits/internal/habit"
)

// queryCounter counts the statements a pool sends.
type queryCounter struct{ n atomic.Int32 }

func (c *queryCounter) TraceQueryStart(ctx context.Context, _ *pgx.Conn, _ pgx.TraceQueryStartData) context.Context {
	c.n.Add(1)
	return ctx
}

func (c *queryCounter) TraceQueryEnd(context.Context, *pgx.Conn, pgx.TraceQueryEndData) {}

// countingStore returns a store on the same schema as s whose queries are counted.
func countingStore(t *testing.T, s *Store) (*Store, *queryCounter) {
	t.Helper()
	counter := &queryCounter{}
	cfg := s.pool.Config()
	cfg.ConnConfig.Tracer = counter
	pool, err := pgxpool.NewWithConfig(context.Background(), cfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	return New(pool), counter
}

func TestHabitHistory(t *testing.T) {
	ctx := context.Background()
	s := newStore(t)
	today := day("2026-11-06")

	// A weekly habit created on 12 October, switched to monthly on 6 November (CA5).
	in := input("Course", habit.Evening)
	in.Period, in.Target, in.Unit = habit.PeriodWeek, 3, "sorties"
	course, err := s.CreateHabit(ctx, in, day("2026-10-12"))
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range []struct {
		day   string
		value int
	}{{"2026-10-13", 2}, {"2026-10-15", 1}, {"2026-10-28", 3}, {"2026-11-03", 5}} {
		if err := s.SetEntry(ctx, course.ID, day(e.day), e.value); err != nil {
			t.Fatal(err)
		}
	}
	month := habit.PeriodMonth
	if _, err := s.UpdateHabit(ctx, course.ID, habit.Patch{Period: &month}, today); err != nil {
		t.Fatal(err)
	}
	other, err := s.CreateHabit(ctx, input("Vitamine D", habit.Morning), day("2026-11-01"))
	if err != nil {
		t.Fatal(err)
	}
	if err := s.SetEntry(ctx, other.ID, day("2026-11-02"), 2); err != nil {
		t.Fatal(err)
	}

	counted, queries := countingStore(t, s)
	h, err := counted.HabitHistory(ctx, course.ID, today)
	if err != nil {
		t.Fatal(err)
	}
	if got := queries.n.Load(); got != 3 {
		t.Errorf("HabitHistory ran %d queries, want 3", got)
	}
	if h.Habit.Name != "Course" || h.Habit.Period != habit.PeriodMonth || len(h.Rules) != 2 || len(h.Entries) != 4 {
		t.Errorf("history = %+v", h)
	}
	periods := h.Periods(today)
	want := []habit.Status{
		habit.StatusSuccess, habit.StatusMissed, habit.StatusInterrupted, habit.StatusSuccess,
	}
	if len(periods) != len(want) {
		t.Fatalf("periods = %+v", periods)
	}
	for i, p := range periods {
		if p.Status != want[i] {
			t.Errorf("period %d (%s) = %s, want %s", i, habit.FormatDay(p.Start), p.Status, want[i])
		}
	}
	if cut := periods[2]; habit.FormatDay(cut.Start) != "2026-10-26" || habit.FormatDay(cut.End) != "2026-10-31" {
		t.Errorf("interrupted period = %v to %v, want 26 to 31 October", cut.Start, cut.End)
	}

	queries.n.Store(0)
	all, err := counted.ActiveHistories(ctx, today)
	if err != nil {
		t.Fatal(err)
	}
	if got := queries.n.Load(); got != 3 {
		t.Errorf("ActiveHistories ran %d queries, want 3", got)
	}
	// Moment order: the morning habit first.
	if len(all) != 2 || all[0].Habit.Name != "Vitamine D" || all[1].Habit.Name != "Course" {
		t.Errorf("active histories = %+v", all)
	}
}

func TestHabitHistoryArchivedAndUnknown(t *testing.T) {
	ctx := context.Background()
	s := newStore(t)
	today := day("2026-11-06")
	h, err := s.CreateHabit(ctx, input("Piano", habit.Day), day("2026-11-01"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.ArchiveHabit(ctx, h.ID, today); err != nil {
		t.Fatal(err)
	}
	if active, err := s.ActiveHistories(ctx, today); err != nil || len(active) != 0 {
		t.Errorf("active histories = %d (err %v), want none: the habit is archived", len(active), err)
	}
	got, err := s.HabitHistory(ctx, h.ID, today)
	if err != nil || got.Habit.ArchivedOn == nil {
		t.Errorf("archived history = %+v (err %v), want it readable", got.Habit, err)
	}
	if _, err := s.HabitHistory(ctx, 999, today); !errors.Is(err, habit.ErrNotFound) {
		t.Errorf("unknown habit error = %v, want ErrNotFound", err)
	}
}
