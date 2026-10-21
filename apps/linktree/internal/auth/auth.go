// Package auth protects the admin with a single password and server-side sessions.
// Only the bcrypt hash of the password is known to the app, and only the SHA-256
// hash of each session token is stored in the database.
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

	"golang.org/x/crypto/bcrypt"
)

// SessionTTL is the absolute lifetime of a session.
const SessionTTL = 30 * 24 * time.Hour

// ErrBadPassword means the password does not match.
var ErrBadPassword = errors.New("bad password")

// SessionStore persists sessions. *store.Store satisfies it; tests use a fake.
type SessionStore interface {
	CreateSession(ctx context.Context, tokenHash, fingerprint []byte, expiresAt time.Time) error
	SessionValid(ctx context.Context, tokenHash, fingerprint []byte, now time.Time) (bool, error)
	DeleteSession(ctx context.Context, tokenHash []byte) error
}

// Manager checks the password and handles the session cookie.
type Manager struct {
	hash        []byte
	fingerprint []byte // identifies the current password hash, see NewManager
	store       SessionStore
	secure      bool // HTTPS: Secure cookie with the __Host- prefix
	now         func() time.Time
}

// NewManager returns a Manager for the given bcrypt hash. secure must be true when
// the site is served over HTTPS (always in production).
func NewManager(passwordHash string, st SessionStore, secure bool) (*Manager, error) {
	if _, err := bcrypt.Cost([]byte(passwordHash)); err != nil {
		return nil, fmt.Errorf("ADMIN_PASSWORD_HASH is not a bcrypt hash: %w", err)
	}
	// Each session remembers the fingerprint of the hash it was created with:
	// changing the password changes the fingerprint and invalidates every session.
	sum := sha256.Sum256([]byte(passwordHash))
	return &Manager{
		hash:        []byte(passwordHash),
		fingerprint: sum[:],
		store:       st,
		secure:      secure,
		now:         time.Now,
	}, nil
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

	token := make([]byte, 32)
	// crypto/rand.Read never returns an error on supported platforms.
	_, _ = rand.Read(token)
	expires := m.now().Add(SessionTTL)
	if err := m.store.CreateSession(ctx, hashToken(token), m.fingerprint, expires); err != nil {
		return fmt.Errorf("create session: %w", err)
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
		if err := m.store.DeleteSession(ctx, hashToken(token)); err != nil {
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

// Authenticated reports whether the request carries a valid session cookie.
func (m *Manager) Authenticated(r *http.Request) (bool, error) {
	token, ok := m.token(r)
	if !ok {
		return false, nil
	}
	valid, err := m.store.SessionValid(r.Context(), hashToken(token), m.fingerprint, m.now())
	if err != nil {
		return false, fmt.Errorf("check session: %w", err)
	}
	return valid, nil
}

func (m *Manager) token(r *http.Request) ([]byte, bool) {
	c, err := r.Cookie(m.cookieName())
	if err != nil {
		return nil, false
	}
	token, err := base64.RawURLEncoding.DecodeString(c.Value)
	if err != nil || len(token) != 32 {
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

// HashPassword returns the bcrypt hash to put in ADMIN_PASSWORD_HASH.
func HashPassword(password string) (string, error) {
	if len(password) < 12 {
		return "", errors.New("the password must be at least 12 characters long")
	}
	// bcrypt ignores bytes past 72: GenerateFromPassword refuses them rather than
	// silently truncating.
	hash, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	if err != nil {
		return "", fmt.Errorf("hash password: %w", err)
	}
	return string(hash), nil
}
