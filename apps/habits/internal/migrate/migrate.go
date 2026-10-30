// Package migrate applies versioned SQL migrations. Each *.sql file runs once, in
// lexical order, inside its own transaction; applied versions are recorded in the
// schema_migrations table.
package migrate

import (
	"context"
	"fmt"
	"io/fs"
	"log/slog"
	"path"
	"sort"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// lockID identifies the advisory lock that prevents two replicas from migrating at
// the same time. Any constant works as long as it is not reused for another purpose.
const lockID = 727274

// Up applies every migration of fsys that has not been applied yet.
func Up(ctx context.Context, pool *pgxpool.Pool, fsys fs.FS) error {
	files, err := fs.Glob(fsys, "*.sql")
	if err != nil {
		return fmt.Errorf("list migrations: %w", err)
	}
	sort.Strings(files)

	conn, err := pool.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("acquire connection: %w", err)
	}
	defer conn.Release()

	if _, err := conn.Exec(ctx, "SELECT pg_advisory_lock($1)", lockID); err != nil {
		return fmt.Errorf("take migration lock: %w", err)
	}
	// The lock belongs to the session, so it must be released on the same connection.
	defer func() {
		if _, err := conn.Exec(context.WithoutCancel(ctx), "SELECT pg_advisory_unlock($1)", lockID); err != nil {
			slog.Error("release migration lock", "error", err)
		}
	}()

	if _, err := conn.Exec(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations (
		version    text        PRIMARY KEY,
		applied_at timestamptz NOT NULL DEFAULT now()
	)`); err != nil {
		return fmt.Errorf("create schema_migrations: %w", err)
	}

	for _, file := range files {
		version := path.Base(file)
		if err := apply(ctx, conn.Conn(), fsys, file, version); err != nil {
			return err
		}
	}
	return nil
}

func apply(ctx context.Context, conn *pgx.Conn, fsys fs.FS, file, version string) error {
	var applied bool
	err := conn.QueryRow(ctx,
		"SELECT EXISTS (SELECT 1 FROM schema_migrations WHERE version = $1)", version,
	).Scan(&applied)
	if err != nil {
		return fmt.Errorf("check migration %s: %w", version, err)
	}
	if applied {
		return nil
	}

	sql, err := fs.ReadFile(fsys, file)
	if err != nil {
		return fmt.Errorf("read migration %s: %w", version, err)
	}

	// pgx.BeginFunc commits if the function returns nil and rolls back otherwise,
	// so a failing migration leaves no partial change behind.
	err = pgx.BeginFunc(ctx, conn, func(tx pgx.Tx) error {
		// Without arguments, pgx uses the simple protocol, which accepts a file
		// containing several statements.
		if _, err := tx.Exec(ctx, string(sql)); err != nil {
			return err
		}
		_, err := tx.Exec(ctx, "INSERT INTO schema_migrations (version) VALUES ($1)", version)
		return err
	})
	if err != nil {
		return fmt.Errorf("apply migration %s: %w", version, err)
	}
	slog.Info("migration applied", "version", version)
	return nil
}
