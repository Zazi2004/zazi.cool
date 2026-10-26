package content

import (
	"slices"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

// Limits of the editable fields, in characters (F13).
const (
	MaxTitle     = 80
	MaxBody      = 600
	MaxBannerAlt = 150
)

// Banner choices that do not name an illustration.
const (
	BannerNone   = "none"
	BannerUpload = "upload"
)

// Edit is what the admin form submits.
type Edit struct {
	Title        string
	Body         string
	BannerChoice string
	BannerAlt    string
	Background   string
}

// Validate cleans e and checks every rule of F13. It returns the cleaned edit and
// one message per invalid field, keyed "title", "body", "banner", "banner_alt" or
// "background"; the map is empty when the edit is valid. hasUpload tells whether an
// uploaded image already exists and hasFile whether a file was sent with the form:
// the "upload" choice needs one of them, and any other choice refuses a file (F13).
// Built-in illustrations arrive with the gallery (T7): until then only "none" and
// "upload" are valid choices.
func (e Edit) Validate(hasUpload, hasFile bool) (Edit, map[string]string) {
	errs := map[string]string{}
	e.Title = cleanText(e.Title, false, 1, MaxTitle, "Le titre", errs, "title")
	e.Body = cleanText(e.Body, true, 0, MaxBody, "Le paragraphe", errs, "body")
	e.BannerAlt = cleanText(e.BannerAlt, false, 0, MaxBannerAlt, "Le texte alternatif", errs, "banner_alt")

	switch e.BannerChoice {
	case BannerNone:
	case BannerUpload:
		if !hasUpload && !hasFile {
			errs["banner"] = "Aucune image n'a encore été envoyée."
		}
	default:
		errs["banner"] = "Choisissez une bannière de la liste."
	}
	if hasFile && e.BannerChoice != BannerUpload {
		errs["banner"] = "Choisissez soit une illustration, soit votre image"
	}
	if !slices.Contains(Backgrounds, e.Background) {
		errs["background"] = "Choisissez un fond de la liste."
	}
	return e, errs
}

// cleanText trims s and records in errs, under key, the first rule it breaks.
// multiline allows line breaks (a browser sends them as CRLF; they are stored as LF).
func cleanText(s string, multiline bool, min, max int, label string, errs map[string]string, key string) string {
	if !utf8.ValidString(s) {
		errs[key] = label + " doit être en UTF-8 valide."
		return s
	}
	if multiline {
		s = strings.ReplaceAll(s, "\r\n", "\n")
	}
	s = strings.TrimSpace(s)
	for _, r := range s {
		if unicode.IsControl(r) && !(multiline && r == '\n') {
			errs[key] = label + " contient un caractère de contrôle."
			return s
		}
	}
	switch n := utf8.RuneCountInString(s); {
	case n < min:
		errs[key] = label + " est obligatoire."
	case n > max:
		errs[key] = label + " doit faire " + strconv.Itoa(max) + " caractères au plus."
	}
	return s
}
