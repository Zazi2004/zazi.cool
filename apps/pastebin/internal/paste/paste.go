// Package paste defines a paste and the rules it must follow: identifier format,
// allowed languages and expirations (docs/specs/pastebin.md, F1, F2, D3).
package paste

import (
	"crypto/rand"
	"fmt"
	"time"
)

// Paste is a stored text snippet.
type Paste struct {
	ID        string
	Content   string
	Language  string
	CreatedAt time.Time
	ExpiresAt time.Time
}

// IDLength is the number of base62 characters of an identifier (~47 bits).
const IDLength = 8

const alphabet = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"

// maxUnbiased is the largest multiple of len(alphabet) that fits in a byte. Random bytes
// at or above it are discarded, otherwise b % 62 would favour the first characters.
const maxUnbiased = 256 - 256%len(alphabet)

// NewID returns a random base62 identifier drawn from crypto/rand.
func NewID() (string, error) {
	id := make([]byte, 0, IDLength)
	buf := make([]byte, IDLength*2)
	for len(id) < IDLength {
		if _, err := rand.Read(buf); err != nil {
			return "", fmt.Errorf("read random bytes: %w", err)
		}
		for _, b := range buf {
			if int(b) < maxUnbiased && len(id) < IDLength {
				id = append(id, alphabet[int(b)%len(alphabet)])
			}
		}
	}
	return string(id), nil
}

// ValidID reports whether s has the shape of an identifier, so malformed URLs are
// answered without querying the database.
func ValidID(s string) bool {
	if len(s) != IDLength {
		return false
	}
	for _, c := range []byte(s) {
		if !('0' <= c && c <= '9' || 'A' <= c && c <= 'Z' || 'a' <= c && c <= 'z') {
			return false
		}
	}
	return true
}

// Option is an entry of a form select: the submitted key and its French label.
type Option struct {
	Key   string
	Label string
}

// Expiration is an allowed lifetime of a paste.
type Expiration struct {
	Option
	Duration time.Duration
}

// Expirations lists the lifetimes offered by the form, in display order (D3: no
// "never" option).
var Expirations = []Expiration{
	{Option{"10m", "10 minutes"}, 10 * time.Minute},
	{Option{"1h", "1 heure"}, time.Hour},
	{Option{"1d", "1 jour"}, 24 * time.Hour},
	{Option{"1w", "1 semaine"}, 7 * 24 * time.Hour},
	{Option{"1mo", "1 mois"}, 30 * 24 * time.Hour},
}

// DefaultExpiration is preselected in the form and used when the field is missing.
const DefaultExpiration = "10m"

// LookupExpiration returns the expiration with the given key.
func LookupExpiration(key string) (Expiration, bool) {
	for _, e := range Expirations {
		if e.Key == key {
			return e, true
		}
	}
	return Expiration{}, false
}

// Languages lists the languages offered by the form. Keys are chroma lexer names, so
// syntax highlighting (T6) can use them as they are.
var Languages = []Option{
	{"plaintext", "Texte brut"},
	{"bash", "Bash"},
	{"c", "C"},
	{"cpp", "C++"},
	{"css", "CSS"},
	{"diff", "Diff"},
	{"docker", "Dockerfile"},
	{"go", "Go"},
	{"html", "HTML"},
	{"java", "Java"},
	{"javascript", "JavaScript"},
	{"json", "JSON"},
	{"markdown", "Markdown"},
	{"php", "PHP"},
	{"python", "Python"},
	{"rust", "Rust"},
	{"sql", "SQL"},
	{"typescript", "TypeScript"},
	{"yaml", "YAML"},
}

// DefaultLanguage is preselected in the form and used when the field is missing.
const DefaultLanguage = "plaintext"

// ValidLanguage reports whether key is one of Languages.
func ValidLanguage(key string) bool {
	for _, l := range Languages {
		if l.Key == key {
			return true
		}
	}
	return false
}
