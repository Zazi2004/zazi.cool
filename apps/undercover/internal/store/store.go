// Package store persists word pairs and reports in PostgreSQL.
package store

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"zazi.cool/undercover/internal/pairs"
	"zazi.cool/undercover/internal/seed"
)

var (
	// ErrNotFound means no pair matches the request.
	ErrNotFound = errors.New("pair not found")
	// ErrDuplicate means a pair with the same normalized words already exists.
	ErrDuplicate = errors.New("duplicate pair")
)

// uniqueViolation is the PostgreSQL error code raised by a UNIQUE constraint.
const uniqueViolation = "23505"

// Store is safe for concurrent use: the pool hands out one connection per call.
type Store struct {
	pool *pgxpool.Pool
}

// New wraps an open connection pool.
func New(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

// Ping checks that the database is reachable.
func (s *Store) Ping(ctx context.Context) error {
	return s.pool.Ping(ctx)
}

// ExtremeMode tells whether extreme pairs can be drawn.
type ExtremeMode int

const (
	// ExtremeExcluded draws all-ages pairs only (the zero value, so the default).
	ExtremeExcluded ExtremeMode = iota
	// ExtremeIncluded draws all-ages and extreme pairs.
	ExtremeIncluded
	// ExtremeOnly draws extreme pairs only.
	ExtremeOnly
)

// Filter restricts which pairs can be drawn.
type Filter struct {
	Extreme         ExtremeMode
	SuggestionsOnly bool
	// Exclude lists pairs already played on this device.
	Exclude []int64
}

const pairColumns = "id, word_a, word_b, extreme, hidden, source"

// RandomPair draws a visible pair matching f. When every matching pair has been
// excluded, it draws again without the exclusion list rather than failing.
func (s *Store) RandomPair(ctx context.Context, f Filter) (pairs.Pair, error) {
	p, err := s.randomPair(ctx, f)
	if errors.Is(err, ErrNotFound) && len(f.Exclude) > 0 {
		f.Exclude = nil
		return s.randomPair(ctx, f)
	}
	return p, err
}

func (s *Store) randomPair(ctx context.Context, f Filter) (pairs.Pair, error) {
	exclude := f.Exclude
	if exclude == nil {
		// A NULL array would make "id <> ALL(...)" NULL and filter out every row.
		exclude = []int64{}
	}
	row := s.pool.QueryRow(ctx, `
		SELECT `+pairColumns+` FROM word_pairs
		WHERE NOT hidden
		  AND (NOT extreme OR $1)
		  AND (extreme OR NOT $2)
		  AND (NOT $3 OR source = 'suggestion')
		  AND id <> ALL($4)
		ORDER BY random()
		LIMIT 1`,
		f.Extreme != ExtremeExcluded, f.Extreme == ExtremeOnly, f.SuggestionsOnly, exclude)
	return scanPair(row)
}

// Pair returns the pair with the given id, hidden or not.
func (s *Store) Pair(ctx context.Context, id int64) (pairs.Pair, error) {
	row := s.pool.QueryRow(ctx, "SELECT "+pairColumns+" FROM word_pairs WHERE id = $1", id)
	return scanPair(row)
}

// AllPairs returns every pair, hidden ones included, for redundancy checks.
func (s *Store) AllPairs(ctx context.Context) ([]pairs.Pair, error) {
	rows, err := s.pool.Query(ctx, "SELECT "+pairColumns+" FROM word_pairs ORDER BY id")
	if err != nil {
		return nil, fmt.Errorf("query pairs: %w", err)
	}
	// CollectRows iterates, scans each row with the given function and closes rows.
	ps, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (pairs.Pair, error) {
		return scanPair(row)
	})
	if err != nil {
		return nil, fmt.Errorf("scan pairs: %w", err)
	}
	return ps, nil
}

// CreatePair stores a new pair. It returns ErrDuplicate if the same words exist.
func (s *Store) CreatePair(ctx context.Context, c pairs.Candidate, extreme bool, source pairs.Source) (pairs.Pair, error) {
	row := s.pool.QueryRow(ctx, `
		INSERT INTO word_pairs (word_a, word_b, norm_a, norm_b, extreme, source)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING `+pairColumns,
		c.WordA, c.WordB, c.NormA, c.NormB, extreme, source)
	p, err := scanPair(row)
	if pgErr, ok := errors.AsType[*pgconn.PgError](err); ok && pgErr.Code == uniqueViolation {
		return pairs.Pair{}, ErrDuplicate
	}
	return p, err
}

// Report records a report and applies its effect at once, in one transaction.
// Reporting the same pair twice for the same reason from the same reporter is a no-op.
func (s *Store) Report(ctx context.Context, pairID int64, reason pairs.Reason, reporterHash string) error {
	var update string
	switch reason {
	case pairs.ReasonExtreme:
		update = "UPDATE word_pairs SET extreme = true WHERE id = $1"
	case pairs.ReasonRemove:
		update = "UPDATE word_pairs SET hidden = true WHERE id = $1"
	default:
		return fmt.Errorf("unknown report reason %q", reason)
	}

	return pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		tag, err := tx.Exec(ctx, update, pairID)
		if err != nil {
			return fmt.Errorf("apply report: %w", err)
		}
		if tag.RowsAffected() == 0 {
			return ErrNotFound
		}
		_, err = tx.Exec(ctx, `
			INSERT INTO pair_reports (pair_id, reason, reporter_hash)
			VALUES ($1, $2, $3)
			ON CONFLICT DO NOTHING`,
			pairID, reason, reporterHash)
		if err != nil {
			return fmt.Errorf("insert report: %w", err)
		}
		return nil
	})
}

// Seed inserts the starter pairs that are not in the bank yet and returns how many
// were added. It is idempotent: pairs already present (even hidden) are left as is.
func (s *Store) Seed(ctx context.Context, ps []seed.Pair) (int, error) {
	var inserted int
	err := pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		for _, p := range ps {
			c, err := pairs.NewCandidate(p.A, p.B)
			if err != nil {
				return fmt.Errorf("seed pair %s / %s: %w", p.A, p.B, err)
			}
			tag, err := tx.Exec(ctx, `
				INSERT INTO word_pairs (word_a, word_b, norm_a, norm_b, extreme, source)
				VALUES ($1, $2, $3, $4, $5, 'seed')
				ON CONFLICT (norm_a, norm_b) DO NOTHING`,
				c.WordA, c.WordB, c.NormA, c.NormB, p.Extreme)
			if err != nil {
				return fmt.Errorf("insert seed pair %s / %s: %w", p.A, p.B, err)
			}
			inserted += int(tag.RowsAffected())
		}
		return nil
	})
	return inserted, err
}

func scanPair(row pgx.Row) (pairs.Pair, error) {
	var p pairs.Pair
	err := row.Scan(&p.ID, &p.WordA, &p.WordB, &p.Extreme, &p.Hidden, &p.Source)
	if errors.Is(err, pgx.ErrNoRows) {
		return pairs.Pair{}, ErrNotFound
	}
	if err != nil {
		return pairs.Pair{}, fmt.Errorf("scan pair: %w", err)
	}
	return p, nil
}
