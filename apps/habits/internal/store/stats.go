package store

import (
	"context"
	"fmt"
	"time"

	"zazi.cool/habits/internal/habit"
)

// ActiveHistories returns the history of every habit active on today, in moment then
// manual order. Three queries whatever the number of habits.
func (s *Store) ActiveHistories(ctx context.Context, today time.Time) ([]habit.History, error) {
	return s.histories(ctx, today, `WHERE h.archived_on IS NULL OR h.archived_on > $1`, today)
}

// HabitHistory returns the history of one habit, archived or not; habit.ErrNotFound
// when it does not exist. Three queries whatever the age of the habit.
func (s *Store) HabitHistory(ctx context.Context, id int64, today time.Time) (habit.History, error) {
	got, err := s.histories(ctx, today, `WHERE h.id = $1`, id)
	if err != nil {
		return habit.History{}, err
	}
	if len(got) == 0 {
		return habit.History{}, habit.ErrNotFound
	}
	return got[0], nil
}

// histories reads the habits selected by where, then all their rules, then all their
// entries: one query each, joined in memory by habit id. $1 is today.
func (s *Store) histories(ctx context.Context, today time.Time, where string, args ...any) ([]habit.History, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT h.id, h.name, h.emoji, h.color, h.moment, h.position, h.unit, h.step, h.archived_on
		 FROM habits h `+where+`
		 ORDER BY CASE h.moment WHEN 'morning' THEN 0 WHEN 'day' THEN 1 ELSE 2 END,
		          h.position, h.id`, args...)
	if err != nil {
		return nil, fmt.Errorf("histories: habits: %w", err)
	}
	defer rows.Close()
	var out []habit.History
	var ids []int64
	for rows.Next() {
		var (
			h             habit.Habit
			color, moment string
		)
		if err := rows.Scan(&h.ID, &h.Name, &h.Emoji, &color, &moment, &h.Position, &h.Unit, &h.Step, &h.ArchivedOn); err != nil {
			return nil, fmt.Errorf("histories: scan habit: %w", err)
		}
		h.Color, h.Moment = habit.Color(color), habit.Moment(moment)
		out = append(out, habit.History{Habit: h, Entries: map[time.Time]int{}})
		ids = append(ids, h.ID)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("histories: habits: %w", err)
	}
	if len(out) == 0 {
		return nil, nil
	}
	byID := make(map[int64]*habit.History, len(out))
	for i := range out {
		byID[out[i].Habit.ID] = &out[i]
	}

	if err := s.readRules(ctx, ids, byID); err != nil {
		return nil, err
	}
	if err := s.readEntries(ctx, ids, byID); err != nil {
		return nil, err
	}
	for i := range out {
		h := &out[i]
		if len(h.Rules) == 0 {
			return nil, fmt.Errorf("histories: habit %d has no rule", h.Habit.ID)
		}
		// The habit shows the rule in force today (the first one if it starts later).
		rule := h.Rules[0].Rule
		for _, v := range h.Rules {
			if !v.ValidFrom.After(today) {
				rule = v.Rule
			}
		}
		h.Habit.Period, h.Habit.Target, h.Habit.Weekdays = rule.Period, rule.Target, rule.Weekdays
	}
	return out, nil
}

func (s *Store) readRules(ctx context.Context, ids []int64, byID map[int64]*habit.History) error {
	rows, err := s.pool.Query(ctx,
		`SELECT habit_id, valid_from, period, target, weekdays FROM habit_rules
		 WHERE habit_id = ANY($1) ORDER BY habit_id, valid_from`, ids)
	if err != nil {
		return fmt.Errorf("histories: rules: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var (
			id       int64
			v        habit.RuleVersion
			period   string
			weekdays int16
		)
		if err := rows.Scan(&id, &v.ValidFrom, &period, &v.Target, &weekdays); err != nil {
			return fmt.Errorf("histories: scan rule: %w", err)
		}
		v.Period, v.Weekdays = habit.Period(period), habit.Weekdays(weekdays)
		byID[id].Rules = append(byID[id].Rules, v)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("histories: rules: %w", err)
	}
	return nil
}

func (s *Store) readEntries(ctx context.Context, ids []int64, byID map[int64]*habit.History) error {
	rows, err := s.pool.Query(ctx, `SELECT habit_id, day, value FROM entries WHERE habit_id = ANY($1)`, ids)
	if err != nil {
		return fmt.Errorf("histories: entries: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var (
			id    int64
			day   time.Time
			value int
		)
		if err := rows.Scan(&id, &day, &value); err != nil {
			return fmt.Errorf("histories: scan entry: %w", err)
		}
		byID[id].Entries[day] = value
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("histories: entries: %w", err)
	}
	return nil
}
