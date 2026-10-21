package pages

import (
	"net/url"
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

// Limits of a page (see docs/specs/linktree.md).
const (
	MaxTitle       = 80
	MaxSubtitle    = 80
	MaxDescription = 280
	MaxSlug        = 40
	MaxURL         = 2048
	MaxTopItems    = 100
	MaxItems       = 500
	MaxGroupLinks  = 30
)

var (
	slugPattern = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)
	uuidPattern = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)
	// Telephone numbers (RFC 3966): digits and visual separators, optional leading "+".
	telPattern = regexp.MustCompile(`^\+?[0-9][0-9().-]*$`)
)

// reservedSlugs are paths used by the app itself.
var reservedSlugs = []string{"admin", "api", "assets", "images", "shapes", "healthz", "readyz", "version"}

// Errors maps a field to a message shown to the admin, e.g. "slug" or
// "items.<uuid>.url". It implements error so it can travel as one.
type Errors map[string]string

func (e Errors) Error() string {
	keys := make([]string, 0, len(e))
	for k := range e {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	parts := make([]string, len(keys))
	for i, k := range keys {
		parts[i] = k + ": " + e[k]
	}
	return "invalid page: " + strings.Join(parts, "; ")
}

// ValidateSlug returns a message explaining why slug is not usable, or "".
func ValidateSlug(slug string) string {
	switch {
	case slug == "":
		return "Le slug est obligatoire."
	case len(slug) > MaxSlug:
		return "Le slug fait au plus 40 caractères."
	case !slugPattern.MatchString(slug):
		return "Lettres minuscules sans accents, chiffres et tirets uniquement (pas de tiret au début, à la fin ni deux de suite)."
	case slices.Contains(reservedSlugs, slug):
		return "Ce slug est réservé par l'application."
	}
	return ""
}

// ValidateURL returns a message explaining why raw is not an allowed link URL, or "".
// Only http, https, mailto and tel are allowed: "javascript:" or "data:" URLs would
// run code or show arbitrary content when clicked.
func ValidateURL(raw string) string {
	if raw == "" {
		return "L'URL est obligatoire."
	}
	if len(raw) > MaxURL {
		return "L'URL fait au plus 2048 caractères."
	}
	// Browsers drop tabs and line breaks inside URLs ("java\tscript:" would become
	// "javascript:"), so any space or control character is refused outright.
	if strings.IndexFunc(raw, func(r rune) bool { return unicode.IsSpace(r) || unicode.IsControl(r) }) >= 0 {
		return "L'URL ne doit pas contenir d'espace."
	}
	u, err := url.Parse(raw)
	if err != nil {
		return "L'URL est invalide."
	}
	switch u.Scheme {
	case "http", "https":
		if u.Host == "" || u.Opaque != "" {
			return "L'URL doit être complète, par exemple https://exemple.fr."
		}
		if u.User != nil {
			return "L'URL ne doit pas contenir d'identifiants."
		}
	case "mailto":
		if u.Opaque == "" || !strings.Contains(u.Opaque, "@") {
			return "Adresse e-mail invalide (mailto:nom@exemple.fr)."
		}
	case "tel":
		if !telPattern.MatchString(u.Opaque) {
			return "Numéro de téléphone invalide (tel:+33…)."
		}
	case "":
		return "L'URL doit commencer par https://, http://, mailto: ou tel:."
	default:
		return "Schéma non autorisé : seuls https, http, mailto et tel le sont."
	}
	return ""
}

// IsUUID reports whether s is a lowercase UUID, as produced by crypto.randomUUID().
func IsUUID(s string) bool {
	return uuidPattern.MatchString(s)
}

// Normalize trims the text fields of p, so that "  Musique " and "Musique" are the
// same title. Call it before Validate.
func Normalize(p *Page) {
	p.Slug = strings.TrimSpace(p.Slug)
	p.Title = strings.TrimSpace(p.Title)
	p.Description = strings.TrimSpace(p.Description)
	for i := range p.Items {
		normalizeItem(&p.Items[i])
	}
}

func normalizeItem(it *Item) {
	it.Title = strings.TrimSpace(it.Title)
	it.Subtitle = strings.TrimSpace(it.Subtitle)
	it.URL = strings.TrimSpace(it.URL)
	for i := range it.Links {
		normalizeItem(&it.Links[i])
	}
}

// Validate checks the whole document. shapes lists the existing shapes. It returns
// nil when the page is valid.
func Validate(p Page, shapes Shapes) Errors {
	v := validator{errs: Errors{}, shapes: shapes, seen: map[string]bool{}}

	if msg := ValidateSlug(p.Slug); msg != "" {
		v.errs["slug"] = msg
	}
	v.text("title", p.Title, 1, MaxTitle, false)
	v.text("description", p.Description, 0, MaxDescription, true)
	v.visual("avatar", p.Avatar)
	if !slices.Contains(Themes, p.Theme) {
		v.errs["theme"] = "Thème inconnu."
	}
	if !slices.Contains(Accents, p.Accent) {
		v.errs["accent"] = "Couleur d'accent inconnue."
	}

	if len(p.Items) > MaxTopItems {
		v.errs["items"] = "Une page contient au plus 100 éléments au premier niveau."
	}
	total := 0
	for _, it := range p.Items {
		total += 1 + len(it.Links)
		v.item(it, true)
	}
	if total > MaxItems {
		v.errs["items"] = "Une page contient au plus 500 éléments au total."
	}

	if len(v.errs) == 0 {
		return nil
	}
	return v.errs
}

type validator struct {
	errs   Errors
	shapes Shapes
	seen   map[string]bool // item IDs already met, to refuse duplicates
}

func (v *validator) item(it Item, topLevel bool) {
	if !IsUUID(it.ID) {
		// Without a valid ID, errors cannot be attached to the item.
		v.errs["items"] = "Identifiant d'élément invalide."
		return
	}
	if v.seen[it.ID] {
		v.errs["items"] = "Identifiant d'élément en double."
		return
	}
	v.seen[it.ID] = true
	prefix := "items." + it.ID + "."

	v.text(prefix+"title", it.Title, 1, MaxTitle, false)
	v.visual(prefix+"visual", it.Visual)

	switch it.Kind {
	case KindLink:
		if msg := ValidateURL(it.URL); msg != "" {
			v.errs[prefix+"url"] = msg
		}
		if it.Visual.Shape != "" {
			v.errs[prefix+"visual"] = "Un lien ne peut pas avoir de forme, seulement une image."
		}
		if it.Subtitle != "" || it.OpenByDefault || len(it.Links) > 0 {
			v.errs[prefix+"kind"] = "Un lien n'a ni sous-titre, ni liens."
		}
	case KindGroup:
		if !topLevel {
			v.errs[prefix+"kind"] = "Un groupe ne peut pas contenir d'autre groupe."
			return
		}
		if it.URL != "" {
			v.errs[prefix+"url"] = "Un groupe n'a pas d'URL."
		}
		v.text(prefix+"subtitle", it.Subtitle, 0, MaxSubtitle, false)
		if len(it.Links) > MaxGroupLinks {
			v.errs[prefix+"links"] = "Un groupe contient au plus 30 liens."
		}
		for _, l := range it.Links {
			v.item(l, false)
		}
	default:
		v.errs[prefix+"kind"] = "Type d'élément inconnu."
	}
}

// text checks the length (in characters, not bytes) and the characters of a field.
func (v *validator) text(field, s string, minLen, maxLen int, multiline bool) {
	n := utf8.RuneCountInString(s)
	switch {
	case !utf8.ValidString(s):
		v.errs[field] = "Texte invalide."
	case n < minLen:
		v.errs[field] = "Ce champ est obligatoire."
	case n > maxLen:
		v.errs[field] = "Ce champ fait au plus " + strconv.Itoa(maxLen) + " caractères."
	case strings.IndexFunc(s, func(r rune) bool { return unicode.IsControl(r) && !(multiline && r == '\n') }) >= 0:
		v.errs[field] = "Caractère non autorisé."
	}
}

func (v *validator) visual(field string, vis Visual) {
	switch {
	case vis.ImageID != "" && vis.Shape != "":
		v.errs[field] = "Choisir une image ou une forme, pas les deux."
	case vis.ImageID != "" && !IsUUID(vis.ImageID):
		v.errs[field] = "Image invalide."
	case vis.Shape != "" && !v.shapes.Has(vis.Shape):
		v.errs[field] = "Forme inconnue."
	}
}
