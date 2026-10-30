package store

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgconn"

	"zazi.cool/habits/internal/habit"
)

// uniqueViolation is the PostgreSQL error code of a broken unique index.
const uniqueViolation = "23505"

// currentSchedule is the weekday mask in force on the day $1 (the latest version
// that started on or before it). Past days keep the rules they had (NF5).
const currentSchedule = `(SELECT s.weekdays FROM habit_schedules s
	WHERE s.habit_id = h.id AND s.valid_from <= $1 ORDER BY s.valid_from DESC LIMIT 1)`

func isNameTaken(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == uniqueViolation
}

// CreateHabit adds an active habit that exists from today on.
func (s *Store) CreateHabit(ctx context.Context, name string, days habit.Weekdays, today time.Time) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin: %w", err)
	}
	// After Commit, Rollback is a no-op that returns an error nobody can act on.
	defer func() { _ = tx.Rollback(ctx) }()

	var id int64
	err = tx.QueryRow(ctx,
		`INSERT INTO habits (name, created_on) VALUES ($1, $2) RETURNING id`, name, today).Scan(&id)
	if isNameTaken(err) {
		return habit.ErrNameTaken
	}
	if err != nil {
		return fmt.Errorf("insert habit: %w", err)
	}
	if _, err := tx.Exec(ctx,
		`INSERT INTO habit_schedules (habit_id, valid_from, weekdays) VALUES ($1, $2, $3)`,
		id, today, int16(days)); err != nil {
		return fmt.Errorf("insert schedule: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit: %w", err)
	}
	return nil
}

// UpdateHabit renames a habit and, if the weekdays changed, starts a new schedule
// version today: earlier days keep counting with the old one (F2).
func (s *Store) UpdateHabit(ctx context.Context, id int64, name string, days habit.Weekdays, today time.Time) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	tag, err := tx.Exec(ctx, `UPDATE habits SET name = $2 WHERE id = $1`, id, name)
	if isNameTaken(err) {
		return habit.ErrNameTaken
	}
	if err != nil {
		return fmt.Errorf("update habit: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return habit.ErrNotFound
	}

	var current int16
	err = tx.QueryRow(ctx,
		`SELECT weekdays FROM habit_schedules WHERE habit_id = $1 AND valid_from <= $2
		 ORDER BY valid_from DESC LIMIT 1`, id, today).Scan(&current)
	if err != nil {
		return fmt.Errorf("read schedule: %w", err)
	}
	if habit.Weekdays(current) != days {
		// Upsert: editing twice on the same day replaces that day's version.
		if _, err := tx.Exec(ctx,
			`INSERT INTO habit_schedules (habit_id, valid_from, weekdays) VALUES ($1, $2, $3)
			 ON CONFLICT (habit_id, valid_from) DO UPDATE SET weekdays = EXCLUDED.weekdays`,
			id, today, int16(days)); err != nil {
			return fmt.Errorf("write schedule: %w", err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit: %w", err)
	}
	return nil
}

// ArchiveHabit hides a habit from today on; archiving twice keeps the first date.
func (s *Store) ArchiveHabit(ctx context.Context, id int64, today time.Time) error {
	tag, err := s.pool.Exec(ctx,
		`UPDATE habits SET archived_on = COALESCE(archived_on, $2) WHERE id = $1`, id, today)
	if err != nil {
		return fmt.Errorf("archive habit: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return habit.ErrNotFound
	}
	return nil
}

// RestoreHabit makes an archived habit active again. It fails with ErrNameTaken if
// another active habit took its name meanwhile.
func (s *Store) RestoreHabit(ctx context.Context, id int64) error {
	tag, err := s.pool.Exec(ctx, `UPDATE habits SET archived_on = NULL WHERE id = $1`, id)
	if isNameTaken(err) {
		return habit.ErrNameTaken
	}
	if err != nil {
		return fmt.Errorf("restore habit: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return habit.ErrNotFound
	}
	return nil
}

// ListHabits returns every habit, active ones first, with the weekdays in force today.
func (s *Store) ListHabits(ctx context.Context, today time.Time) ([]habit.Habit, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT h.id, h.name, h.created_on, h.archived_on, `+currentSchedule+`
		 FROM habits h ORDER BY h.archived_on IS NOT NULL, lower(h.name), h.id`, today)
	if err != nil {
		return nil, fmt.Errorf("list habits: %w", err)
	}
	defer rows.Close()

	var habits []habit.Habit
	for rows.Next() {
		var h habit.Habit
		var days *int16
		if err := rows.Scan(&h.ID, &h.Name, &h.CreatedOn, &h.ArchivedOn, &days); err != nil {
			return nil, fmt.Errorf("scan habit: %w", err)
		}
		if days != nil {
			h.Weekdays = habit.Weekdays(*days)
		}
		habits = append(habits, h)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list habits: %w", err)
	}
	return habits, nil
}

// dayItemsSQL selects the habits that apply to day $1: created by then, not yet
// archived, and scheduled on its weekday ($2, a mask bit position). $3 restricts the
// result to one habit, or 0 for all.
const dayItemsSQL = `SELECT h.id, h.name, c.habit_id IS NOT NULL
	FROM habits h
	LEFT JOIN completions c ON c.habit_id = h.id AND c.day = $1
	WHERE h.created_on <= $1
	  AND (h.archived_on IS NULL OR $1 < h.archived_on)
	  AND (COALESCE(` + currentSchedule + `, 0) >> $2) & 1 = 1
	  AND ($3 = 0 OR h.id = $3)
	ORDER BY lower(h.name), h.id`

func (s *Store) dayItems(ctx context.Context, day time.Time, habitID int64) ([]habit.DayItem, error) {
	rows, err := s.pool.Query(ctx, dayItemsSQL, day, int32(habit.Bit(day.Weekday())), habitID)
	if err != nil {
		return nil, fmt.Errorf("list day habits: %w", err)
	}
	defer rows.Close()

	var items []habit.DayItem
	for rows.Next() {
		var it habit.DayItem
		if err := rows.Scan(&it.HabitID, &it.Name, &it.Done); err != nil {
			return nil, fmt.Errorf("scan day habit: %w", err)
		}
		items = append(items, it)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list day habits: %w", err)
	}
	return items, nil
}

// DayHabits returns the habits that apply to day, with their state on that day.
func (s *Store) DayHabits(ctx context.Context, day time.Time) ([]habit.DayItem, error) {
	return s.dayItems(ctx, day, 0)
}

// SetDone checks or unchecks a habit on a day. It refuses a day the habit does not
// apply to, so no completion is ever recorded for a day that does not count.
func (s *Store) SetDone(ctx context.Context, habitID int64, day time.Time, done bool) error {
	items, err := s.dayItems(ctx, day, habitID)
	if err != nil {
		return err
	}
	if len(items) == 0 {
		var exists bool
		if err := s.pool.QueryRow(ctx,
			`SELECT EXISTS (SELECT 1 FROM habits WHERE id = $1)`, habitID).Scan(&exists); err != nil {
			return fmt.Errorf("check habit: %w", err)
		}
		if !exists {
			return habit.ErrNotFound
		}
		return habit.ErrNotScheduled
	}

	if done {
		_, err = s.pool.Exec(ctx,
			`INSERT INTO completions (habit_id, day) VALUES ($1, $2) ON CONFLICT DO NOTHING`, habitID, day)
	} else {
		_, err = s.pool.Exec(ctx, `DELETE FROM completions WHERE habit_id = $1 AND day = $2`, habitID, day)
	}
	if err != nil {
		return fmt.Errorf("set completion: %w", err)
	}
	return nil
}

// History returns what the statistics need for the habits that existed at some point
// of [from, to]: their schedule versions and the days they were done in that range.
// It takes two queries whatever the range, so a year costs no more than a week (NF6).
func (s *Store) History(ctx context.Context, from, to time.Time) ([]habit.History, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT h.id, h.name, h.created_on, h.archived_on, s.valid_from, s.weekdays
		 FROM habits h JOIN habit_schedules s ON s.habit_id = h.id
		 WHERE h.created_on <= $2 AND (h.archived_on IS NULL OR h.archived_on > $1)
		 ORDER BY lower(h.name), h.id, s.valid_from`, from, to)
	if err != nil {
		return nil, fmt.Errorf("list schedules: %w", err)
	}
	defer rows.Close()

	var out []habit.History
	byID := map[int64]int{}
	for rows.Next() {
		var h habit.History
		var v habit.ScheduleVersion
		var days int16
		if err := rows.Scan(&h.ID, &h.Name, &h.CreatedOn, &h.ArchivedOn, &v.ValidFrom, &days); err != nil {
			return nil, fmt.Errorf("scan schedule: %w", err)
		}
		v.Weekdays = habit.Weekdays(days)
		// Rows come grouped by habit: the first one creates the entry.
		i, seen := byID[h.ID]
		if !seen {
			h.Done = map[time.Time]bool{}
			out = append(out, h)
			i = len(out) - 1
			byID[h.ID] = i
		}
		out[i].Schedules = append(out[i].Schedules, v)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list schedules: %w", err)
	}

	done, err := s.pool.Query(ctx,
		`SELECT habit_id, day FROM completions WHERE day BETWEEN $1 AND $2`, from, to)
	if err != nil {
		return nil, fmt.Errorf("list completions: %w", err)
	}
	defer done.Close()
	for done.Next() {
		var id int64
		var day time.Time
		if err := done.Scan(&id, &day); err != nil {
			return nil, fmt.Errorf("scan completion: %w", err)
		}
		if i, ok := byID[id]; ok {
			out[i].Done[day] = true
		}
	}
	if err := done.Err(); err != nil {
		return nil, fmt.Errorf("list completions: %w", err)
	}
	return out, nil
}
