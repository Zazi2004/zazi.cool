package migrate_test

import (
	"context"
	"io/fs"
	"testing"

	"zazi.cool/home/internal/dbtest"
	"zazi.cool/home/internal/migrate"
	"zazi.cool/home/migrations"
)

func TestUpOnEmptyDatabaseThenNoop(t *testing.T) {
	pool := dbtest.New(t)
	ctx := context.Background()

	if err := migrate.Up(ctx, pool, migrations.FS); err != nil {
		t.Fatalf("first run: %v", err)
	}
	// Change the default content: if 001_init.sql ran again, it would fail on the
	// existing tables, or the row would be reset.
	if _, err := pool.Exec(ctx, "UPDATE content SET title = 'changed'"); err != nil {
		t.Fatalf("update content: %v", err)
	}
	if err := migrate.Up(ctx, pool, migrations.FS); err != nil {
		t.Fatalf("second run: %v", err)
	}

	var applied int
	if err := pool.QueryRow(ctx, "SELECT count(*) FROM schema_migrations").Scan(&applied); err != nil {
		t.Fatalf("count migrations: %v", err)
	}
	files, err := fs.Glob(migrations.FS, "*.sql")
	if err != nil {
		t.Fatal(err)
	}
	if applied != len(files) {
		t.Errorf("schema_migrations has %d rows, want %d", applied, len(files))
	}
	var title string
	if err := pool.QueryRow(ctx, "SELECT title FROM content").Scan(&title); err != nil {
		t.Fatalf("read content: %v", err)
	}
	if title != "changed" {
		t.Errorf("title = %q after the second run, want it untouched", title)
	}
}

func TestInitialContent(t *testing.T) {
	pool := dbtest.New(t)
	ctx := context.Background()
	if err := migrate.Up(ctx, pool, migrations.FS); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	var title, body, banner, alt, background string
	var version int
	err := pool.QueryRow(ctx, `SELECT title, body, banner_choice, banner_alt, background, version
		FROM content`).Scan(&title, &body, &banner, &alt, &background, &version)
	if err != nil {
		t.Fatalf("read content: %v", err)
	}
	if title != "zazi.cool" || body == "" || banner != "none" || alt != "" || background != "aurore" || version != 1 {
		t.Errorf("default content = %q, %q, %q, %q, %q, %d", title, body, banner, alt, background, version)
	}
}

func TestContentConstraints(t *testing.T) {
	pool := dbtest.New(t)
	ctx := context.Background()
	if err := migrate.Up(ctx, pool, migrations.FS); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	for name, sql := range map[string]string{
		"second row":           "INSERT INTO content (id, title) VALUES (2, 'x')",
		"upload without image": "UPDATE content SET banner_choice = 'upload'",
	} {
		if _, err := pool.Exec(ctx, sql); err == nil {
			t.Errorf("%s: accepted", name)
		}
	}

	var imageID string
	err := pool.QueryRow(ctx, `INSERT INTO images (content_type, data, width, height)
		VALUES ('image/png', '\x00', 1, 1) RETURNING id`).Scan(&imageID)
	if err != nil {
		t.Fatalf("insert image: %v", err)
	}
	if _, err := pool.Exec(ctx, "UPDATE content SET banner_choice = 'upload', upload_image_id = $1", imageID); err != nil {
		t.Errorf("upload with image: %v", err)
	}
}
