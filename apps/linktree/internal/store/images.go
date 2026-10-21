package store

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"

	"zazi.cool/linktree/internal/images"
)

// CreateImage stores a processed image and returns its ID.
func (s *Store) CreateImage(ctx context.Context, img images.Image) (string, error) {
	var id string
	err := s.pool.QueryRow(ctx,
		`INSERT INTO images (content_type, data, width, height) VALUES ($1, $2, $3, $4)
		 RETURNING id::text`,
		img.ContentType, img.Data, img.Width, img.Height).Scan(&id)
	if err != nil {
		return "", fmt.Errorf("insert image: %w", err)
	}
	return id, nil
}

// Image returns a stored image. id must be a valid UUID (see pages.IsUUID).
func (s *Store) Image(ctx context.Context, id string) (images.Image, error) {
	var img images.Image
	err := s.pool.QueryRow(ctx,
		`SELECT content_type, data, width, height FROM images WHERE id = $1`, id,
	).Scan(&img.ContentType, &img.Data, &img.Width, &img.Height)
	if errors.Is(err, pgx.ErrNoRows) {
		return images.Image{}, ErrNotFound
	}
	if err != nil {
		return images.Image{}, fmt.Errorf("query image: %w", err)
	}
	return img, nil
}

// DeleteOrphanImages removes the images created before cutoff that no page and no
// item uses any more (replaced images, abandoned uploads), and returns how many.
// The delay leaves time to save a page after uploading its images.
func (s *Store) DeleteOrphanImages(ctx context.Context, cutoff time.Time) (int64, error) {
	tag, err := s.pool.Exec(ctx,
		`DELETE FROM images i
		 WHERE i.created_at < $1
		   AND NOT EXISTS (SELECT 1 FROM pages p WHERE p.avatar_image_id = i.id)
		   AND NOT EXISTS (SELECT 1 FROM items it WHERE it.image_id = i.id)`, cutoff)
	if err != nil {
		return 0, fmt.Errorf("delete orphan images: %w", err)
	}
	return tag.RowsAffected(), nil
}
