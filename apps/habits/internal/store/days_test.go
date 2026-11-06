package store

import (
	"context"
	"errors"
	"testing"

	"zazi.cool/habits/internal/habit"
)

func TestSetEntryAndDayHabits(t *testing.T) {
	ctx := context.Background()
	s := newStore(t)
	today := day("2026-11-06") // a Friday
	vit, err := s.CreateHabit(ctx, input("Vitamine D", habit.Morning), today)
	if err != nil {
		t.Fatal(err)
	}

	values := func() map[int64]int {
		t.Helper()
		got, err := s.DayHabits(ctx, today)
		if err != nil {
			t.Fatal(err)
		}
		m := map[int64]int{}
		for _, l := range got {
			m[l.Habit.ID] = l.Value
		}
		return m
	}

	if got := values(); len(got) != 1 || got[vit.ID] != 0 {
		t.Fatalf("before any entry = %v", got)
	}
	for _, v := range []int{1, 2, 3} {
		if err := s.SetEntry(ctx, vit.ID, today, v); err != nil {
			t.Fatal(err)
		}
		if got := values()[vit.ID]; got != v {
			t.Errorf("after SetEntry(%d) value = %d", v, got)
		}
	}
	if err := s.SetEntry(ctx, vit.ID, today, 0); err != nil {
		t.Fatal(err)
	}
	var n int
	if err := s.pool.QueryRow(ctx, `SELECT count(*) FROM entries`).Scan(&n); err != nil || n != 0 {
		t.Errorf("rows after value 0 = %d (err %v), want the entry deleted", n, err)
	}
}

func TestSetEntryRefusals(t *testing.T) {
	ctx := context.Background()
	s := newStore(t)
	friday := day("2026-11-06")
	weekdaysOnly := input("Pas d'écran", habit.Evening)
	weekdaysOnly.Weekdays = 0b0011111 // Monday to Friday
	h, err := s.CreateHabit(ctx, weekdaysOnly, friday)
	if err != nil {
		t.Fatal(err)
	}

	if err := s.SetEntry(ctx, h.ID, day("2026-11-07"), 1); !errors.Is(err, habit.ErrDayNotAllowed) {
		t.Errorf("Saturday = %v, want ErrDayNotAllowed", err)
	}
	if err := s.SetEntry(ctx, 999, friday, 1); !errors.Is(err, habit.ErrNotFound) {
		t.Errorf("unknown habit = %v, want ErrNotFound", err)
	}
	if err := s.SetEntry(ctx, h.ID, day("2026-11-05"), 1); !errors.Is(err, habit.ErrNotFound) {
		t.Errorf("before creation = %v, want ErrNotFound", err)
	}
	// A Saturday habit is not listed on Friday, a weekday one is.
	got, err := s.DayHabits(ctx, day("2026-11-07"))
	if err != nil || len(got) != 0 {
		t.Errorf("Saturday habits = %v (err %v), want none", got, err)
	}
	got, err = s.DayHabits(ctx, friday)
	if err != nil || len(got) != 1 {
		t.Errorf("Friday habits = %v (err %v), want one", got, err)
	}
}
