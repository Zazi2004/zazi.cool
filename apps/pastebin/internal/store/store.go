// Package store persists pastes in PostgreSQL.
package store

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"zazi.cool/pastebin/internal/paste"
)

// ErrNotFound means no readable paste has this identifier: it never existed or it
// has expired. Callers cannot, and must not, tell the two apart (F5).
var ErrNotFound = errors.New("paste not found")

// uniqueViolation is the PostgreSQL error code raised by a PRIMARY KEY or UNIQUE constraint.
const uniqueViolation = "23505"

// maxIDAttempts bounds the retries on an identifier collision. With ~47 bits of
// randomness, even a second attempt is very unlikely.
const maxIDAttempts = 5

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

// Create stores a paste that expires after ttl and returns it with its new identifier.
// Both timestamps come from the database clock, so expires_at is exactly created_at + ttl.
func (s *Store) Create(ctx context.Context, content, language string, ttl time.Duration) (paste.Paste, error) {
	for range maxIDAttempts {
		id, err := paste.NewID()
		if err != nil {
			return paste.Paste{}, err
		}
		p := paste.Paste{ID: id, Content: content, Language: language}
		err = s.pool.QueryRow(ctx, `
			INSERT INTO pastes (id, content, language, expires_at)
			VALUES ($1, $2, $3, now() + make_interval(secs => $4))
			RETURNING created_at, expires_at`,
			id, content, language, ttl.Seconds(),
		).Scan(&p.CreatedAt, &p.ExpiresAt)

		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == uniqueViolation {
			continue
		}
		if err != nil {
			return paste.Paste{}, fmt.Errorf("insert paste: %w", err)
		}
		return p, nil
	}
	return paste.Paste{}, fmt.Errorf("insert paste: no free identifier after %d attempts", maxIDAttempts)
}

// Get returns the paste with this identifier, or ErrNotFound if it does not exist or
// has expired (expired rows stay in the table until the purge removes them).
func (s *Store) Get(ctx context.Context, id string) (paste.Paste, error) {
	p := paste.Paste{ID: id}
	err := s.pool.QueryRow(ctx, `
		SELECT content, language, created_at, expires_at
		FROM pastes
		WHERE id = $1 AND expires_at > now()`,
		id,
	).Scan(&p.Content, &p.Language, &p.CreatedAt, &p.ExpiresAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return paste.Paste{}, ErrNotFound
	}
	if err != nil {
		return paste.Paste{}, fmt.Errorf("select paste: %w", err)
	}
	return p, nil
}

// Purge deletes the expired pastes and returns how many were removed (F6). It is
// idempotent, so several replicas running it at once would be harmless.
func (s *Store) Purge(ctx context.Context) (int64, error) {
	tag, err := s.pool.Exec(ctx, `DELETE FROM pastes WHERE expires_at <= now()`)
	if err != nil {
		return 0, fmt.Errorf("delete expired pastes: %w", err)
	}
	return tag.RowsAffected(), nil
}
