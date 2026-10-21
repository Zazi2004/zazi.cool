// Package store persists pages, items, images and sessions in PostgreSQL.
package store

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	// ErrNotFound means no row matches the request.
	ErrNotFound = errors.New("not found")
	// ErrSlugTaken means another page already uses the slug.
	ErrSlugTaken = errors.New("slug already taken")
	// ErrVersionConflict means the page was saved elsewhere since it was loaded.
	ErrVersionConflict = errors.New("page modified since it was loaded")
	// ErrForeignItem means an item ID of the document belongs to another page.
	ErrForeignItem = errors.New("item belongs to another page")
	// ErrImageNotFound means a referenced image does not exist (or was cleaned up).
	ErrImageNotFound = errors.New("image not found")
	// ErrInvalidOrder means the new index order does not list every page exactly once.
	ErrInvalidOrder = errors.New("order must list every page once")
)

// PostgreSQL error codes (https://www.postgresql.org/docs/current/errcodes-appendix.html).
const (
	uniqueViolation     = "23505"
	foreignKeyViolation = "23503"
)

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

// translate turns constraint violations into the errors of this package.
func translate(err error) error {
	// errors.As walks the chain of wrapped errors and fills pgErr when it finds a
	// PostgreSQL error.
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return err
	}
	switch {
	case pgErr.Code == uniqueViolation && strings.Contains(pgErr.ConstraintName, "slug"):
		return ErrSlugTaken
	case pgErr.Code == foreignKeyViolation && strings.Contains(pgErr.ConstraintName, "image"):
		return ErrImageNotFound
	}
	return err
}
