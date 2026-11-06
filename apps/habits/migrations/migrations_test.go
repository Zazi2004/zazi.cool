package migrations_test

import (
	"context"
	"testing"
	"testing/fstest"

	"github.com/jackc/pgx/v5/pgxpool"

	"zazi.cool/habits/internal/dbtest"
	"zazi.cool/habits/internal/migrate"
	"zazi.cool/habits/migrations"
)

func tableExists(t *testing.T, pool *pgxpool.Pool, name string) bool {
	t.Helper()
	var exists bool
	err := pool.QueryRow(context.Background(),
		`SELECT EXISTS (SELECT 1 FROM information_schema.tables
		 WHERE table_schema = current_schema() AND table_name = $1)`, name).Scan(&exists)
	if err != nil {
		t.Fatal(err)
	}
	return exists
}

func TestMigration003OnEmptyDatabase(t *testing.T) {
	pool := dbtest.New(t)
	if err := migrate.Up(context.Background(), pool, migrations.FS); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	for _, name := range []string{"habits", "habit_rules", "entries", "day_notes", "sessions"} {
		if !tableExists(t, pool, name) {
			t.Errorf("table %s missing", name)
		}
	}
	for _, name := range []string{"completions", "habit_schedules"} {
		if tableExists(t, pool, name) {
			t.Errorf("v1 table %s still exists", name)
		}
	}
}

// A v1 database (migrations 001 and 002, with data) upgrades: the v1 habit tables go,
// sessions stay.
func TestMigration003OnV1Database(t *testing.T) {
	ctx := context.Background()
	pool := dbtest.New(t)
	v1 := fstest.MapFS{}
	for _, name := range []string{"001_sessions.sql", "002_habits.sql"} {
		data, err := migrations.FS.ReadFile(name)
		if err != nil {
			t.Fatal(err)
		}
		v1[name] = &fstest.MapFile{Data: data}
	}
	if err := migrate.Up(ctx, pool, v1); err != nil {
		t.Fatalf("migrate v1: %v", err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO habits (name, created_on) VALUES ('Lire', '2026-10-30')`); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx,
		`INSERT INTO sessions (token_hash, password_hash_fp, created_at, expires_at)
		 VALUES ('\x01', '\x02', now(), now() + interval '1 day')`); err != nil {
		t.Fatal(err)
	}

	if err := migrate.Up(ctx, pool, migrations.FS); err != nil {
		t.Fatalf("migrate v2: %v", err)
	}
	var sessions, habits int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM sessions`).Scan(&sessions); err != nil || sessions != 1 {
		t.Errorf("sessions = %d, %v; want 1", sessions, err)
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM habits`).Scan(&habits); err != nil || habits != 0 {
		t.Errorf("habits = %d, %v; want the v1 data gone", habits, err)
	}
	if tableExists(t, pool, "completions") || tableExists(t, pool, "habit_schedules") {
		t.Error("v1 tables still exist")
	}
}
