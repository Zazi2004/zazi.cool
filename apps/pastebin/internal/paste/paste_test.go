package paste

import (
	"testing"
	"time"
)

func TestNewID(t *testing.T) {
	seen := make(map[string]bool)
	for range 1000 {
		id, err := NewID()
		if err != nil {
			t.Fatal(err)
		}
		if !ValidID(id) {
			t.Fatalf("NewID() = %q, not 8 base62 characters", id)
		}
		if seen[id] {
			t.Fatalf("NewID() returned %q twice", id)
		}
		seen[id] = true
	}
}

func TestValidID(t *testing.T) {
	tests := map[string]bool{
		"aZ09bY18":  true,
		"aZ09bY1":   false, // too short
		"aZ09bY18x": false, // too long
		"aZ09-Y18":  false,
		"aZ09bY1é":  false,
		"":          false,
	}
	for id, want := range tests {
		if got := ValidID(id); got != want {
			t.Errorf("ValidID(%q) = %v, want %v", id, got, want)
		}
	}
}

// CA9 / D3: exactly these durations, 10 minutes by default, no "never".
func TestExpirations(t *testing.T) {
	want := map[string]time.Duration{
		"10m": 10 * time.Minute,
		"1h":  time.Hour,
		"1d":  24 * time.Hour,
		"1w":  7 * 24 * time.Hour,
		"1mo": 30 * 24 * time.Hour,
	}
	if len(Expirations) != len(want) {
		t.Fatalf("%d expirations, want %d", len(Expirations), len(want))
	}
	for key, d := range want {
		e, ok := LookupExpiration(key)
		if !ok || e.Duration != d {
			t.Errorf("LookupExpiration(%q) = %v, %v; want %v", key, e.Duration, ok, d)
		}
	}
	if e, _ := LookupExpiration(DefaultExpiration); e.Duration != 10*time.Minute {
		t.Errorf("default expiration = %v, want 10m", e.Duration)
	}
	for _, key := range []string{"never", "", "0", "2h"} {
		if _, ok := LookupExpiration(key); ok {
			t.Errorf("LookupExpiration(%q) accepted", key)
		}
	}
}

func TestValidLanguage(t *testing.T) {
	if !ValidLanguage(DefaultLanguage) {
		t.Error("default language is not in the list")
	}
	if ValidLanguage("cobol") {
		t.Error("unknown language accepted")
	}
}
