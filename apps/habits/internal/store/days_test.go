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

func TestDayHabitsPeriodTotals(t *testing.T) {
	ctx := context.Background()
	s := newStore(t)
	created := day("2026-01-01")

	create := func(name string, period habit.Period, target int, weekdays habit.Weekdays) habit.Habit {
		t.Helper()
		in := input(name, habit.Day)
		in.Period, in.Target, in.Weekdays = period, target, weekdays
		h, err := s.CreateHabit(ctx, in, created)
		if err != nil {
			t.Fatal(err)
		}
		return h
	}
	sport := create("Sport", habit.PeriodYear, 150, habit.AllDays)
	run := create("Course", habit.PeriodWeek, 3, habit.AllDays)
	read := create("Lecture", habit.PeriodMonth, 10, habit.AllDays)
	pills := create("Pilules", habit.PeriodDay, 2, habit.AllDays)

	entries := []struct {
		h     habit.Habit
		day   string
		value int
	}{
		{sport, "2025-12-31", 99}, // previous year: ignored
		{sport, "2026-02-10", 100}, {sport, "2026-11-06", 28},
		{run, "2026-11-01", 5}, // the Sunday before the week
		{run, "2026-11-02", 1}, {run, "2026-11-05", 2},
		{read, "2026-10-31", 7}, {read, "2026-11-03", 4},
		{pills, "2026-11-05", 2},
	}
	for _, e := range entries {
		if _, err := s.pool.Exec(ctx, `INSERT INTO entries (habit_id, day, value) VALUES ($1, $2, $3)`,
			e.h.ID, day(e.day), e.value); err != nil {
			t.Fatal(err)
		}
	}

	got, err := s.DayHabits(ctx, day("2026-11-06"))
	if err != nil {
		t.Fatal(err)
	}
	type tv struct{ value, total int }
	have := map[int64]tv{}
	for _, l := range got {
		have[l.Habit.ID] = tv{l.Value, l.Total}
	}
	want := map[int64]tv{
		sport.ID: {28, 128}, // the previous year's entry is not counted
		run.ID:   {0, 3},    // Monday to Friday of the week only
		read.ID:  {0, 4},    // November only
		pills.ID: {0, 0},    // `day`: yesterday does not count
	}
	for id, w := range want {
		if have[id] != w {
			t.Errorf("habit %d = %+v, want %+v", id, have[id], w)
		}
	}

	// A habit limited to weekdays is absent on Saturday (CA6), whatever its period.
	weekdaysOnly := create("Footing", habit.PeriodWeek, 3, 0b0011111)
	got, err = s.DayHabits(ctx, day("2026-11-07"))
	if err != nil {
		t.Fatal(err)
	}
	for _, l := range got {
		if l.Habit.ID == weekdaysOnly.ID {
			t.Error("weekday-only habit listed on a Saturday")
		}
	}
}
