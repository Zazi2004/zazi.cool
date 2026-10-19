package migrate

import (
	"context"
	"crypto/rand"
	"fmt"
	"os"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/jackc/pgx/v5/pgxpool"
)

// newTestPool returns a pool bound to a fresh schema, dropped at the end of the test.
// It needs TEST_DATABASE_URL (set by `make test`).
func newTestPool(t *testing.T) *pgxpool.Pool {
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

	schema := "test_" + strings.ToLower(rand.Text()[:12])
	if _, err := admin.Exec(ctx, fmt.Sprintf(`CREATE SCHEMA "%s"`, schema)); err != nil {
		t.Fatalf("create schema: %v", err)
	}
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
	return pool
}

func count(t *testing.T, pool *pgxpool.Pool, query string) int {
	t.Helper()
	var n int
	if err := pool.QueryRow(context.Background(), query).Scan(&n); err != nil {
		t.Fatalf("%s: %v", query, err)
	}
	return n
}

func TestUpAppliesEachMigrationOnce(t *testing.T) {
	pool := newTestPool(t)
	ctx := context.Background()
	fsys := fstest.MapFS{
		"001_create_things.sql": {Data: []byte(`CREATE TABLE things (id int); INSERT INTO things VALUES (1);`)},
		"002_add_more.sql":      {Data: []byte(`INSERT INTO things VALUES (2);`)},
		"notes.txt":             {Data: []byte("ignored")},
	}

	for i := range 2 { // the second run must be a no-op
		if err := Up(ctx, pool, fsys); err != nil {
			t.Fatalf("Up (run %d): %v", i+1, err)
		}
	}

	if got := count(t, pool, "SELECT count(*) FROM things"); got != 2 {
		t.Errorf("things rows = %d, want 2 (a migration ran twice?)", got)
	}
	if got := count(t, pool, "SELECT count(*) FROM schema_migrations"); got != 2 {
		t.Errorf("schema_migrations rows = %d, want 2", got)
	}
}

func TestUpRollsBackAFailingMigration(t *testing.T) {
	pool := newTestPool(t)
	ctx := context.Background()
	fsys := fstest.MapFS{
		"001_broken.sql": {Data: []byte(`CREATE TABLE half_done (id int); SELECT * FROM missing_table;`)},
	}

	if err := Up(ctx, pool, fsys); err == nil {
		t.Fatal("Up succeeded, want an error")
	}
	if got := count(t, pool, "SELECT count(*) FROM pg_tables WHERE tablename = 'half_done' AND schemaname = current_schema()"); got != 0 {
		t.Error("table of the failed migration still exists")
	}
	if got := count(t, pool, "SELECT count(*) FROM schema_migrations"); got != 0 {
		t.Errorf("schema_migrations rows = %d, want 0", got)
	}
}

func TestUpWithoutMigrations(t *testing.T) {
	pool := newTestPool(t)
	if err := Up(context.Background(), pool, fstest.MapFS{}); err != nil {
		t.Fatalf("Up: %v", err)
	}
}
