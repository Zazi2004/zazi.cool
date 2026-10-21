package store

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"zazi.cool/linktree/internal/pages"
)

// nullable turns "" into SQL NULL, for optional columns.
func nullable(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

const summaryQuery = `
SELECT p.id::text, p.slug, p.title, p.description,
       COALESCE(p.avatar_image_id::text, ''), COALESCE(p.avatar_shape, ''),
       COALESCE(i.width, 0), COALESCE(i.height, 0),
       p.published, p.listed,
       (SELECT count(*) FROM items it WHERE it.page_id = p.id AND it.kind = 'link')
FROM pages p LEFT JOIN images i ON i.id = p.avatar_image_id`

// Pages returns every page, in the order of the public index.
func (s *Store) Pages(ctx context.Context) ([]pages.Summary, error) {
	return s.summaries(ctx, summaryQuery+` ORDER BY p.position, p.created_at`)
}

// PublicPages returns the published and listed pages, in the order of the index.
func (s *Store) PublicPages(ctx context.Context) ([]pages.Summary, error) {
	return s.summaries(ctx, summaryQuery+` WHERE p.published AND p.listed ORDER BY p.position, p.created_at`)
}

func (s *Store) summaries(ctx context.Context, query string) ([]pages.Summary, error) {
	rows, err := s.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("query pages: %w", err)
	}
	// CollectRows reads every row with the function, then closes rows.
	list, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (pages.Summary, error) {
		var p pages.Summary
		err := row.Scan(&p.ID, &p.Slug, &p.Title, &p.Description,
			&p.Avatar.ImageID, &p.Avatar.Shape, &p.Avatar.Width, &p.Avatar.Height,
			&p.Published, &p.Listed, &p.LinkCount)
		return p, err
	})
	if err != nil {
		return nil, fmt.Errorf("read pages: %w", err)
	}
	return list, nil
}

// CreatePage creates a draft page at the end of the index and returns it.
func (s *Store) CreatePage(ctx context.Context, title, slug, avatarShape string) (pages.Page, error) {
	var id string
	err := s.pool.QueryRow(ctx,
		`INSERT INTO pages (slug, title, theme, accent, avatar_shape, position)
		 VALUES ($1, $2, $3, $4, $5, (SELECT COALESCE(MAX(position), 0) + 1 FROM pages))
		 RETURNING id::text`,
		slug, title, pages.DefaultTheme, pages.DefaultAccent, nullable(avatarShape)).Scan(&id)
	if err != nil {
		return pages.Page{}, fmt.Errorf("insert page: %w", translate(err))
	}
	return s.Page(ctx, id)
}

const pageQuery = `
SELECT p.id::text, p.slug, p.title, p.description,
       COALESCE(p.avatar_image_id::text, ''), COALESCE(p.avatar_shape, ''),
       COALESCE(i.width, 0), COALESCE(i.height, 0),
       p.theme, p.accent, p.published, p.listed, p.version
FROM pages p LEFT JOIN images i ON i.id = p.avatar_image_id`

// Page returns the full document of a page, drafts included (admin). id must be a
// valid UUID.
func (s *Store) Page(ctx context.Context, id string) (pages.Page, error) {
	return s.page(ctx, pageQuery+` WHERE p.id = $1`, id)
}

// PublishedPage returns the page with this slug if it is published, draft items
// included: call Visible before showing it. A draft page is reported as not found.
func (s *Store) PublishedPage(ctx context.Context, slug string) (pages.Page, error) {
	return s.page(ctx, pageQuery+` WHERE p.slug = $1 AND p.published`, slug)
}

func (s *Store) page(ctx context.Context, query string, arg string) (pages.Page, error) {
	var p pages.Page
	err := s.pool.QueryRow(ctx, query, arg).Scan(&p.ID, &p.Slug, &p.Title, &p.Description,
		&p.Avatar.ImageID, &p.Avatar.Shape, &p.Avatar.Width, &p.Avatar.Height,
		&p.Theme, &p.Accent, &p.Published, &p.Listed, &p.Version)
	if errors.Is(err, pgx.ErrNoRows) {
		return pages.Page{}, ErrNotFound
	}
	if err != nil {
		return pages.Page{}, fmt.Errorf("query page: %w", err)
	}
	if p.Items, err = s.items(ctx, p.ID); err != nil {
		return pages.Page{}, err
	}
	return p, nil
}

// items loads the items of a page as a tree: groups hold their links.
func (s *Store) items(ctx context.Context, pageID string) ([]pages.Item, error) {
	// Top-level items come first (parent_id NULLS FIRST), so a group is always
	// known before its links.
	rows, err := s.pool.Query(ctx, `
		SELECT it.id::text, COALESCE(it.parent_id::text, ''), it.kind, it.title, it.subtitle,
		       COALESCE(it.url, ''), COALESCE(it.image_id::text, ''), COALESCE(it.shape, ''),
		       COALESCE(im.width, 0), COALESCE(im.height, 0), it.open_by_default, it.published
		FROM items it LEFT JOIN images im ON im.id = it.image_id
		WHERE it.page_id = $1
		ORDER BY it.parent_id NULLS FIRST, it.position`, pageID)
	if err != nil {
		return nil, fmt.Errorf("query items: %w", err)
	}
	defer rows.Close()

	var top []pages.Item
	groupIndex := map[string]int{} // group ID → index in top
	for rows.Next() {
		var it pages.Item
		var parentID string
		if err := rows.Scan(&it.ID, &parentID, &it.Kind, &it.Title, &it.Subtitle, &it.URL,
			&it.Visual.ImageID, &it.Visual.Shape, &it.Visual.Width, &it.Visual.Height,
			&it.OpenByDefault, &it.Published); err != nil {
			return nil, fmt.Errorf("read item: %w", err)
		}
		if parentID == "" {
			if it.Kind == pages.KindGroup {
				groupIndex[it.ID] = len(top)
			}
			top = append(top, it)
			continue
		}
		// Indexes rather than pointers: append may move the slice in memory.
		i, ok := groupIndex[parentID]
		if !ok {
			return nil, fmt.Errorf("item %s: parent %s is not a group of the page", it.ID, parentID)
		}
		top[i].Links = append(top[i].Links, it)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read items: %w", err)
	}
	return top, nil
}

// SavePage replaces the document of a page in one transaction and returns its new
// version. p must have been validated (pages.Validate) and carry the version it
// was loaded with: ErrVersionConflict is returned if it was saved since.
func (s *Store) SavePage(ctx context.Context, p pages.Page) (int, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("begin: %w", err)
	}
	// Rollback after a successful Commit does nothing: deferring it makes every
	// early return below undo the transaction.
	defer func() { _ = tx.Rollback(ctx) }()

	// FOR UPDATE locks the row until the end of the transaction, so two saves of
	// the same page cannot both pass the version check.
	var version int
	err = tx.QueryRow(ctx, `SELECT version FROM pages WHERE id = $1 FOR UPDATE`, p.ID).Scan(&version)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, ErrNotFound
	}
	if err != nil {
		return 0, fmt.Errorf("lock page: %w", err)
	}
	if version != p.Version {
		return 0, ErrVersionConflict
	}

	_, err = tx.Exec(ctx, `
		UPDATE pages SET slug = $2, title = $3, description = $4, avatar_image_id = $5,
		       avatar_shape = $6, theme = $7, accent = $8, published = $9, listed = $10,
		       version = version + 1, updated_at = now()
		WHERE id = $1`,
		p.ID, p.Slug, p.Title, p.Description, nullable(p.Avatar.ImageID), nullable(p.Avatar.Shape),
		p.Theme, p.Accent, p.Published, p.Listed)
	if err != nil {
		return 0, fmt.Errorf("update page: %w", translate(err))
	}

	ids, err := upsertItems(ctx, tx, p)
	if err != nil {
		return 0, err
	}
	// Items missing from the document were deleted in the editor. The links of a
	// deleted group go with it (ON DELETE CASCADE) unless they were moved, in which
	// case their parent_id was already updated above.
	if _, err := tx.Exec(ctx,
		`DELETE FROM items WHERE page_id = $1 AND NOT (id = ANY($2::uuid[]))`, p.ID, ids); err != nil {
		return 0, fmt.Errorf("delete removed items: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("commit: %w", translate(err))
	}
	return version + 1, nil
}

const upsertItem = `
INSERT INTO items (id, page_id, parent_id, kind, position, title, subtitle, url, image_id, shape,
                   open_by_default, published)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
ON CONFLICT (id) DO UPDATE SET
    parent_id = EXCLUDED.parent_id, kind = EXCLUDED.kind, position = EXCLUDED.position,
    title = EXCLUDED.title, subtitle = EXCLUDED.subtitle, url = EXCLUDED.url,
    image_id = EXCLUDED.image_id, shape = EXCLUDED.shape,
    open_by_default = EXCLUDED.open_by_default, published = EXCLUDED.published
-- An ID already used by another page is not updated: the row count reveals it.
WHERE items.page_id = EXCLUDED.page_id`

// upsertItems inserts or updates every item of p and returns their IDs. Groups are
// written before their links, so a link never points to a missing parent.
func upsertItems(ctx context.Context, tx pgx.Tx, p pages.Page) ([]string, error) {
	// Never nil: "id = ANY(NULL)" would be NULL and the DELETE would keep everything.
	ids := []string{}
	batch := &pgx.Batch{}
	queue := func(it pages.Item, parentID string, position int) {
		ids = append(ids, it.ID)
		var url *string
		if it.Kind == pages.KindLink {
			url = &it.URL // the column is NOT NULL for links
		}
		batch.Queue(upsertItem, it.ID, p.ID, nullable(parentID), string(it.Kind), position,
			it.Title, it.Subtitle, url, nullable(it.Visual.ImageID), nullable(it.Visual.Shape),
			it.OpenByDefault, it.Published)
	}
	for i, it := range p.Items {
		queue(it, "", i)
	}
	for _, group := range p.Items {
		for i, link := range group.Links {
			queue(link, group.ID, i)
		}
	}

	// A batch sends every query in one round trip; results are read in order.
	results := tx.SendBatch(ctx, batch)
	for range ids {
		tag, err := results.Exec()
		if err != nil {
			_ = results.Close()
			return nil, fmt.Errorf("save item: %w", translate(err))
		}
		if tag.RowsAffected() != 1 {
			_ = results.Close()
			return nil, ErrForeignItem
		}
	}
	if err := results.Close(); err != nil {
		return nil, fmt.Errorf("save items: %w", err)
	}
	return ids, nil
}

// DeletePage deletes a page and its items.
func (s *Store) DeletePage(ctx context.Context, id string) error {
	tag, err := s.pool.Exec(ctx, `DELETE FROM pages WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete page: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// ReorderPages sets the order of the public index. ids must list every page once.
func (s *Store) ReorderPages(ctx context.Context, ids []string) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var count int
	if err := tx.QueryRow(ctx, `SELECT count(*) FROM pages`).Scan(&count); err != nil {
		return fmt.Errorf("count pages: %w", err)
	}
	if len(ids) != count {
		return ErrInvalidOrder
	}
	// unnest … WITH ORDINALITY numbers the IDs of the array: 1, 2, 3…
	tag, err := tx.Exec(ctx, `
		UPDATE pages p SET position = o.idx
		FROM unnest($1::uuid[]) WITH ORDINALITY AS o(id, idx)
		WHERE p.id = o.id`, ids)
	if err != nil {
		return fmt.Errorf("reorder pages: %w", err)
	}
	// A duplicated or unknown ID leaves some page without a new position.
	if tag.RowsAffected() != int64(count) {
		return ErrInvalidOrder
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit: %w", err)
	}
	return nil
}

// SlugAvailable reports whether no page other than exceptID uses slug. exceptID
// may be "" (new page).
func (s *Store) SlugAvailable(ctx context.Context, slug, exceptID string) (bool, error) {
	var available bool
	err := s.pool.QueryRow(ctx,
		`SELECT NOT EXISTS (SELECT 1 FROM pages WHERE slug = $1 AND id IS DISTINCT FROM $2::uuid)`,
		slug, nullable(exceptID)).Scan(&available)
	if err != nil {
		return false, fmt.Errorf("check slug: %w", err)
	}
	return available, nil
}
