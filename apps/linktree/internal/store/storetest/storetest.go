// Package storetest provides a real PostgreSQL store for tests, in a temporary
// schema. It needs TEST_DATABASE_URL (set by `make test`); without it, tests skip.
package storetest

import (
	"context"
	"crypto/rand"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"zazi.cool/linktree/internal/migrate"
	"zazi.cool/linktree/internal/store"
	"zazi.cool/linktree/migrations"
)

// New returns a Store backed by a fresh, migrated schema that is dropped at the end
// of the test, so tests never touch the development data.
func New(t testing.TB) *store.Store {
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
	return store.New(pool)
}
