package store_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"zazi.cool/linktree/internal/images"
	"zazi.cool/linktree/internal/pages"
	"zazi.cool/linktree/internal/store"
	"zazi.cool/linktree/internal/store/storetest"
)

func uuid(n int) string {
	return fmt.Sprintf("0199e6a8-0000-7000-8000-%012d", n)
}

func link(n int, title string) pages.Item {
	return pages.Item{ID: uuid(n), Kind: pages.KindLink, Title: title, URL: "https://exemple.fr/" + title, Published: true}
}

func mustCreatePage(t *testing.T, s *store.Store, slug string) pages.Page {
	t.Helper()
	p, err := s.CreatePage(context.Background(), "Page "+slug, slug, "star-1")
	if err != nil {
		t.Fatalf("create page %s: %v", slug, err)
	}
	return p
}

func mustSave(t *testing.T, s *store.Store, p pages.Page) pages.Page {
	t.Helper()
	ctx := context.Background()
	if _, err := s.SavePage(ctx, p); err != nil {
		t.Fatalf("save page: %v", err)
	}
	saved, err := s.Page(ctx, p.ID)
	if err != nil {
		t.Fatalf("reload page: %v", err)
	}
	return saved
}

func titles(items []pages.Item) string {
	var parts []string
	for _, it := range items {
		s := it.Title
		if it.Kind == pages.KindGroup {
			s += "[" + titles(it.Links) + "]"
		}
		parts = append(parts, s)
	}
	return strings.Join(parts, ",")
}

func TestCreatePage(t *testing.T) {
	s := storetest.New(t)
	ctx := context.Background()

	p := mustCreatePage(t, s, "musique")
	if p.Published || !p.Listed || p.Theme != pages.DefaultTheme || p.Avatar.Shape != "star-1" || p.Version != 1 {
		t.Errorf("unexpected new page: %+v", p)
	}
	if _, err := s.CreatePage(ctx, "Autre", "musique", ""); !errors.Is(err, store.ErrSlugTaken) {
		t.Errorf("err = %v, want ErrSlugTaken", err)
	}
	if ok, _ := s.SlugAvailable(ctx, "musique", ""); ok {
		t.Error("taken slug reported available")
	}
	if ok, _ := s.SlugAvailable(ctx, "musique", p.ID); !ok {
		t.Error("a page's own slug should be available to it")
	}
	if ok, _ := s.SlugAvailable(ctx, "jeux", ""); !ok {
		t.Error("free slug reported taken")
	}
}

func TestSavePageTree(t *testing.T) {
	s := storetest.New(t)
	p := mustCreatePage(t, s, "musique")

	p.Title = "Musique"
	p.Items = []pages.Item{
		link(1, "a"),
		{ID: uuid(2), Kind: pages.KindGroup, Title: "G", Subtitle: "Artiste", Visual: pages.Visual{Shape: "moon-2"},
			OpenByDefault: true, Published: true, Links: []pages.Item{link(3, "b"), link(4, "c")}},
		link(5, "d"),
	}
	saved := mustSave(t, s, p)
	if got := titles(saved.Items); got != "a,G[b,c],d" {
		t.Fatalf("items = %s", got)
	}
	g := saved.Items[1]
	if g.Subtitle != "Artiste" || !g.OpenByDefault || g.Visual.Shape != "moon-2" || saved.Version != 2 {
		t.Errorf("group or version not saved: %+v, version %d", g, saved.Version)
	}

	// Reorder, move a link out of the group, delete another, add a new one.
	saved.Items = []pages.Item{
		link(5, "d"),
		link(3, "b"),
		{ID: uuid(2), Kind: pages.KindGroup, Title: "G", Published: true, Links: []pages.Item{link(6, "e")}},
	}
	saved = mustSave(t, s, saved)
	if got := titles(saved.Items); got != "d,b,G[e]" {
		t.Fatalf("items after edit = %s", got)
	}

	// Deleting a group deletes the links it still contains.
	saved.Items = saved.Items[:2]
	saved = mustSave(t, s, saved)
	if got := titles(saved.Items); got != "d,b" {
		t.Fatalf("items after group deletion = %s", got)
	}
}

func TestSavePageConflicts(t *testing.T) {
	s := storetest.New(t)
	ctx := context.Background()
	musique := mustCreatePage(t, s, "musique")
	jeux := mustCreatePage(t, s, "jeux")

	// Stale version: another tab saved first.
	if _, err := s.SavePage(ctx, musique); err != nil {
		t.Fatal(err)
	}
	if _, err := s.SavePage(ctx, musique); !errors.Is(err, store.ErrVersionConflict) {
		t.Errorf("err = %v, want ErrVersionConflict", err)
	}

	jeux.Items = []pages.Item{link(1, "a")}
	jeux = mustSave(t, s, jeux)

	// An item ID of another page must not be stolen.
	musique, _ = s.Page(ctx, musique.ID)
	musique.Items = []pages.Item{link(1, "volé")}
	if _, err := s.SavePage(ctx, musique); !errors.Is(err, store.ErrForeignItem) {
		t.Errorf("err = %v, want ErrForeignItem", err)
	}
	if got, _ := s.Page(ctx, jeux.ID); titles(got.Items) != "a" {
		t.Errorf("the other page was modified: %s", titles(got.Items))
	}

	musique.Items = nil
	musique.Slug = "jeux"
	if _, err := s.SavePage(ctx, musique); !errors.Is(err, store.ErrSlugTaken) {
		t.Errorf("err = %v, want ErrSlugTaken", err)
	}

	musique.Slug = "musique"
	musique.Avatar = pages.Visual{ImageID: uuid(999)}
	if _, err := s.SavePage(ctx, musique); !errors.Is(err, store.ErrImageNotFound) {
		t.Errorf("err = %v, want ErrImageNotFound", err)
	}

	missing := musique
	missing.ID = uuid(12345)
	if _, err := s.SavePage(ctx, missing); !errors.Is(err, store.ErrNotFound) {
		t.Errorf("err = %v, want ErrNotFound", err)
	}
}

func TestPublishedPageAndIndex(t *testing.T) {
	s := storetest.New(t)
	ctx := context.Background()
	a := mustCreatePage(t, s, "a")
	b := mustCreatePage(t, s, "b")
	c := mustCreatePage(t, s, "c")

	if _, err := s.PublishedPage(ctx, "a"); !errors.Is(err, store.ErrNotFound) {
		t.Errorf("draft page: err = %v, want ErrNotFound", err)
	}
	a.Published, b.Published, c.Published = true, true, true
	c.Listed = false
	for _, p := range []pages.Page{a, b, c} {
		mustSave(t, s, p)
	}
	if _, err := s.PublishedPage(ctx, "c"); err != nil {
		t.Errorf("unlisted but published page should be reachable: %v", err)
	}

	if err := s.ReorderPages(ctx, []string{c.ID, b.ID, a.ID}); err != nil {
		t.Fatal(err)
	}
	public, err := s.PublicPages(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(public) != 2 || public[0].Slug != "b" || public[1].Slug != "a" {
		t.Errorf("public index = %+v, want b, a", public)
	}
	all, err := s.Pages(ctx)
	if err != nil || len(all) != 3 || all[0].Slug != "c" {
		t.Errorf("admin list = %+v, %v", all, err)
	}

	for _, bad := range [][]string{{a.ID, b.ID}, {a.ID, a.ID, b.ID}, {a.ID, b.ID, uuid(42)}} {
		if err := s.ReorderPages(ctx, bad); !errors.Is(err, store.ErrInvalidOrder) {
			t.Errorf("ReorderPages(%v) = %v, want ErrInvalidOrder", bad, err)
		}
	}
}

func TestImages(t *testing.T) {
	s := storetest.New(t)
	ctx := context.Background()

	img := images.Image{Data: []byte{1, 2, 3}, ContentType: "image/png", Width: 10, Height: 20}
	used, err := s.CreateImage(ctx, img)
	if err != nil {
		t.Fatal(err)
	}
	orphan, err := s.CreateImage(ctx, img)
	if err != nil {
		t.Fatal(err)
	}
	got, err := s.Image(ctx, used)
	if err != nil || !bytes.Equal(got.Data, img.Data) || got.Width != 10 {
		t.Fatalf("Image = %+v, %v", got, err)
	}

	p := mustCreatePage(t, s, "musique")
	p.Avatar = pages.Visual{ImageID: used}
	saved := mustSave(t, s, p)
	if saved.Avatar.Width != 10 || saved.Avatar.Height != 20 {
		t.Errorf("avatar dimensions not loaded: %+v", saved.Avatar)
	}

	// Recent orphans are kept: their page may not be saved yet.
	if n, err := s.DeleteOrphanImages(ctx, time.Now().Add(-time.Hour)); err != nil || n != 0 {
		t.Errorf("deleted %d recent images, %v", n, err)
	}
	if n, err := s.DeleteOrphanImages(ctx, time.Now().Add(time.Hour)); err != nil || n != 1 {
		t.Errorf("deleted %d images, want 1 (%v)", n, err)
	}
	if _, err := s.Image(ctx, orphan); !errors.Is(err, store.ErrNotFound) {
		t.Errorf("orphan still there: %v", err)
	}
	if _, err := s.Image(ctx, used); err != nil {
		t.Errorf("used image deleted: %v", err)
	}
}

func TestSessions(t *testing.T) {
	s := storetest.New(t)
	ctx := context.Background()
	now := time.Now()
	token, fp := []byte("token-hash"), []byte("fingerprint")

	if err := s.CreateSession(ctx, token, fp, now.Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	if ok, err := s.SessionValid(ctx, token, fp, now); err != nil || !ok {
		t.Errorf("valid session: %v, %v", ok, err)
	}
	if ok, _ := s.SessionValid(ctx, token, []byte("other"), now); ok {
		t.Error("wrong fingerprint accepted")
	}
	if ok, _ := s.SessionValid(ctx, token, fp, now.Add(2*time.Hour)); ok {
		t.Error("expired session accepted")
	}
	if n, err := s.DeleteExpiredSessions(ctx, now.Add(2*time.Hour)); err != nil || n != 1 {
		t.Errorf("deleted %d sessions, %v", n, err)
	}
}

func TestDeletePage(t *testing.T) {
	s := storetest.New(t)
	ctx := context.Background()
	p := mustCreatePage(t, s, "musique")
	p.Items = []pages.Item{link(1, "a")}
	mustSave(t, s, p)

	if err := s.DeletePage(ctx, p.ID); err != nil {
		t.Fatal(err)
	}
	if err := s.DeletePage(ctx, p.ID); !errors.Is(err, store.ErrNotFound) {
		t.Errorf("err = %v, want ErrNotFound", err)
	}
	// Its items are gone too: the ID can be reused by another page.
	q := mustCreatePage(t, s, "jeux")
	q.Items = []pages.Item{link(1, "a")}
	mustSave(t, s, q)
}
