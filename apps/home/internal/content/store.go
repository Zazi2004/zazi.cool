package content

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"zazi.cool/home/internal/images"
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

// ErrConflict means the content was saved by someone else since it was read (F12).
var ErrConflict = errors.New("content changed since it was read")

// Save stores e if the content is still at version, and raises the version. One
// UPDATE is atomic, and its WHERE clause is the optimistic lock: when another
// save got in first, no row matches and ErrConflict is returned.
// A non-nil img is stored in the same transaction and becomes the uploaded image
// of the page: either both are written, or neither (a conflict leaves no orphan).
func (s *Store) Save(ctx context.Context, version int, e Edit, img *images.Image) error {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("save content: %w", err)
	}
	// Rollback after a successful Commit does nothing, so deferring it covers every
	// early return.
	defer tx.Rollback(ctx) //nolint:errcheck

	var imageID *string
	if img != nil {
		var id string
		err := tx.QueryRow(ctx,
			`INSERT INTO images (content_type, data, width, height) VALUES ($1, $2, $3, $4)
			 RETURNING id::text`,
			img.ContentType, img.Data, img.Width, img.Height).Scan(&id)
		if err != nil {
			return fmt.Errorf("insert image: %w", err)
		}
		imageID = &id
	}
	// COALESCE keeps the current image when no new one is sent: choosing an
	// illustration or "none" does not forget the uploaded image (F16).
	tag, err := tx.Exec(ctx, `
		UPDATE content
		SET title = $1, body = $2, banner_choice = $3, banner_alt = $4, background = $5,
		    upload_image_id = COALESCE($7::uuid, upload_image_id),
		    version = version + 1, updated_at = now()
		WHERE id = 1 AND version = $6`,
		e.Title, e.Body, e.BannerChoice, e.BannerAlt, e.Background, version, imageID)
	if err != nil {
		return fmt.Errorf("save content: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrConflict
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("save content: %w", err)
	}
	return nil
}

// ErrImageNotFound means no image has the requested id.
var ErrImageNotFound = errors.New("image not found")

// Image returns a stored image. id must be a valid UUID (see IsUUID).
func (s *Store) Image(ctx context.Context, id string) (images.Image, error) {
	var img images.Image
	err := s.db.QueryRow(ctx,
		`SELECT content_type, data, width, height FROM images WHERE id = $1`, id,
	).Scan(&img.ContentType, &img.Data, &img.Width, &img.Height)
	if errors.Is(err, pgx.ErrNoRows) {
		return images.Image{}, ErrImageNotFound
	}
	if err != nil {
		return images.Image{}, fmt.Errorf("query image: %w", err)
	}
	return img, nil
}

// DeleteOrphanImages removes the images created before cutoff that the page does
// not use as its uploaded image (replaced images, images of a save that failed),
// and returns how many (F16). The delay leaves time to choose "Mon image" again.
// Running it twice in a row deletes nothing the second time.
func (s *Store) DeleteOrphanImages(ctx context.Context, cutoff time.Time) (int64, error) {
	tag, err := s.db.Exec(ctx, `
		DELETE FROM images i
		WHERE i.created_at < $1
		  AND NOT EXISTS (SELECT 1 FROM content c WHERE c.upload_image_id = i.id)`, cutoff)
	if err != nil {
		return 0, fmt.Errorf("delete orphan images: %w", err)
	}
	return tag.RowsAffected(), nil
}

var uuidPattern = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

// IsUUID reports whether s is a lowercase UUID. PostgreSQL would fail on any other
// string, so the handler checks first and answers 404.
func IsUUID(s string) bool { return uuidPattern.MatchString(s) }
