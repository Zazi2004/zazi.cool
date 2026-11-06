package store

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"zazi.cool/habits/internal/habit"
)

// uniqueViolation is the PostgreSQL error code of a violated unique index.
const uniqueViolation = "23505"

// CreateHabit stores a validated habit and its first rule, effective from today (the
// civil day of creation). The habit goes last in its moment. It returns
// habit.ErrNameTaken when another active habit has the same name.
func (s *Store) CreateHabit(ctx context.Context, in habit.Input, today time.Time) (habit.Habit, error) {
	h := habit.Habit{
		Name: in.Name, Emoji: in.Emoji, Color: in.Color, Moment: in.Moment, Unit: in.Unit,
		Step: in.Step, Period: in.Period, Target: in.Target, Weekdays: in.Weekdays,
	}
	// BeginFunc commits when the function returns nil and rolls back otherwise, so the
	// habit never exists without its rule.
	err := pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		err := tx.QueryRow(ctx,
			`INSERT INTO habits (name, emoji, color, moment, position, unit, step, created_on)
			 VALUES ($1, $2, $3, $4,
			         (SELECT COALESCE(MAX(position) + 1, 0) FROM habits WHERE moment = $4),
			         $5, $6, $7)
			 RETURNING id, position`,
			in.Name, in.Emoji, string(in.Color), string(in.Moment), in.Unit, in.Step, today,
		).Scan(&h.ID, &h.Position)
		if err != nil {
			return err
		}
		_, err = tx.Exec(ctx,
			`INSERT INTO habit_rules (habit_id, valid_from, period, target, weekdays)
			 VALUES ($1, $2, $3, $4, $5)`,
			h.ID, today, string(in.Period), in.Target, int16(in.Weekdays))
		return err
	})
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == uniqueViolation {
		return habit.Habit{}, habit.ErrNameTaken
	}
	if err != nil {
		return habit.Habit{}, fmt.Errorf("create habit: %w", err)
	}
	return h, nil
}

// ListHabits returns the active and the archived habits as they are on day today
// (each with the rule in force that day). Habits come in moment order, then manual
// order.
func (s *Store) ListHabits(ctx context.Context, today time.Time) (active, archived []habit.Habit, err error) {
	rows, err := s.pool.Query(ctx,
		`SELECT h.id, h.name, h.emoji, h.color, h.moment, h.position, h.unit, h.step,
		        r.period, r.target, r.weekdays, h.archived_on
		 FROM habits h
		 JOIN LATERAL (
		     SELECT period, target, weekdays FROM habit_rules
		     WHERE habit_id = h.id AND valid_from <= $1
		     ORDER BY valid_from DESC LIMIT 1
		 ) r ON true
		 ORDER BY CASE h.moment WHEN 'morning' THEN 0 WHEN 'day' THEN 1 ELSE 2 END,
		          h.position, h.id`, today)
	if err != nil {
		return nil, nil, fmt.Errorf("list habits: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var (
			h                     habit.Habit
			color, moment, period string
			weekdays              int16
		)
		err := rows.Scan(&h.ID, &h.Name, &h.Emoji, &color, &moment, &h.Position, &h.Unit, &h.Step,
			&period, &h.Target, &weekdays, &h.ArchivedOn)
		if err != nil {
			return nil, nil, fmt.Errorf("scan habit: %w", err)
		}
		h.Color, h.Moment, h.Period, h.Weekdays = habit.Color(color), habit.Moment(moment), habit.Period(period), habit.Weekdays(weekdays)
		if h.ArchivedOn != nil && !h.ArchivedOn.After(today) {
			archived = append(archived, h)
		} else {
			active = append(active, h)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, nil, fmt.Errorf("list habits: %w", err)
	}
	return active, archived, nil
}
