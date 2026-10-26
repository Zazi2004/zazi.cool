// Sessions of the admin. Copied from apps/linktree (internal/auth/auth.go and
// internal/store/sessions.go) and simplified: the Manager talks to PostgreSQL
// directly instead of going through a store interface. A security fix here must be
// reported to the linktree, and the other way round.

package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

// SessionTTL is the absolute lifetime of a session.
const SessionTTL = 30 * 24 * time.Hour

// tokenBytes is the size of a session token: 256 random bits.
const tokenBytes = 32

// ErrBadPassword means the password does not match.
var ErrBadPassword = errors.New("bad password")

// Manager checks the password and handles the session cookie. Sessions live in
// the sessions table; only the SHA-256 hash of each token is stored.
type Manager struct {
	hash        []byte
	fingerprint []byte // identifies the current password hash, see NewManager
	db          *pgxpool.Pool
	secure      bool // HTTPS: Secure cookie with the __Host- prefix
	now         func() time.Time
}

// NewManager returns a Manager for passwordHash, which must have passed CheckHash.
// secure must be true when the site is served over HTTPS (always in production).
func NewManager(passwordHash string, db *pgxpool.Pool, secure bool) *Manager {
	// Each session remembers the fingerprint of the hash it was created with:
	// changing the password changes the fingerprint and invalidates every session.
	sum := sha256.Sum256([]byte(passwordHash))
	return &Manager{
		hash:        []byte(passwordHash),
		fingerprint: sum[:],
		db:          db,
		secure:      secure,
		now:         time.Now,
	}
}

// cookieName is prefixed with __Host- over HTTPS: browsers then refuse the cookie
// unless it is Secure, has Path=/ and no Domain, so no subdomain can overwrite it.
// Browsers do not all accept Secure cookies on http://localhost, hence the plain
// name in local development.
func (m *Manager) cookieName() string {
	if m.secure {
		return "__Host-session"
	}
	return "session"
}

// Login checks the password and, if it matches, opens a session and sets its cookie.
func (m *Manager) Login(ctx context.Context, w http.ResponseWriter, password string) error {
	// bcrypt compares in constant time and is deliberately slow, which slows down
	// brute-force attacks on top of the rate limiting.
	if err := bcrypt.CompareHashAndPassword(m.hash, []byte(password)); err != nil {
		return ErrBadPassword
	}

	token := make([]byte, tokenBytes)
	// crypto/rand.Read never returns an error on supported platforms.
	_, _ = rand.Read(token)
	expires := m.now().Add(SessionTTL)
	_, err := m.db.Exec(ctx,
		`INSERT INTO sessions (token_hash, password_fingerprint, created_at, expires_at) VALUES ($1, $2, $3, $4)`,
		hashToken(token), m.fingerprint, m.now(), expires)
	if err != nil {
		return fmt.Errorf("insert session: %w", err)
	}

	http.SetCookie(w, &http.Cookie{
		Name:     m.cookieName(),
		Value:    base64.RawURLEncoding.EncodeToString(token),
		Path:     "/",
		Expires:  expires,
		HttpOnly: true, // unreachable from JavaScript, so an XSS cannot steal it
		Secure:   m.secure,
		SameSite: http.SameSiteStrictMode, // never sent by requests from other sites
	})
	return nil
}

// Logout closes the session of the request, if any, and clears its cookie.
func (m *Manager) Logout(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	if token, ok := m.token(r); ok {
		if _, err := m.db.Exec(ctx, `DELETE FROM sessions WHERE token_hash = $1`, hashToken(token)); err != nil {
			return fmt.Errorf("delete session: %w", err)
		}
	}
	http.SetCookie(w, &http.Cookie{
		Name:     m.cookieName(),
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   m.secure,
		SameSite: http.SameSiteStrictMode,
	})
	return nil
}

// Authenticated reports whether the request carries the cookie of a session that
// exists, has not expired and was opened with the current password.
func (m *Manager) Authenticated(r *http.Request) (bool, error) {
	token, ok := m.token(r)
	if !ok {
		return false, nil
	}
	var valid bool
	err := m.db.QueryRow(r.Context(),
		`SELECT EXISTS (SELECT 1 FROM sessions
		 WHERE token_hash = $1 AND password_fingerprint = $2 AND expires_at > $3)`,
		hashToken(token), m.fingerprint, m.now()).Scan(&valid)
	if err != nil {
		return false, fmt.Errorf("check session: %w", err)
	}
	return valid, nil
}

// DeleteExpired removes the sessions expired at now and returns how many.
func (m *Manager) DeleteExpired(ctx context.Context, now time.Time) (int64, error) {
	tag, err := m.db.Exec(ctx, `DELETE FROM sessions WHERE expires_at <= $1`, now)
	if err != nil {
		return 0, fmt.Errorf("delete expired sessions: %w", err)
	}
	return tag.RowsAffected(), nil
}

func (m *Manager) token(r *http.Request) ([]byte, bool) {
	c, err := r.Cookie(m.cookieName())
	if err != nil {
		return nil, false
	}
	token, err := base64.RawURLEncoding.DecodeString(c.Value)
	if err != nil || len(token) != tokenBytes {
		return nil, false
	}
	return token, true
}

// hashToken is what the database stores: a leaked database does not leak usable
// cookies. A fast hash is enough because the token is 256 random bits.
func hashToken(token []byte) []byte {
	sum := sha256.Sum256(token)
	return sum[:]
}
