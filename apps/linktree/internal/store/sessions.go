package store

import (
	"context"
	"fmt"
	"time"
)

// CreateSession stores a new session.
func (s *Store) CreateSession(ctx context.Context, tokenHash, fingerprint []byte, expiresAt time.Time) error {
	_, err := s.pool.Exec(ctx,
		`INSERT INTO sessions (token_hash, password_fingerprint, expires_at) VALUES ($1, $2, $3)`,
		tokenHash, fingerprint, expiresAt)
	if err != nil {
		return fmt.Errorf("insert session: %w", err)
	}
	return nil
}

// SessionValid reports whether the session exists, has not expired and was created
// with the current password.
func (s *Store) SessionValid(ctx context.Context, tokenHash, fingerprint []byte, now time.Time) (bool, error) {
	var valid bool
	err := s.pool.QueryRow(ctx,
		`SELECT EXISTS (SELECT 1 FROM sessions
		 WHERE token_hash = $1 AND password_fingerprint = $2 AND expires_at > $3)`,
		tokenHash, fingerprint, now).Scan(&valid)
	if err != nil {
		return false, fmt.Errorf("query session: %w", err)
	}
	return valid, nil
}

// DeleteSession removes a session (logout).
func (s *Store) DeleteSession(ctx context.Context, tokenHash []byte) error {
	if _, err := s.pool.Exec(ctx, `DELETE FROM sessions WHERE token_hash = $1`, tokenHash); err != nil {
		return fmt.Errorf("delete session: %w", err)
	}
	return nil
}

// DeleteExpiredSessions removes the sessions expired before now and returns how many.
func (s *Store) DeleteExpiredSessions(ctx context.Context, now time.Time) (int64, error) {
	tag, err := s.pool.Exec(ctx, `DELETE FROM sessions WHERE expires_at <= $1`, now)
	if err != nil {
		return 0, fmt.Errorf("delete expired sessions: %w", err)
	}
	return tag.RowsAffected(), nil
}
