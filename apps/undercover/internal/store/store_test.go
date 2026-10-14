package store

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"zazi.cool/undercover/internal/migrate"
	"zazi.cool/undercover/internal/pairs"
	"zazi.cool/undercover/internal/seed"
	"zazi.cool/undercover/migrations"
)

// newTestStore returns a Store backed by a fresh, migrated PostgreSQL schema that is
// dropped at the end of the test. It needs TEST_DATABASE_URL (set by `make test`).
func newTestStore(t *testing.T) *Store {
	t.Helper()
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL not set: skipping PostgreSQL integration test")
	}
	ctx := context.Background()

	admin, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	t.Cleanup(admin.Close)

	// Unquoted identifiers are folded to lower case by PostgreSQL, search_path included.
	schema := "test_" + strings.ToLower(rand.Text()[:12])
	if _, err := admin.Exec(ctx, fmt.Sprintf(`CREATE SCHEMA "%s"`, schema)); err != nil {
		t.Fatalf("create schema: %v", err)
	}
	// Cleanups run in reverse order: the test pool closes before the schema is dropped.
	t.Cleanup(func() {
		if _, err := admin.Exec(ctx, fmt.Sprintf(`DROP SCHEMA "%s" CASCADE`, schema)); err != nil {
			t.Errorf("drop schema: %v", err)
		}
	})

	cfg, err := pgxpool.ParseConfig(url)
	if err != nil {
		t.Fatalf("parse config: %v", err)
	}
	cfg.ConnConfig.RuntimeParams["search_path"] = schema
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatalf("connect to schema: %v", err)
	}
	t.Cleanup(pool.Close)

	if err := migrate.Up(ctx, pool, migrations.FS); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	// Running migrations twice must be a no-op.
	if err := migrate.Up(ctx, pool, migrations.FS); err != nil {
		t.Fatalf("migrate again: %v", err)
	}
	return New(pool)
}

func mustCreate(t *testing.T, s *Store, a, b string, extreme bool, source pairs.Source) pairs.Pair {
	t.Helper()
	c, err := pairs.NewCandidate(a, b)
	if err != nil {
		t.Fatal(err)
	}
	p, err := s.CreatePair(context.Background(), c, extreme, source)
	if err != nil {
		t.Fatalf("CreatePair(%s, %s): %v", a, b, err)
	}
	return p
}

func TestCreatePairRejectsDuplicates(t *testing.T) {
	s := newTestStore(t)
	mustCreate(t, s, "Chat", "Tigre", false, pairs.SourceSuggestion)

	c, err := pairs.NewCandidate("les tigres", "chat")
	if err != nil {
		t.Fatal(err)
	}
	_, err = s.CreatePair(context.Background(), c, false, pairs.SourceSuggestion)
	if !errors.Is(err, ErrDuplicate) {
		t.Errorf("CreatePair duplicate: error = %v, want ErrDuplicate", err)
	}
}

func TestRandomPairFilters(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()
	normal := mustCreate(t, s, "Chat", "Tigre", false, pairs.SourceSeed)
	extreme := mustCreate(t, s, "Israël", "Palestine", true, pairs.SourceSeed)
	suggestion := mustCreate(t, s, "Pomme", "Poire", false, pairs.SourceSuggestion)
	hidden := mustCreate(t, s, "Café", "Thé", false, pairs.SourceSuggestion)
	if err := s.Report(ctx, hidden.ID, pairs.ReasonRemove, "someone"); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name    string
		filter  Filter
		allowed []int64
	}{
		{"all-ages only", Filter{}, []int64{normal.ID, suggestion.ID}},
		{"with extreme", Filter{Extreme: ExtremeIncluded}, []int64{normal.ID, extreme.ID, suggestion.ID}},
		{"extreme only", Filter{Extreme: ExtremeOnly}, []int64{extreme.ID}},
		{"suggestions only", Filter{SuggestionsOnly: true}, []int64{suggestion.ID}},
		{"exclusion", Filter{Exclude: []int64{suggestion.ID}}, []int64{normal.ID}},
		{"exclusion ignored when exhausted", Filter{SuggestionsOnly: true, Exclude: []int64{suggestion.ID}}, []int64{suggestion.ID}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// The draw is random: repeat it to make a wrong result very likely to show.
			for range 20 {
				p, err := s.RandomPair(ctx, tt.filter)
				if err != nil {
					t.Fatalf("RandomPair: %v", err)
				}
				if !slices.Contains(tt.allowed, p.ID) {
					t.Fatalf("RandomPair drew pair %d (%s / %s), allowed %v", p.ID, p.WordA, p.WordB, tt.allowed)
				}
			}
		})
	}
}

func TestRandomPairNoMatch(t *testing.T) {
	s := newTestStore(t)
	mustCreate(t, s, "Chat", "Tigre", false, pairs.SourceSeed)

	for _, f := range []Filter{{SuggestionsOnly: true}, {Extreme: ExtremeOnly}} {
		_, err := s.RandomPair(context.Background(), f)
		if !errors.Is(err, ErrNotFound) {
			t.Errorf("RandomPair(%+v): error = %v, want ErrNotFound", f, err)
		}
	}
}

func TestReport(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()
	p := mustCreate(t, s, "Chat", "Tigre", false, pairs.SourceSeed)

	if err := s.Report(ctx, p.ID, pairs.ReasonExtreme, "a"); err != nil {
		t.Fatalf("Report extreme: %v", err)
	}
	// The same report twice is accepted and changes nothing.
	if err := s.Report(ctx, p.ID, pairs.ReasonExtreme, "a"); err != nil {
		t.Fatalf("Report extreme again: %v", err)
	}
	got, err := s.Pair(ctx, p.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !got.Extreme || got.Hidden {
		t.Errorf("after extreme report: extreme=%v hidden=%v, want true/false", got.Extreme, got.Hidden)
	}

	if err := s.Report(ctx, p.ID, pairs.ReasonRemove, "b"); err != nil {
		t.Fatalf("Report remove: %v", err)
	}
	got, err = s.Pair(ctx, p.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !got.Hidden {
		t.Error("after remove report: pair is not hidden")
	}

	if err := s.Report(ctx, 999999, pairs.ReasonRemove, "a"); !errors.Is(err, ErrNotFound) {
		t.Errorf("Report unknown pair: error = %v, want ErrNotFound", err)
	}
}

func TestSeedIsIdempotent(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()
	ps, err := seed.Pairs()
	if err != nil {
		t.Fatal(err)
	}

	n, err := s.Seed(ctx, ps)
	if err != nil {
		t.Fatalf("Seed: %v", err)
	}
	if n != len(ps) {
		t.Errorf("first Seed inserted %d pairs, want %d", n, len(ps))
	}
	n, err = s.Seed(ctx, ps)
	if err != nil {
		t.Fatalf("Seed again: %v", err)
	}
	if n != 0 {
		t.Errorf("second Seed inserted %d pairs, want 0", n)
	}

	all, err := s.AllPairs(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != len(ps) {
		t.Errorf("AllPairs returned %d pairs, want %d", len(all), len(ps))
	}
}
