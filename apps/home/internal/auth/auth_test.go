package auth

import (
	"strings"
	"testing"

	"golang.org/x/crypto/bcrypt"
)

func TestHashPassword(t *testing.T) {
	hash, err := HashPassword("correct horse battery")
	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}
	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte("correct horse battery")); err != nil {
		t.Errorf("hash does not match the password: %v", err)
	}
	if err := CheckHash(hash); err != nil {
		t.Errorf("CheckHash: %v", err)
	}
}

func TestHashPasswordRejectsBadLengths(t *testing.T) {
	for name, password := range map[string]string{
		"11 characters": "short-pass!",
		"73 bytes":      strings.Repeat("a", 73),
	} {
		if _, err := HashPassword(password); err == nil {
			t.Errorf("%s: no error", name)
		}
	}
}

func TestCheckHashRejectsGarbage(t *testing.T) {
	for _, hash := range []string{"", "secret", "$2a$12$tooshort"} {
		if err := CheckHash(hash); err == nil {
			t.Errorf("CheckHash(%q): no error", hash)
		}
	}
}
