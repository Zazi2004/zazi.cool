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

// habitSelect reads a habit with the rule in force on day $2 (a lateral join, like
// ListHabits); the caller appends the WHERE clause.
const habitSelect = `SELECT h.id, h.name, h.emoji, h.color, h.moment, h.position, h.unit, h.step,
		        r.period, r.target, r.weekdays, h.archived_on
		 FROM habits h
		 JOIN LATERAL (
		     SELECT period, target, weekdays FROM habit_rules
		     WHERE habit_id = h.id AND valid_from <= $2
		     ORDER BY valid_from DESC LIMIT 1
		 ) r ON true `

// lockHabit reads a habit as it is on today and locks its row until the end of the
// transaction, so two concurrent edits do not interleave. habit.ErrNotFound when it
// does not exist.
func lockHabit(ctx context.Context, tx pgx.Tx, id int64, today time.Time) (habit.Habit, error) {
	var (
		h                     habit.Habit
		color, moment, period string
		weekdays              int16
	)
	err := tx.QueryRow(ctx, habitSelect+`WHERE h.id = $1 FOR UPDATE OF h`, id, today).Scan(
		&h.ID, &h.Name, &h.Emoji, &color, &moment, &h.Position, &h.Unit, &h.Step,
		&period, &h.Target, &weekdays, &h.ArchivedOn)
	if errors.Is(err, pgx.ErrNoRows) {
		return habit.Habit{}, habit.ErrNotFound
	}
	if err != nil {
		return habit.Habit{}, fmt.Errorf("read habit: %w", err)
	}
	h.Color, h.Moment, h.Period, h.Weekdays = habit.Color(color), habit.Moment(moment), habit.Period(period), habit.Weekdays(weekdays)
	return h, nil
}

// isArchived reports whether the habit no longer applies on today.
func isArchived(h habit.Habit, today time.Time) bool {
	return h.ArchivedOn != nil && !h.ArchivedOn.After(today)
}

// nextPosition returns the position after the last habit of the moment.
func nextPosition(ctx context.Context, tx pgx.Tx, moment habit.Moment) (int, error) {
	var pos int
	err := tx.QueryRow(ctx, `SELECT COALESCE(MAX(position) + 1, 0) FROM habits WHERE moment = $1`,
		string(moment)).Scan(&pos)
	return pos, err
}

// inTx runs fn in a transaction and turns a unique-index violation into habit.ErrNameTaken.
func (s *Store) inTx(ctx context.Context, what string, fn func(tx pgx.Tx) error) error {
	err := pgx.BeginFunc(ctx, s.pool, fn)
	var pgErr *pgconn.PgError
	switch {
	case err == nil:
		return nil
	case errors.Is(err, habit.ErrNotFound), errors.As(err, new(*habit.ValidationError)):
		return err
	case errors.As(err, &pgErr) && pgErr.Code == uniqueViolation:
		return habit.ErrNameTaken
	}
	return fmt.Errorf("%s: %w", what, err)
}

// UpdateHabit applies a patch to an active habit. A change of period, target or weekdays
// writes a new rule version from the start of the current period (habit.PlanRuleChange);
// every other attribute applies everywhere. A habit that changes moment goes last in the
// new one. habit.ErrNotFound for an unknown or archived habit, a *habit.ValidationError
// for an invalid result, habit.ErrNameTaken for a name used by another active habit.
func (s *Store) UpdateHabit(ctx context.Context, id int64, p habit.Patch, today time.Time) (habit.Habit, error) {
	var out habit.Habit
	err := s.inTx(ctx, "update habit", func(tx pgx.Tx) error {
		current, err := lockHabit(ctx, tx, id, today)
		if err != nil {
			return err
		}
		if isArchived(current, today) {
			return habit.ErrNotFound
		}
		in, err := p.Apply(current).Validate()
		if err != nil {
			return err
		}
		position := current.Position
		if in.Moment != current.Moment {
			if position, err = nextPosition(ctx, tx, in.Moment); err != nil {
				return err
			}
		}
		_, err = tx.Exec(ctx,
			`UPDATE habits SET name = $2, emoji = $3, color = $4, moment = $5, position = $6, unit = $7, step = $8
			 WHERE id = $1`,
			id, in.Name, in.Emoji, string(in.Color), string(in.Moment), position, in.Unit, in.Step)
		if err != nil {
			return err
		}
		if err := writeRule(ctx, tx, id, habit.Rule{Period: in.Period, Target: in.Target, Weekdays: in.Weekdays}, today); err != nil {
			return err
		}
		out = habit.Habit{
			ID: id, Name: in.Name, Emoji: in.Emoji, Color: in.Color, Moment: in.Moment, Position: position,
			Unit: in.Unit, Step: in.Step, Period: in.Period, Target: in.Target, Weekdays: in.Weekdays,
		}
		return nil
	})
	return out, err
}

// writeRule makes next the rule in force on today, as planned by habit.PlanRuleChange:
// the versions from the start of the period on are replaced, earlier ones are not touched.
func writeRule(ctx context.Context, tx pgx.Tx, id int64, next habit.Rule, today time.Time) error {
	rows, err := tx.Query(ctx,
		`SELECT valid_from, period, target, weekdays FROM habit_rules WHERE habit_id = $1 ORDER BY valid_from`, id)
	if err != nil {
		return err
	}
	rules, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (habit.RuleVersion, error) {
		var (
			v        habit.RuleVersion
			period   string
			weekdays int16
		)
		err := row.Scan(&v.ValidFrom, &period, &v.Target, &weekdays)
		v.Period, v.Weekdays = habit.Period(period), habit.Weekdays(weekdays)
		return v, err
	})
	if err != nil {
		return err
	}
	change := habit.PlanRuleChange(rules, next, today)
	if change.None {
		return nil
	}
	if _, err := tx.Exec(ctx, `DELETE FROM habit_rules WHERE habit_id = $1 AND valid_from >= $2`, id, change.From); err != nil {
		return err
	}
	if !change.Insert {
		return nil
	}
	_, err = tx.Exec(ctx,
		`INSERT INTO habit_rules (habit_id, valid_from, period, target, weekdays) VALUES ($1, $2, $3, $4, $5)`,
		id, change.From, string(next.Period), next.Target, int16(next.Weekdays))
	return err
}

// ArchiveHabit retires a habit from today on (archived_on = today); its history is
// untouched. Archiving an archived habit changes nothing.
func (s *Store) ArchiveHabit(ctx context.Context, id int64, today time.Time) (habit.Habit, error) {
	var h habit.Habit
	err := s.inTx(ctx, "archive habit", func(tx pgx.Tx) error {
		var err error
		if h, err = lockHabit(ctx, tx, id, today); err != nil {
			return err
		}
		if isArchived(h, today) {
			return nil
		}
		h.ArchivedOn = &today
		_, err = tx.Exec(ctx, `UPDATE habits SET archived_on = $2 WHERE id = $1`, id, today)
		return err
	})
	return h, err
}

// RestoreHabit brings an archived habit back, last in its moment. habit.ErrNameTaken when
// an active habit has taken its name meanwhile.
func (s *Store) RestoreHabit(ctx context.Context, id int64, today time.Time) (habit.Habit, error) {
	var h habit.Habit
	err := s.inTx(ctx, "restore habit", func(tx pgx.Tx) error {
		var err error
		if h, err = lockHabit(ctx, tx, id, today); err != nil {
			return err
		}
		if !isArchived(h, today) {
			return nil
		}
		if h.Position, err = nextPosition(ctx, tx, h.Moment); err != nil {
			return err
		}
		h.ArchivedOn = nil
		_, err = tx.Exec(ctx, `UPDATE habits SET archived_on = NULL, position = $2 WHERE id = $1`, id, h.Position)
		return err
	})
	return h, err
}

// ReorderHabits sets the manual order of the active habits of a moment: ids must list
// each of them exactly once (*habit.ValidationError otherwise, and nothing changes).
func (s *Store) ReorderHabits(ctx context.Context, moment habit.Moment, ids []int64, today time.Time) error {
	return s.inTx(ctx, "reorder habits", func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx,
			`SELECT id FROM habits WHERE moment = $1 AND (archived_on IS NULL OR archived_on > $2)
			 ORDER BY id FOR UPDATE`, string(moment), today)
		if err != nil {
			return err
		}
		current, err := pgx.CollectRows(rows, pgx.RowTo[int64])
		if err != nil {
			return err
		}
		if err := habit.ValidateOrder(current, ids); err != nil {
			return err
		}
		for position, id := range ids {
			if _, err := tx.Exec(ctx, `UPDATE habits SET position = $2 WHERE id = $1`, id, position); err != nil {
				return err
			}
		}
		return nil
	})
}
