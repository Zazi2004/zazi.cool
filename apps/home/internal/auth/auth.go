// Package auth protects the admin with a single password (its bcrypt hash only)
// and server-side sessions.
package auth

import (
	"errors"
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

// minPasswordLength is the shortest admin password accepted, in bytes.
const minPasswordLength = 12

// HashPassword returns the bcrypt hash to put in ADMIN_PASSWORD_HASH.
func HashPassword(password string) (string, error) {
	if len(password) < minPasswordLength {
		return "", fmt.Errorf("the password must be at least %d characters long", minPasswordLength)
	}
	// bcrypt ignores bytes past 72: GenerateFromPassword refuses them rather than
	// silently truncating.
	hash, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	if err != nil {
		return "", fmt.Errorf("hash password: %w", err)
	}
	return string(hash), nil
}

// CheckHash reports whether hash is a usable bcrypt hash, so that a mistyped
// ADMIN_PASSWORD_HASH stops the application at startup rather than at login.
func CheckHash(hash string) error {
	if _, err := bcrypt.Cost([]byte(hash)); err != nil {
		return errors.New("ADMIN_PASSWORD_HASH is not a bcrypt hash (see `make hash-password`)")
	}
	return nil
}
