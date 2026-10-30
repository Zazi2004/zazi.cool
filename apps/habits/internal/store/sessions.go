package store

import (
	"context"
	"fmt"
	"time"
)

// CreateSession records a session identified by the hash of its token.
func (s *Store) CreateSession(ctx context.Context, tokenHash, passwordFP []byte, createdAt, expiresAt time.Time) error {
	_, err := s.pool.Exec(ctx,
		`INSERT INTO sessions (token_hash, password_hash_fp, created_at, expires_at) VALUES ($1, $2, $3, $4)`,
		tokenHash, passwordFP, createdAt, expiresAt)
	if err != nil {
		return fmt.Errorf("insert session: %w", err)
	}
	return nil
}

// SessionValid reports whether a session exists for tokenHash, was opened with the
// password identified by passwordFP and has not expired at now.
func (s *Store) SessionValid(ctx context.Context, tokenHash, passwordFP []byte, now time.Time) (bool, error) {
	var valid bool
	err := s.pool.QueryRow(ctx,
		`SELECT EXISTS (SELECT 1 FROM sessions
		 WHERE token_hash = $1 AND password_hash_fp = $2 AND expires_at > $3)`,
		tokenHash, passwordFP, now).Scan(&valid)
	if err != nil {
		return false, fmt.Errorf("check session: %w", err)
	}
	return valid, nil
}

// DeleteSession closes the session identified by tokenHash (no error if it is gone).
func (s *Store) DeleteSession(ctx context.Context, tokenHash []byte) error {
	if _, err := s.pool.Exec(ctx, `DELETE FROM sessions WHERE token_hash = $1`, tokenHash); err != nil {
		return fmt.Errorf("delete session: %w", err)
	}
	return nil
}

// DeleteExpiredSessions removes the sessions expired at now and returns how many.
func (s *Store) DeleteExpiredSessions(ctx context.Context, now time.Time) (int64, error) {
	tag, err := s.pool.Exec(ctx, `DELETE FROM sessions WHERE expires_at <= $1`, now)
	if err != nil {
		return 0, fmt.Errorf("delete expired sessions: %w", err)
	}
	return tag.RowsAffected(), nil
}
