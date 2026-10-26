package content

import (
	"context"
	"reflect"
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
	tests := []struct {
		name string
		c    Content
		want *Banner
	}{
		{"none", Content{BannerChoice: "none", Upload: upload}, nil},
		{"upload", Content{BannerChoice: "upload", BannerAlt: "Ciel", Upload: upload},
			&Banner{URL: "/images/0193", Alt: "Ciel", Width: 1600, Height: 533}},
		{"upload without image", Content{BannerChoice: "upload"}, nil},
		{"unknown illustration", Content{BannerChoice: "aube", Upload: upload}, nil},
	}
	for _, tt := range tests {
		if got := tt.c.Banner(); !reflect.DeepEqual(got, tt.want) {
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
	if c.Title != "zazi.cool" || c.BannerChoice != "none" || c.Background != "aurore" || c.Version != 1 || c.Upload != nil {
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
