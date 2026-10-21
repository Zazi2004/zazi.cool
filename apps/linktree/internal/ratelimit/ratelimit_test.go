package ratelimit

import (
	"testing"
	"time"
)

func TestLimiter(t *testing.T) {
	now := time.Date(2026, 10, 14, 20, 0, 0, 0, time.UTC)
	l := New(2, time.Hour)
	l.now = func() time.Time { return now }

	if !l.Allow("a") || !l.Allow("a") {
		t.Fatal("first two events should be allowed")
	}
	if l.Allow("a") {
		t.Error("third event within the window should be rejected")
	}
	if !l.Allow("b") {
		t.Error("another key has its own budget")
	}

	now = now.Add(59 * time.Minute)
	if l.Allow("a") {
		t.Error("events are still within the window after 59 minutes")
	}

	now = now.Add(2 * time.Minute)
	if !l.Allow("a") {
		t.Error("events older than the window should be forgotten")
	}
}

func TestLimiterSweep(t *testing.T) {
	now := time.Date(2026, 10, 14, 20, 0, 0, 0, time.UTC)
	l := New(1, time.Minute)
	l.now = func() time.Time { return now }

	l.Allow("idle")
	now = now.Add(2 * time.Minute)
	for range sweepEvery {
		l.Allow("active")
	}
	if _, ok := l.hits["idle"]; ok {
		t.Error("idle key should have been swept")
	}
}
