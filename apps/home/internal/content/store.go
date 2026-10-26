package content

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Store reads the content from PostgreSQL.
type Store struct {
	db *pgxpool.Pool
}

// NewStore returns a Store on db.
func NewStore(db *pgxpool.Pool) *Store {
	return &Store{db: db}
}

// Load reads the content row, with the dimensions of the uploaded image if any.
// It is called on every request (NF9): one indexed row, no cache to invalidate.
func (s *Store) Load(ctx context.Context) (Content, error) {
	var (
		c             Content
		imageID       *string
		width, height *int
	)
	err := s.db.QueryRow(ctx, `
		SELECT c.title, c.body, c.banner_choice, c.banner_alt, c.background, c.version,
		       i.id::text, i.width, i.height
		FROM content c
		LEFT JOIN images i ON i.id = c.upload_image_id
		WHERE c.id = 1`,
	).Scan(&c.Title, &c.Body, &c.BannerChoice, &c.BannerAlt, &c.Background, &c.Version,
		&imageID, &width, &height)
	if errors.Is(err, pgx.ErrNoRows) {
		return Content{}, errors.New("load content: the content row is missing (migrations not applied?)")
	}
	if err != nil {
		return Content{}, fmt.Errorf("load content: %w", err)
	}
	if imageID != nil {
		c.Upload = &Image{ID: *imageID, Width: *width, Height: *height}
	}
	return c, nil
}
