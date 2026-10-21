package pages

import (
	"fmt"
	"strings"
	"testing"
	"testing/fstest"
)

const (
	id1 = "0199e6a8-0000-7000-8000-000000000001"
	id2 = "0199e6a8-0000-7000-8000-000000000002"
	id3 = "0199e6a8-0000-7000-8000-000000000003"
)

var testShapes = Shapes{"star-1": true, "flower-3": true}

func validPage() Page {
	return Page{
		Slug:   "musique",
		Title:  "Musique",
		Theme:  DefaultTheme,
		Accent: DefaultAccent,
		Avatar: Visual{Shape: "star-1"},
		Items: []Item{
			{ID: id1, Kind: KindLink, Title: "Ma playlist", URL: "https://open.spotify.com/playlist/1", Published: true},
			{ID: id2, Kind: KindGroup, Title: "Titre 1", Subtitle: "Artiste", Visual: Visual{Shape: "flower-3"}, Published: true,
				Links: []Item{{ID: id3, Kind: KindLink, Title: "Deezer", URL: "https://www.deezer.com/track/1", Published: true}}},
		},
	}
}

func TestValidateAcceptsValidPage(t *testing.T) {
	if errs := Validate(validPage(), testShapes); errs != nil {
		t.Fatalf("unexpected errors: %v", errs)
	}
}

func TestValidateSlug(t *testing.T) {
	for _, slug := range []string{"musique", "jeux-video", "a", "2026", strings.Repeat("a", 40)} {
		if msg := ValidateSlug(slug); msg != "" {
			t.Errorf("ValidateSlug(%q) = %q, want valid", slug, msg)
		}
	}
	for _, slug := range []string{"", "Musique", "jeux vidéo", "-jeux", "jeux-", "jeux--video", "é", "a.b",
		strings.Repeat("a", 41), "admin", "shapes", "healthz", "version"} {
		if msg := ValidateSlug(slug); msg == "" {
			t.Errorf("ValidateSlug(%q) should fail", slug)
		}
	}
}

func TestValidateURL(t *testing.T) {
	for _, u := range []string{
		"https://open.spotify.com/track/1?si=abc", "http://example.com", "mailto:moi@zazi.cool",
		"tel:+33612345678", "tel:+33-6-12-34-56-78", "https://exemple.fr/chemin#ancre",
	} {
		if msg := ValidateURL(u); msg != "" {
			t.Errorf("ValidateURL(%q) = %q, want valid", u, msg)
		}
	}
	for _, u := range []string{
		"", "javascript:alert(1)", "JavaScript:alert(1)", "java\tscript:alert(1)", "data:text/html,<script>",
		"vbscript:x", "/musique", "exemple.fr", "https://", "https:exemple.fr", "https://user:pw@exemple.fr",
		"mailto:", "tel:abc", "tel:+33 6 12", " https://exemple.fr", "https://exemple.fr/a b", "ftp://exemple.fr",
		"https://exemple.fr/" + strings.Repeat("a", MaxURL),
	} {
		if msg := ValidateURL(u); msg == "" {
			t.Errorf("ValidateURL(%q) should fail", u)
		}
	}
}

func TestValidateRejects(t *testing.T) {
	tests := []struct {
		name   string
		change func(p *Page)
		field  string
	}{
		{"empty title", func(p *Page) { p.Title = "" }, "title"},
		{"long title", func(p *Page) { p.Title = strings.Repeat("é", MaxTitle+1) }, "title"},
		{"control char", func(p *Page) { p.Title = "a\u0000b" }, "title"},
		{"long description", func(p *Page) { p.Description = strings.Repeat("a", MaxDescription+1) }, "description"},
		{"unknown theme", func(p *Page) { p.Theme = "rose" }, "theme"},
		{"unknown accent", func(p *Page) { p.Accent = "fuchsia" }, "accent"},
		{"unknown shape", func(p *Page) { p.Avatar.Shape = "star-99" }, "avatar"},
		{"image and shape", func(p *Page) { p.Avatar.ImageID = id1 }, "avatar"},
		{"bad image id", func(p *Page) { p.Avatar = Visual{ImageID: "x"} }, "avatar"},
		{"bad url", func(p *Page) { p.Items[0].URL = "javascript:alert(1)" }, "items." + id1 + ".url"},
		{"shape on link", func(p *Page) { p.Items[0].Visual.Shape = "star-1" }, "items." + id1 + ".visual"},
		{"url on group", func(p *Page) { p.Items[1].URL = "https://exemple.fr" }, "items." + id2 + ".url"},
		{"nested group", func(p *Page) { p.Items[1].Links[0].Kind = KindGroup }, "items." + id3 + ".kind"},
		{"unknown kind", func(p *Page) { p.Items[0].Kind = "image" }, "items." + id1 + ".kind"},
		{"bad item id", func(p *Page) { p.Items[0].ID = "1" }, "items"},
		{"duplicate id", func(p *Page) { p.Items[1].Links[0].ID = id1 }, "items"},
		{"too many group links", func(p *Page) {
			p.Items[1].Links = manyLinks(MaxGroupLinks + 1)
		}, "items." + id2 + ".links"},
		{"too many top items", func(p *Page) { p.Items = manyLinks(MaxTopItems + 1) }, "items"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := validPage()
			tt.change(&p)
			errs := Validate(p, testShapes)
			if _, ok := errs[tt.field]; !ok {
				t.Errorf("want an error on %q, got %v", tt.field, errs)
			}
		})
	}
}

func TestValidateTotalItems(t *testing.T) {
	p := validPage()
	p.Items = nil
	// 20 groups of 25 links: under the per-level limits, over the total.
	for g := range 20 {
		group := Item{ID: uuid(10000 + g), Kind: KindGroup, Title: "G", Links: manyLinks(25)}
		for i := range group.Links {
			group.Links[i].ID = uuid(g*100 + i)
		}
		p.Items = append(p.Items, group)
	}
	if _, ok := Validate(p, testShapes)["items"]; !ok {
		t.Error("more than 500 items in total should be refused")
	}
}

func manyLinks(n int) []Item {
	links := make([]Item, n)
	for i := range links {
		links[i] = Item{ID: uuid(50000 + i), Kind: KindLink, Title: "L", URL: "https://exemple.fr"}
	}
	return links
}

func uuid(n int) string {
	return fmt.Sprintf("0199e6a8-0000-7000-8000-%012d", n)
}

func TestNormalize(t *testing.T) {
	p := validPage()
	p.Title = "  Musique "
	p.Items[1].Links[0].URL = " https://www.deezer.com/track/1\n"
	Normalize(&p)
	if p.Title != "Musique" || p.Items[1].Links[0].URL != "https://www.deezer.com/track/1" {
		t.Errorf("not trimmed: %q, %q", p.Title, p.Items[1].Links[0].URL)
	}
}

func TestVisibleHidesDrafts(t *testing.T) {
	p := validPage()
	p.Items = append(p.Items,
		Item{ID: uuid(1), Kind: KindLink, Title: "Brouillon", URL: "https://exemple.fr", Published: false},
		// A published group whose links are all drafts is hidden too.
		Item{ID: uuid(2), Kind: KindGroup, Title: "Vide", Published: true,
			Links: []Item{{ID: uuid(3), Kind: KindLink, Title: "B", URL: "https://exemple.fr"}}},
	)
	p.Items[1].Links = append(p.Items[1].Links, Item{ID: uuid(4), Kind: KindLink, Title: "B", URL: "https://exemple.fr"})

	v := p.Visible()
	if len(v.Items) != 2 {
		t.Fatalf("visible items = %d, want 2: %+v", len(v.Items), v.Items)
	}
	if len(v.Items[1].Links) != 1 {
		t.Errorf("draft link inside a group should be hidden: %+v", v.Items[1].Links)
	}
	if len(p.Items) != 4 || len(p.Items[1].Links) != 2 {
		t.Error("Visible must not modify the original page")
	}
}

func TestLoadShapes(t *testing.T) {
	shapes, err := LoadShapes(fstest.MapFS{
		"shapes/star-1.svg":   {},
		"shapes/moon-12.svg":  {},
		"shapes/LICENSE":      {},
		"assets/other.svg":    {},
		"shapes/sub/deep.svg": {},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(shapes) != 2 || !shapes.Has("star-1") || !shapes.Has("moon-12") {
		t.Errorf("shapes = %v", shapes)
	}
}
