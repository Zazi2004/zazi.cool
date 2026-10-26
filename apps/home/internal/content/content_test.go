package content

import (
	"context"
	"reflect"
	"regexp"
	"strings"
	"testing"
	"unicode/utf8"

	"zazi.cool/home/internal/dbtest"
	"zazi.cool/home/internal/migrate"
	"zazi.cool/home/migrations"
)

func TestParagraphs(t *testing.T) {
	tests := []struct {
		body string
		want [][]string
	}{
		{"", nil},
		{"\n \n", nil},
		{"un", [][]string{{"un"}}},
		{"un\ndeux", [][]string{{"un", "deux"}}},
		{"un\n\ndeux", [][]string{{"un"}, {"deux"}}},
		{"un\r\n\r\ndeux\r\ntrois", [][]string{{"un"}, {"deux", "trois"}}},
		{"\n\nun  \n\t\n\n\ndeux\n", [][]string{{"un"}, {"deux"}}},
		{"<script>alert(1)</script>\n\ndeux", [][]string{{"<script>alert(1)</script>"}, {"deux"}}},
	}
	for _, tt := range tests {
		if got := Paragraphs(tt.body); !reflect.DeepEqual(got, tt.want) {
			t.Errorf("Paragraphs(%q) = %q, want %q", tt.body, got, tt.want)
		}
	}
}

func TestSummary(t *testing.T) {
	if got := Summary("  un\n\ndeux\ttrois ", 200); got != "un deux trois" {
		t.Errorf("Summary = %q, want whitespace collapsed", got)
	}
	long := strings.Repeat("é", 250)
	got := Summary(long, 200)
	if n := utf8.RuneCountInString(got); n != 200 {
		t.Errorf("Summary has %d characters, want 200", n)
	}
	if !strings.HasSuffix(got, "…") {
		t.Errorf("a cut summary does not end with an ellipsis: %q", got)
	}
	if got := Summary(strings.Repeat("a", 200), 200); got != strings.Repeat("a", 200) {
		t.Error("a 200-character body is cut")
	}
}

func TestBanner(t *testing.T) {
	upload := &Image{ID: "0193", Width: 1600, Height: 533}
	urls := func(id string) (string, bool) { return "/assets/banners/" + id + ".abc.jpg", id != "lagon" }
	aube, _ := FindIllustration("aube")
	tests := []struct {
		name string
		c    Content
		want *Banner
	}{
		{"none", Content{BannerChoice: "none", Upload: upload}, nil},
		{"upload", Content{BannerChoice: "upload", BannerAlt: "Ciel", Upload: upload},
			&Banner{URL: "/images/0193", Alt: "Ciel", Width: 1600, Height: 533}},
		{"upload without image", Content{BannerChoice: "upload"}, nil},
		{"upload without alt stays decorative", Content{BannerChoice: "upload", Upload: upload},
			&Banner{URL: "/images/0193", Width: 1600, Height: 533}},
		{"illustration with its default alt", Content{BannerChoice: "aube", Upload: upload},
			&Banner{URL: "/assets/banners/aube.abc.jpg", Alt: aube.Alt, Width: 1600, Height: 533}},
		{"illustration with the saved alt", Content{BannerChoice: "aube", BannerAlt: "Mon texte"},
			&Banner{URL: "/assets/banners/aube.abc.jpg", Alt: "Mon texte", Width: 1600, Height: 533}},
		{"illustration removed from the code", Content{BannerChoice: "disparue", Upload: upload}, nil},
		{"illustration without a picture", Content{BannerChoice: "lagon"}, nil},
	}
	for _, tt := range tests {
		if got := tt.c.Banner(urls); !reflect.DeepEqual(got, tt.want) {
			t.Errorf("%s: Banner() = %+v, want %+v", tt.name, got, tt.want)
		}
	}
}

func TestBackgroundOrStatic(t *testing.T) {
	for _, bg := range Backgrounds {
		if got := (Content{Background: bg}).BackgroundOrStatic(); got != bg {
			t.Errorf("BackgroundOrStatic(%q) = %q", bg, got)
		}
	}
	if got := (Content{Background: "disco"}).BackgroundOrStatic(); got != StaticBackground {
		t.Errorf("BackgroundOrStatic(disco) = %q, want %q", got, StaticBackground)
	}
}

func TestServicesUseHTTPS(t *testing.T) {
	for _, s := range Services {
		if s.URL != "https://"+s.Domain {
			t.Errorf("%s: URL %q does not match https://%s", s.Name, s.URL, s.Domain)
		}
	}
}

func TestStoreLoad(t *testing.T) {
	pool := dbtest.New(t)
	ctx := context.Background()
	if err := migrate.Up(ctx, pool, migrations.FS); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	s := NewStore(pool)

	c, err := s.Load(ctx)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if c.Title != "zazi.cool" || c.BannerChoice != "aube" || c.Background != "aurore" || c.Version != 1 || c.Upload != nil {
		t.Errorf("default content = %+v", c)
	}

	var id string
	err = pool.QueryRow(ctx, `INSERT INTO images (content_type, data, width, height)
		VALUES ('image/png', '\x89', 1200, 400) RETURNING id::text`).Scan(&id)
	if err != nil {
		t.Fatalf("insert image: %v", err)
	}
	// The uploaded image is kept when another banner is chosen (F16).
	if _, err := pool.Exec(ctx, `UPDATE content SET upload_image_id = $1, banner_choice = 'aube'`, id); err != nil {
		t.Fatalf("update content: %v", err)
	}
	c, err = s.Load(ctx)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if want := (&Image{ID: id, Width: 1200, Height: 400}); !reflect.DeepEqual(c.Upload, want) {
		t.Errorf("Upload = %+v, want %+v", c.Upload, want)
	}
}

func TestStoreLoadWithoutRow(t *testing.T) {
	pool := dbtest.New(t)
	ctx := context.Background()
	if err := migrate.Up(ctx, pool, migrations.FS); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if _, err := pool.Exec(ctx, `DELETE FROM content`); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, err := NewStore(pool).Load(ctx); err == nil {
		t.Error("Load without content row: no error")
	}
}

func TestEditValidate(t *testing.T) {
	valid := Edit{Title: " Titre ", Body: "a\r\nb", BannerChoice: "none", Background: "aurore"}
	got, errs := valid.Validate(false, false)
	if len(errs) != 0 || got.Title != "Titre" || got.Body != "a\nb" {
		t.Errorf("valid edit: %+v, errors %v", got, errs)
	}
	if _, errs := (Edit{Title: "t", BannerChoice: "upload", Background: "aurore"}).Validate(true, false); len(errs) != 0 {
		t.Errorf("upload with an image: %v", errs)
	}
	_, errs = (Edit{Title: "t", BannerChoice: "upload", Background: "aurore"}).Validate(false, false)
	if errs["banner"] == "" {
		t.Error("upload without an image should be refused")
	}
	if _, errs := (Edit{Title: "t", BannerChoice: "upload", Background: "aurore"}).Validate(false, true); len(errs) != 0 {
		t.Errorf("upload with a file: %v", errs)
	}
	// A file with another choice is refused (F13).
	for _, choice := range []string{"none", "aube"} {
		_, errs := (Edit{Title: "t", BannerChoice: choice, Background: "aurore"}).Validate(true, true)
		if errs["banner"] == "" {
			t.Errorf("file with choice %q should be refused", choice)
		}
	}
	// 80 multi-byte characters are fine, 81 are not: lengths count characters.
	if _, errs := (Edit{Title: strings.Repeat("é", 80), BannerChoice: "none", Background: "aurore"}).Validate(false, false); len(errs) != 0 {
		t.Errorf("80 characters: %v", errs)
	}
	if _, errs := (Edit{Title: "bad\xff", BannerChoice: "none", Background: "aurore"}).Validate(false, false); errs["title"] == "" {
		t.Error("invalid UTF-8 should be refused")
	}
}

func TestBannerKnown(t *testing.T) {
	for choice, want := range map[string]bool{"none": true, "upload": true, "aube": true, "disparue": false, "": false} {
		if got := (Content{BannerChoice: choice}).BannerKnown(); got != want {
			t.Errorf("BannerKnown(%q) = %v, want %v", choice, got, want)
		}
	}
}

func TestGallery(t *testing.T) {
	if len(Gallery) != 6 {
		t.Errorf("%d illustrations, want 6 (D8)", len(Gallery))
	}
	seen := map[string]bool{}
	for _, il := range Gallery {
		if !regexp.MustCompile(`^[a-z]+$`).MatchString(il.ID) || il.ID == BannerNone || il.ID == BannerUpload {
			t.Errorf("illustration id %q is not a safe, unreserved identifier", il.ID)
		}
		if seen[il.ID] {
			t.Errorf("duplicate illustration id %q", il.ID)
		}
		seen[il.ID] = true
		if il.Name == "" || il.Alt == "" {
			t.Errorf("illustration %q needs a name and an alternative text", il.ID)
		}
		if _, errs := (Edit{Title: "t", BannerChoice: il.ID, Background: "aurore"}).Validate(false, false); len(errs) != 0 {
			t.Errorf("choice %q refused: %v", il.ID, errs)
		}
	}
	if _, errs := (Edit{Title: "t", BannerChoice: "disparue", Background: "aurore"}).Validate(false, false); errs["banner"] == "" {
		t.Error("an unknown illustration should be refused")
	}
}
