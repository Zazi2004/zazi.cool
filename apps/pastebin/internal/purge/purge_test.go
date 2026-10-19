package purge

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"
)

// countingPurger counts its calls and fails every other one.
type countingPurger struct {
	calls atomic.Int64
}

func (c *countingPurger) Purge(context.Context) (int64, error) {
	if c.calls.Add(1)%2 == 0 {
		return 0, errors.New("database unavailable")
	}
	return 1, nil
}

func TestRunPurgesAtStartThenPeriodicallyUntilCancelled(t *testing.T) {
	p := &countingPurger{}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		Run(ctx, p, 10*time.Millisecond)
		close(done)
	}()

	// A failing purge must not stop the loop: wait for several calls.
	deadline := time.After(5 * time.Second)
	for p.calls.Load() < 4 {
		select {
		case <-deadline:
			t.Fatalf("only %d purges after 5s, want at least 4", p.calls.Load())
		case <-time.After(5 * time.Millisecond):
		}
	}

	cancel()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("Run did not return after cancel")
	}
}

func TestRunPurgesImmediately(t *testing.T) {
	p := &countingPurger{}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		Run(ctx, p, time.Hour)
		close(done)
	}()

	deadline := time.After(5 * time.Second)
	for p.calls.Load() < 1 {
		select {
		case <-deadline:
			t.Fatal("no purge at start")
		case <-time.After(5 * time.Millisecond):
		}
	}
	cancel()
	<-done
	if n := p.calls.Load(); n != 1 {
		t.Errorf("%d purges with a 1h interval, want 1", n)
	}
}
