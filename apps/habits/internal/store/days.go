package store

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"

	"zazi.cool/habits/internal/habit"
)

// DayHabits returns the habits that apply on day (active, with the rule in force that
// day and a weekday that allows it) with the value entered for each, in moment then
// manual order, with the total of each period up to day. One query.
func (s *Store) DayHabits(ctx context.Context, day time.Time) ([]habit.Logged, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT h.id, h.name, h.emoji, h.color, h.moment, h.position, h.unit, h.step,
		        r.period, r.target, r.weekdays, COALESCE(e.value, 0),
		        COALESCE((SELECT sum(value) FROM entries
		                  WHERE habit_id = h.id AND day <= $1 AND day >=
		                      CASE r.period WHEN 'week' THEN $3::date WHEN 'month' THEN $4::date
		                                    WHEN 'year' THEN $5::date ELSE $1 END), 0)
		 FROM habits h
		 JOIN LATERAL (
		     SELECT period, target, weekdays FROM habit_rules
		     WHERE habit_id = h.id AND valid_from <= $1
		     ORDER BY valid_from DESC LIMIT 1
		 ) r ON true
		 LEFT JOIN entries e ON e.habit_id = h.id AND e.day = $1
		 WHERE h.created_on <= $1 AND (h.archived_on IS NULL OR h.archived_on > $1)
		   AND (r.weekdays & (1 << $2::int)) <> 0
		 ORDER BY CASE h.moment WHEN 'morning' THEN 0 WHEN 'day' THEN 1 ELSE 2 END,
		          h.position, h.id`, day, habit.Bit(day.Weekday()),
		habit.PeriodStart(habit.PeriodWeek, day), habit.PeriodStart(habit.PeriodMonth, day),
		habit.PeriodStart(habit.PeriodYear, day))
	if err != nil {
		return nil, fmt.Errorf("day habits: %w", err)
	}
	defer rows.Close()

	var out []habit.Logged
	for rows.Next() {
		var (
			l                     habit.Logged
			color, moment, period string
			weekdays              int16
		)
		h := &l.Habit
		err := rows.Scan(&h.ID, &h.Name, &h.Emoji, &color, &moment, &h.Position, &h.Unit, &h.Step,
			&period, &h.Target, &weekdays, &l.Value, &l.Total)
		if err != nil {
			return nil, fmt.Errorf("scan day habit: %w", err)
		}
		h.Color, h.Moment, h.Period, h.Weekdays = habit.Color(color), habit.Moment(moment), habit.Period(period), habit.Weekdays(weekdays)
		out = append(out, l)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("day habits: %w", err)
	}
	return out, nil
}

// SetEntry records value for the habit on day; 0 deletes the entry. It returns
// habit.ErrNotFound when the habit does not exist that day and habit.ErrDayNotAllowed
// when its weekdays exclude the day.
func (s *Store) SetEntry(ctx context.Context, habitID int64, day time.Time, value int) error {
	var weekdays int16
	err := s.pool.QueryRow(ctx,
		`SELECT r.weekdays
		 FROM habits h
		 JOIN LATERAL (
		     SELECT weekdays FROM habit_rules
		     WHERE habit_id = h.id AND valid_from <= $2
		     ORDER BY valid_from DESC LIMIT 1
		 ) r ON true
		 WHERE h.id = $1 AND h.created_on <= $2 AND (h.archived_on IS NULL OR h.archived_on > $2)`,
		habitID, day).Scan(&weekdays)
	if errors.Is(err, pgx.ErrNoRows) {
		return habit.ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("set entry: %w", err)
	}
	if !habit.Weekdays(weekdays).Has(day.Weekday()) {
		return habit.ErrDayNotAllowed
	}
	if value == 0 {
		_, err = s.pool.Exec(ctx, `DELETE FROM entries WHERE habit_id = $1 AND day = $2`, habitID, day)
	} else {
		_, err = s.pool.Exec(ctx,
			`INSERT INTO entries (habit_id, day, value) VALUES ($1, $2, $3)
			 ON CONFLICT (habit_id, day) DO UPDATE SET value = EXCLUDED.value, updated_at = now()`,
			habitID, day, value)
	}
	if err != nil {
		return fmt.Errorf("set entry: %w", err)
	}
	return nil
}

// DayStats sums up the `day` habits of every civil day from the earliest habit (or
// from `from` if that is earlier) to `to`, oldest first: the week strip and the
// streak come from it. One query; days before any habit existed are not listed.
func (s *Store) DayStats(ctx context.Context, from, to time.Time) ([]habit.DayStat, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT d.day, count(x.target), count(*) FILTER (WHERE x.value >= x.target),
		        COALESCE(sum(LEAST(1, x.value::float8 / x.target)), 0)
		 FROM generate_series(
		          LEAST($1::date, (SELECT COALESCE(MIN(created_on), $1::date) FROM habits)),
		          $2::date, interval '1 day') AS g(d)
		 CROSS JOIN LATERAL (SELECT g.d::date AS day) d
		 LEFT JOIN LATERAL (
		     SELECT r.target, COALESCE(e.value, 0) AS value
		     FROM habits h
		     JOIN LATERAL (
		         SELECT period, target, weekdays FROM habit_rules
		         WHERE habit_id = h.id AND valid_from <= d.day
		         ORDER BY valid_from DESC LIMIT 1
		     ) r ON true
		     LEFT JOIN entries e ON e.habit_id = h.id AND e.day = d.day
		     WHERE h.created_on <= d.day AND (h.archived_on IS NULL OR h.archived_on > d.day)
		       AND r.period = 'day'
		       AND (r.weekdays & (1 << (extract(isodow FROM d.day)::int - 1))) <> 0
		 ) x ON true
		 GROUP BY d.day ORDER BY d.day`, from, to)
	if err != nil {
		return nil, fmt.Errorf("day stats: %w", err)
	}
	defer rows.Close()

	var out []habit.DayStat
	for rows.Next() {
		var st habit.DayStat
		if err := rows.Scan(&st.Day, &st.Total, &st.Done, &st.ProgressSum); err != nil {
			return nil, fmt.Errorf("scan day stat: %w", err)
		}
		out = append(out, st)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("day stats: %w", err)
	}
	return out, nil
}

// DayNote returns the note of day, empty when there is none.
func (s *Store) DayNote(ctx context.Context, day time.Time) (string, error) {
	var text string
	err := s.pool.QueryRow(ctx, `SELECT text FROM day_notes WHERE day = $1`, day).Scan(&text)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("day note: %w", err)
	}
	return text, nil
}

// SetNote saves the note of day; an empty text deletes it.
func (s *Store) SetNote(ctx context.Context, day time.Time, text string) error {
	var err error
	if text == "" {
		_, err = s.pool.Exec(ctx, `DELETE FROM day_notes WHERE day = $1`, day)
	} else {
		_, err = s.pool.Exec(ctx,
			`INSERT INTO day_notes (day, text) VALUES ($1, $2)
			 ON CONFLICT (day) DO UPDATE SET text = EXCLUDED.text, updated_at = now()`, day, text)
	}
	if err != nil {
		return fmt.Errorf("set note: %w", err)
	}
	return nil
}
