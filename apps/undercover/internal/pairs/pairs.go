package pairs

import (
	"errors"
	"fmt"
	"strings"
	"unicode"
)

// MaxWordLength is the maximum number of characters of a word, after trimming.
const MaxWordLength = 40

// Pair is a word pair as stored in the bank.
type Pair struct {
	ID      int64
	WordA   string
	WordB   string
	Extreme bool
	Hidden  bool
	Source  Source
}

// Source tells where a pair comes from.
type Source string

const (
	SourceSeed       Source = "seed"
	SourceSuggestion Source = "suggestion"
)

// ErrInvalid is wrapped by every validation error, so callers can test for it with
// errors.Is while still showing the detailed message to the user.
var ErrInvalid = errors.New("invalid pair")

// Candidate is a validated, canonical pair ready to be stored: NormA < NormB, so that
// "chat / tigre" and "tigre / chat" end up as the same row.
type Candidate struct {
	WordA, WordB string
	NormA, NormB string
}

// NewCandidate validates two words typed by a user and returns them in canonical order.
func NewCandidate(a, b string) (Candidate, error) {
	a, b = strings.TrimSpace(a), strings.TrimSpace(b)
	if err := validateWord(a); err != nil {
		return Candidate{}, err
	}
	if err := validateWord(b); err != nil {
		return Candidate{}, err
	}

	c := Candidate{WordA: a, WordB: b, NormA: Normalize(a), NormB: Normalize(b)}
	if c.NormA == "" || c.NormB == "" {
		return Candidate{}, fmt.Errorf("%w: chaque mot doit contenir au moins une lettre", ErrInvalid)
	}
	if c.NormA == c.NormB {
		return Candidate{}, fmt.Errorf("%w: les deux mots doivent être différents", ErrInvalid)
	}
	if c.NormA > c.NormB {
		c.WordA, c.WordB = c.WordB, c.WordA
		c.NormA, c.NormB = c.NormB, c.NormA
	}
	return c, nil
}

func validateWord(w string) error {
	n := len([]rune(w))
	if n == 0 {
		return fmt.Errorf("%w: les deux mots sont obligatoires", ErrInvalid)
	}
	if n > MaxWordLength {
		return fmt.Errorf("%w: « %s » dépasse %d caractères", ErrInvalid, w, MaxWordLength)
	}
	for _, r := range w {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) && !strings.ContainsRune(" -'’", r) {
			return fmt.Errorf("%w: caractère non autorisé « %c » (lettres, chiffres, espaces, tirets et apostrophes uniquement)", ErrInvalid, r)
		}
	}
	return nil
}

// Reason is why a pair was reported. A single report is enough to take effect.
type Reason string

const (
	// ReasonExtreme moves the pair to the extreme category.
	ReasonExtreme Reason = "extreme"
	// ReasonRemove hides the pair: it is never drawn again.
	ReasonRemove Reason = "remove"
)

// Valid reports whether r is a known reason.
func (r Reason) Valid() bool {
	return r == ReasonExtreme || r == ReasonRemove
}
