package store

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"zazi.cool/pastebin/internal/migrate"
	"zazi.cool/pastebin/internal/paste"
	"zazi.cool/pastebin/migrations"
)

// newTestStore returns a store on a fresh, migrated schema, dropped at the end of the
// test. It needs TEST_DATABASE_URL (set by `make test`).
func newTestStore(t *testing.T) (*Store, *pgxpool.Pool) {
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

	if err := migrate.Up(ctx, pool, migrations.FS); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return New(pool), pool
}

// CA1: the stored paste reads back unchanged and expires exactly ttl after creation.
func TestCreateThenGet(t *testing.T) {
	s, _ := newTestStore(t)
	ctx := context.Background()
	content := "line 1\r\nligne 2 é 🙂\n<script>alert(1)</script>\n"

	created, err := s.Create(ctx, content, "go", 10*time.Minute)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if !paste.ValidID(created.ID) {
		t.Errorf("ID = %q, not 8 base62 characters", created.ID)
	}
	if d := created.ExpiresAt.Sub(created.CreatedAt); d != 10*time.Minute {
		t.Errorf("expires_at - created_at = %v, want 10m", d)
	}

	got, err := s.Get(ctx, created.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Content != content || got.Language != "go" {
		t.Errorf("Get = %q (%s), want %q (go)", got.Content, got.Language, content)
	}
	if !got.ExpiresAt.Equal(created.ExpiresAt) {
		t.Errorf("ExpiresAt = %v, want %v", got.ExpiresAt, created.ExpiresAt)
	}
}

// CA3: an expired paste that is still in the table is reported as not found, like an
// unknown identifier.
func TestGetHidesExpiredAndUnknownPastes(t *testing.T) {
	s, pool := newTestStore(t)
	ctx := context.Background()

	if _, err := pool.Exec(ctx, `INSERT INTO pastes (id, content, expires_at)
		VALUES ('expired1', 'old', now() - interval '1 second')`); err != nil {
		t.Fatal(err)
	}

	for _, id := range []string{"expired1", "unknown1"} {
		if _, err := s.Get(ctx, id); !errors.Is(err, ErrNotFound) {
			t.Errorf("Get(%q) error = %v, want ErrNotFound", id, err)
		}
	}
}

// CA4: a purge removes the expired pastes, expired or not yet read, and leaves the
// others untouched.
func TestPurgeDeletesOnlyExpiredPastes(t *testing.T) {
	s, pool := newTestStore(t)
	ctx := context.Background()

	if _, err := pool.Exec(ctx, `INSERT INTO pastes (id, content, expires_at) VALUES
		('expired1', 'old', now() - interval '1 second'),
		('expired2', 'older', now() - interval '2 days')`); err != nil {
		t.Fatal(err)
	}
	alive, err := s.Create(ctx, "still here", "plaintext", 10*time.Minute)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	n, err := s.Purge(ctx)
	if err != nil {
		t.Fatalf("Purge: %v", err)
	}
	if n != 2 {
		t.Errorf("Purge removed %d pastes, want 2", n)
	}

	var ids []string
	rows, err := pool.Query(ctx, `SELECT id FROM pastes ORDER BY id`)
	if err != nil {
		t.Fatal(err)
	}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			t.Fatal(err)
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	if len(ids) != 1 || ids[0] != alive.ID {
		t.Errorf("pastes left = %v, want only %s", ids, alive.ID)
	}
	if got, err := s.Get(ctx, alive.ID); err != nil || got.Content != "still here" {
		t.Errorf("Get(%s) = %q, %v; want the untouched paste", alive.ID, got.Content, err)
	}

	if n, err := s.Purge(ctx); err != nil || n != 0 {
		t.Errorf("second Purge = %d, %v; want 0, nil", n, err)
	}
}
