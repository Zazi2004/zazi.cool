// Package store persists the data of the habit tracker in PostgreSQL.
package store

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
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
