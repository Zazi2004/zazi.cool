package store

import (
	"context"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestPing(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL not set: skipping PostgreSQL integration test")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer pool.Close()

	if err := New(pool).Ping(ctx); err != nil {
		t.Errorf("Ping: %v", err)
	}

	pool.Close()
	if err := New(pool).Ping(ctx); err == nil {
		t.Error("Ping on a closed pool: want an error")
	}
}
