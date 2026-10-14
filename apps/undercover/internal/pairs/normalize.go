// Package pairs holds the pure logic around word pairs: validation, normalization,
// redundancy detection and Mr. White's guess matching. It has no I/O.
package pairs

import (
	"strings"
	"unicode"

	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

// Leading articles are dropped so that "le chat" and "chat" compare equal.
// Longer articles come first so that "de la " wins over "de ".
var articles = []string{"de la ", "de l'", "le ", "la ", "les ", "l'", "un ", "une ", "des ", "du "}

// Normalize returns the comparison form of a word: lower case, without accents,
// punctuation or leading article, with single spaces and simple plurals removed.
// Two words that only differ by these details are considered the same word.
func Normalize(word string) string {
	s := strings.ToLower(removeAccents(word))
	s = strings.ReplaceAll(s, "’", "'")
	s = strings.Join(strings.Fields(s), " ")

	for _, article := range articles {
		if rest, found := strings.CutPrefix(s, article); found && rest != "" {
			s = rest
			break
		}
	}

	// Any remaining punctuation (hyphens, apostrophes…) becomes a word separator,
	// so "porte-monnaie" and "porte monnaie" compare equal.
	s = strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			return r
		}
		return ' '
	}, s)

	tokens := strings.Fields(s)
	for i, token := range tokens {
		tokens[i] = singular(token)
	}
	return strings.Join(tokens, " ")
}

// singular strips a final "s" or "x", which covers most French plurals.
// Short words are left alone ("os", "gaz") to avoid mangling them.
func singular(token string) string {
	if len([]rune(token)) > 3 && (strings.HasSuffix(token, "s") || strings.HasSuffix(token, "x")) {
		return token[:len(token)-1]
	}
	return token
}

// removeAccents decomposes characters ("é" becomes "e" + combining accent) and drops
// the combining marks. "œ" and "æ" are ligatures, not accents, so they are expanded.
func removeAccents(s string) string {
	t := transform.Chain(norm.NFD, runes.Remove(runes.In(unicode.Mn)), norm.NFC)
	result, _, err := transform.String(t, s)
	if err != nil {
		// These transformers only fail on invalid UTF-8; comparing the raw input is
		// the safest fallback.
		result = s
	}
	return strings.NewReplacer("œ", "oe", "Œ", "oe", "æ", "ae", "Æ", "ae").Replace(result)
}

// Distance returns the Levenshtein distance between a and b: the minimum number of
// single-character insertions, deletions or substitutions to turn a into b.
func Distance(a, b string) int {
	ra, rb := []rune(a), []rune(b)
	// Only two rows of the classic dynamic-programming matrix are needed.
	prev := make([]int, len(rb)+1)
	curr := make([]int, len(rb)+1)
	for j := range prev {
		prev[j] = j
	}
	for i := 1; i <= len(ra); i++ {
		curr[0] = i
		for j := 1; j <= len(rb); j++ {
			cost := 1
			if ra[i-1] == rb[j-1] {
				cost = 0
			}
			curr[j] = min(prev[j]+1, curr[j-1]+1, prev[j-1]+cost)
		}
		prev, curr = curr, prev
	}
	return prev[len(rb)]
}

// GuessMatches reports whether Mr. White's guess is close enough to the civilians'
// word: equal once normalized, or one typo away for words of 5 letters or more.
func GuessMatches(guess, word string) bool {
	g, w := Normalize(guess), Normalize(word)
	if g == "" {
		return false
	}
	if g == w {
		return true
	}
	return len([]rune(w)) >= 5 && Distance(g, w) <= 1
}
