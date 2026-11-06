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

func TestDayStatsAndStreak(t *testing.T) {
	ctx := context.Background()
	s := newStore(t)
	start := day("2026-10-25")
	today := day("2026-11-06")
	water, err := s.CreateHabit(ctx, input("Eau", habit.Day), start)
	if err != nil {
		t.Fatal(err)
	}
	two := input("Vitamine D", habit.Morning)
	two.Target, two.Unit = 2, "pilules"
	vit, err := s.CreateHabit(ctx, two, start)
	if err != nil {
		t.Fatal(err)
	}
	// 12 complete days up to yesterday (25 Oct → 5 Nov), today untouched.
	for d := start; d.Before(today); d = d.AddDate(0, 0, 1) {
		if err := s.SetEntry(ctx, water.ID, d, 2); err != nil {
			t.Fatal(err)
		}
		if err := s.SetEntry(ctx, vit.ID, d, 3); err != nil {
			t.Fatal(err)
		}
	}
	streak := func() int {
		t.Helper()
		stats, err := s.DayStats(ctx, habit.WeekDays(today)[0], today)
		if err != nil {
			t.Fatal(err)
		}
		return habit.Streak(stats, today)
	}
	if got := streak(); got != 12 {
		t.Errorf("streak with today incomplete = %d, want 12", got)
	}
	for _, id := range []int64{water.ID, vit.ID} {
		if err := s.SetEntry(ctx, id, today, 2); err != nil {
			t.Fatal(err)
		}
	}
	if got := streak(); got != 13 {
		t.Errorf("streak with today complete = %d, want 13", got)
	}

	// Half of the vitamins one day: the mean progress is (1 + 0.5) / 2.
	if err := s.SetEntry(ctx, vit.ID, today, 1); err != nil {
		t.Fatal(err)
	}
	stats, err := s.DayStats(ctx, today, today)
	if err != nil {
		t.Fatal(err)
	}
	last := stats[len(stats)-1]
	if last.Total != 2 || last.Done != 1 || last.Progress() != 0.75 {
		t.Errorf("today = %+v, want 1 of 2 done, progress 0.75", last)
	}
}

func TestDayStatsSkipsDaysWithoutHabit(t *testing.T) {
	ctx := context.Background()
	s := newStore(t)
	// Monday → Friday only: the weekend is not due, so it neither breaks nor adds to the streak.
	weekdays := input("Pas d'écran", habit.Evening)
	weekdays.Weekdays = 0b0011111
	h, err := s.CreateHabit(ctx, weekdays, day("2026-10-26"))
	if err != nil {
		t.Fatal(err)
	}
	for _, d := range []string{"2026-10-26", "2026-10-27", "2026-10-28", "2026-10-29", "2026-10-30", "2026-11-02"} {
		if err := s.SetEntry(ctx, h.ID, day(d), 2); err != nil {
			t.Fatal(err)
		}
	}
	today := day("2026-11-03") // Tuesday, not done yet
	stats, err := s.DayStats(ctx, habit.WeekDays(today)[0], today)
	if err != nil {
		t.Fatal(err)
	}
	if got := habit.Streak(stats, today); got != 6 {
		t.Errorf("streak = %d, want 6 (weekend skipped)", got)
	}
}

func TestDayNote(t *testing.T) {
	ctx := context.Background()
	s := newStore(t)
	d := day("2026-11-06")
	if got, err := s.DayNote(ctx, d); err != nil || got != "" {
		t.Fatalf("no note = %q (err %v)", got, err)
	}
	for _, text := range []string{"première", "remplacée"} {
		if err := s.SetNote(ctx, d, text); err != nil {
			t.Fatal(err)
		}
		if got, _ := s.DayNote(ctx, d); got != text {
			t.Errorf("note = %q, want %q", got, text)
		}
	}
	if err := s.SetNote(ctx, d, ""); err != nil {
		t.Fatal(err)
	}
	var n int
	if err := s.pool.QueryRow(ctx, `SELECT count(*) FROM day_notes`).Scan(&n); err != nil || n != 0 {
		t.Errorf("rows after empty note = %d (err %v), want it deleted", n, err)
	}
}
