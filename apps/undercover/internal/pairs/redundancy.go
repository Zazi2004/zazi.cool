package pairs

// Status is the outcome of a redundancy check.
type Status string

const (
	StatusOK        Status = "ok"
	StatusSimilar   Status = "similar"
	StatusDuplicate Status = "duplicate"
)

// Report lists the existing pairs a candidate collides with.
type Report struct {
	Status Status
	// Duplicate is the existing pair with the same words, if any (possibly hidden).
	Duplicate *Pair
	// Similar pairs are one typo away on each word; submitting needs a confirmation.
	Similar []Pair
	// Related pairs share one word with the candidate; this is only informative.
	Related []Pair
}

// Check compares a candidate with every existing pair. The bank holds at most a few
// thousand pairs, so a linear scan is fast enough and keeps the logic in one place.
//
// Hidden pairs still count as duplicates (a removed pair cannot be submitted again),
// but they are not shown as similar or related.
func Check(c Candidate, existing []Pair) Report {
	report := Report{Status: StatusOK}
	for _, p := range existing {
		a, b := Normalize(p.WordA), Normalize(p.WordB)
		switch {
		case sameWords(c.NormA, c.NormB, a, b):
			report.Status = StatusDuplicate
			report.Duplicate = &p
			return report
		case p.Hidden:
			continue
		case closeWords(c.NormA, c.NormB, a, b):
			report.Similar = append(report.Similar, p)
		case c.NormA == a || c.NormA == b || c.NormB == a || c.NormB == b:
			report.Related = append(report.Related, p)
		}
	}
	if len(report.Similar) > 0 {
		report.Status = StatusSimilar
	}
	return report
}

// sameWords compares two unordered pairs of normalized words.
func sameWords(x1, x2, y1, y2 string) bool {
	return (x1 == y1 && x2 == y2) || (x1 == y2 && x2 == y1)
}

// closeWords reports whether both words are at most one typo apart, in either order.
func closeWords(x1, x2, y1, y2 string) bool {
	near := func(a, b string) bool { return Distance(a, b) <= 1 }
	return (near(x1, y1) && near(x2, y2)) || (near(x1, y2) && near(x2, y1))
}
