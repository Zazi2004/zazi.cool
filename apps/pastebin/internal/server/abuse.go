package server

import (
	"log/slog"
	"net"
	"net/http"
	"net/netip"
	"strconv"
	"strings"
	"sync"
	"time"
)

// RateWindow is the period over which RATE_LIMIT_CREATE creations are allowed per client.
const RateWindow = time.Hour

// honeypotField is a form field hidden from humans; only bots fill it in.
const honeypotField = "website"

// RateLimiter allows at most limit events per client over a sliding window.
// State lives in memory only (NF4: client IPs are never written to the database),
// which is why the Deployment runs a single replica.
type RateLimiter struct {
	limit  int
	window time.Duration
	now    func() time.Time

	// mu guards the fields below: handlers run concurrently, one goroutine per request.
	mu        sync.Mutex
	hits      map[string][]time.Time
	lastSweep time.Time
}

// NewRateLimiter returns a limiter allowing limit events per client and per window.
func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
	return &RateLimiter{
		limit:  limit,
		window: window,
		now:    time.Now,
		hits:   make(map[string][]time.Time),
	}
}

// Allow records an event for key and reports whether it is within the limit.
// When it is not, it also returns how long until the oldest event leaves the window.
func (l *RateLimiter) Allow(key string) (bool, time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := l.now()
	l.sweep(now)

	recent := dropOlder(l.hits[key], now.Add(-l.window))
	if len(recent) >= l.limit {
		l.hits[key] = recent
		return false, recent[0].Add(l.window).Sub(now)
	}
	l.hits[key] = append(recent, now)
	return true, 0
}

// sweep forgets clients with no event in the window, at most once per window,
// so memory stays bounded without a background goroutine.
func (l *RateLimiter) sweep(now time.Time) {
	if now.Sub(l.lastSweep) < l.window {
		return
	}
	l.lastSweep = now
	cutoff := now.Add(-l.window)
	for key, times := range l.hits {
		if len(dropOlder(times, cutoff)) == 0 {
			delete(l.hits, key)
		}
	}
}

// dropOlder returns the suffix of times (sorted oldest first) that is after cutoff.
func dropOlder(times []time.Time, cutoff time.Time) []time.Time {
	i := 0
	for i < len(times) && !times[i].After(cutoff) {
		i++
	}
	return times[i:]
}

// guardCreate protects paste creation: per-client rate limiting (429 beyond the limit),
// then the honeypot (a filled-in website field is silently dropped). Every attempt counts
// towards the limit, including invalid or honeypot submissions.
func guardCreate(limiter *RateLimiter, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		allowed, retryAfter := limiter.Allow(clientKey(r))
		if !allowed {
			w.Header().Set("Retry-After", strconv.Itoa(int(retryAfter.Seconds())+1))
			writeText(w, http.StatusTooManyRequests,
				"Trop de pastes créés depuis votre adresse. Réessayez plus tard.")
			return
		}
		if r.PostFormValue(honeypotField) != "" {
			slog.Info("honeypot filled, paste dropped")
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// clientKey identifies the client for rate limiting. Behind Traefik, the client IP is
// the last X-Forwarded-For entry (the one Traefik appended; earlier entries come from the
// client and can be forged). Without the header (local run), it is the TCP peer address.
// IPv6 clients are grouped by /64, the block a single subscriber usually gets.
func clientKey(r *http.Request) string {
	raw := lastForwardedFor(r.Header.Values("X-Forwarded-For"))
	if raw == "" {
		host, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			host = r.RemoteAddr
		}
		raw = host
	}
	addr, err := netip.ParseAddr(raw)
	if err != nil {
		return raw
	}
	addr = addr.Unmap()
	if addr.Is6() {
		return netip.PrefixFrom(addr, 64).Masked().String()
	}
	return addr.String()
}

// lastForwardedFor returns the last address of possibly repeated X-Forwarded-For headers.
func lastForwardedFor(values []string) string {
	if len(values) == 0 {
		return ""
	}
	parts := strings.Split(values[len(values)-1], ",")
	return strings.TrimSpace(parts[len(parts)-1])
}
