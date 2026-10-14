// Package ratelimit limits how many requests a client may send over a sliding window.
// State is kept in memory, which is enough for a single replica.
package ratelimit

import (
	"sync"
	"time"
)

// sweepEvery is how many calls happen between two clean-ups of idle clients.
const sweepEvery = 1000

// Limiter allows at most limit events per key over any window-long period.
// It is safe for concurrent use.
type Limiter struct {
	limit  int
	window time.Duration
	now    func() time.Time // replaced in tests

	mu    sync.Mutex
	hits  map[string][]time.Time // recent event times per key, oldest first
	calls int
}

// New returns a Limiter allowing limit events per key over window.
func New(limit int, window time.Duration) *Limiter {
	return &Limiter{
		limit:  limit,
		window: window,
		now:    time.Now,
		hits:   make(map[string][]time.Time),
	}
}

// Allow records an event for key and reports whether it is within the limit.
// Rejected events are not recorded, so a client that waits is let through again.
func (l *Limiter) Allow(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := l.now()
	l.calls++
	if l.calls%sweepEvery == 0 {
		l.sweep(now)
	}

	recent := l.recent(l.hits[key], now)
	if len(recent) >= l.limit {
		l.hits[key] = recent
		return false
	}
	l.hits[key] = append(recent, now)
	return true
}

// recent drops the events older than the window.
func (l *Limiter) recent(times []time.Time, now time.Time) []time.Time {
	cutoff := now.Add(-l.window)
	i := 0
	for i < len(times) && !times[i].After(cutoff) {
		i++
	}
	return times[i:]
}

// sweep forgets the keys with no recent event, so memory does not grow forever.
func (l *Limiter) sweep(now time.Time) {
	for key, times := range l.hits {
		if len(l.recent(times, now)) == 0 {
			delete(l.hits, key)
		}
	}
}
