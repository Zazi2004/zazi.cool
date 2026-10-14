package seed

import (
	"testing"

	"zazi.cool/undercover/internal/pairs"
)

func TestSeedPairs(t *testing.T) {
	ps, err := Pairs()
	if err != nil {
		t.Fatal(err)
	}

	var extreme int
	var bank []pairs.Pair
	for i, p := range ps {
		if p.Extreme {
			extreme++
		}
		c, err := pairs.NewCandidate(p.A, p.B)
		if err != nil {
			t.Errorf("pair %d (%s / %s) is invalid: %v", i, p.A, p.B, err)
			continue
		}
		// Each pair must be new compared to the ones before it.
		if r := pairs.Check(c, bank); r.Status != pairs.StatusOK {
			t.Errorf("pair %d (%s / %s) is %s: duplicate=%v similar=%v", i, p.A, p.B, r.Status, r.Duplicate, r.Similar)
		}
		bank = append(bank, pairs.Pair{ID: int64(i), WordA: p.A, WordB: p.B})
	}

	// The starter bank was specified as 100 pairs, 25 of them extreme; it may grow.
	if len(ps) < 100 {
		t.Errorf("got %d seed pairs, want at least 100", len(ps))
	}
	if extreme < 25 {
		t.Errorf("got %d extreme seed pairs, want at least 25", extreme)
	}
}
