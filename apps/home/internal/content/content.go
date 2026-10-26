// Package content holds the homepage document (title, paragraph, banner,
// background) and the rules to display it. Store reads it from PostgreSQL; the
// other functions have no I/O.
package content

import (
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"
)

// Content is the single row of the content table.
type Content struct {
	Title string
	Body  string
	// BannerChoice is BannerNone, BannerUpload or the id of a Gallery illustration.
	BannerChoice string
	BannerAlt    string
	Background   string
	Version      int
	// Upload is the last uploaded image (upload_image_id), nil if there is none.
	// It is kept when another banner is chosen.
	Upload *Image
}

// Image describes an uploaded banner, without its bytes.
type Image struct {
	ID     string
	Width  int
	Height int
}

// Banner is what the page needs to show the banner.
type Banner struct {
	URL    string // root-relative
	Alt    string
	Width  int
	Height int
}

// IllustrationURL gives the root-relative URL of an illustration's picture, with its
// fingerprint, or false if the picture is not embedded.
type IllustrationURL func(id string) (string, bool)

// Banner returns the banner to display, or nil when the page has none: "none", an
// upload that is gone, or an illustration removed from the code (F16b). The saved
// alternative text wins; an illustration falls back to its own (F16c).
func (c Content) Banner(illustrationURL IllustrationURL) *Banner {
	switch c.BannerChoice {
	case BannerNone:
		return nil
	case BannerUpload:
		if c.Upload == nil {
			return nil
		}
		return &Banner{
			URL:    "/images/" + c.Upload.ID,
			Alt:    c.BannerAlt,
			Width:  c.Upload.Width,
			Height: c.Upload.Height,
		}
	}
	il, ok := FindIllustration(c.BannerChoice)
	if !ok {
		return nil
	}
	url, ok := illustrationURL(il.ID)
	if !ok {
		return nil
	}
	alt := c.BannerAlt
	if alt == "" {
		alt = il.Alt
	}
	return &Banner{URL: url, Alt: alt, Width: IllustrationWidth, Height: IllustrationHeight}
}

// BannerKnown reports whether the stored choice can still be displayed: false only
// for an illustration that is no longer in the gallery, which the admin reports.
func (c Content) BannerKnown() bool {
	if c.BannerChoice == BannerNone || c.BannerChoice == BannerUpload {
		return true
	}
	_, ok := FindIllustration(c.BannerChoice)
	return ok
}

// Backgrounds lists the page backgrounds (F22). The default, aurore, is set by
// the migration.
var Backgrounds = []string{"aurore", "fils", "grain", "statique"}

// StaticBackground is the gradient without animation.
const StaticBackground = "statique"

// BackgroundOrStatic returns the stored background, or the static one if the
// stored value is not in Backgrounds.
func (c Content) BackgroundOrStatic() string {
	if slices.Contains(Backgrounds, c.Background) {
		return c.Background
	}
	return StaticBackground
}

// Paragraphs splits a plain-text body (F4): blank lines separate paragraphs and
// each paragraph is a list of lines, joined by <br> when rendered.
func Paragraphs(body string) [][]string {
	body = strings.ReplaceAll(body, "\r\n", "\n")
	var paragraphs [][]string
	var current []string
	for line := range strings.SplitSeq(body, "\n") {
		if strings.TrimSpace(line) == "" {
			if current != nil {
				paragraphs = append(paragraphs, current)
				current = nil
			}
			continue
		}
		current = append(current, strings.TrimRightFunc(line, unicode.IsSpace))
	}
	if current != nil {
		paragraphs = append(paragraphs, current)
	}
	return paragraphs
}

// Summary returns the body on one line, cut to at most max characters (runes)
// with an ellipsis: the Open Graph description (F6).
func Summary(body string, max int) string {
	s := strings.Join(strings.Fields(body), " ")
	if utf8.RuneCountInString(s) <= max {
		return s
	}
	runes := []rune(s)
	return strings.TrimRightFunc(string(runes[:max-1]), unicode.IsSpace) + "…"
}
