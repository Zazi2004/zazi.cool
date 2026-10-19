// Package purge periodically deletes expired pastes (docs/specs/pastebin.md, F6).
package purge

import (
	"context"
	"log/slog"
	"time"
)

// DefaultInterval is the period of the purge when PURGE_INTERVAL is not set.
const DefaultInterval = 15 * time.Minute

// MaxInterval is the longest allowed period: F6 requires an expired paste to be gone
// at most one hour after its expiration.
const MaxInterval = time.Hour

// Purger deletes expired pastes and reports how many it removed.
type Purger interface {
	Purge(ctx context.Context) (int64, error)
}

// Run purges once right away, so a restarted pod does not wait a full interval, then
// every interval until ctx is cancelled. A failed purge is logged and retried at the
// next tick: the expired rows are already hidden from readers (F5).
func Run(ctx context.Context, p Purger, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		once(ctx, p)
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func once(ctx context.Context, p Purger) {
	n, err := p.Purge(ctx)
	switch {
	case ctx.Err() != nil:
		// Shutting down: the error, if any, only reflects the cancellation.
	case err != nil:
		slog.ErrorContext(ctx, "purge failed", "error", err)
	case n > 0:
		slog.InfoContext(ctx, "expired pastes purged", "count", n)
	}
}
