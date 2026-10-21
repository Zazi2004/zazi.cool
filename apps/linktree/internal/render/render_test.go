package render

import (
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"
	"testing/fstest"
)

const manifest = `{
  "src/public/main.tsx": {
    "file": "assets/public-a1.js", "isEntry": true,
    "imports": ["_shared-c3.js"], "css": ["assets/public-a1.css"]
  },
  "src/admin/main.tsx": {
    "file": "assets/admin-b2.js", "isEntry": true, "imports": ["_shared-c3.js"]
  },
  "_shared-c3.js": { "file": "assets/shared-c3.js", "css": ["assets/shared-c3.css"] },
  "src/backgrounds/aurora.ts": { "file": "assets/aurora-d4.js", "isDynamicEntry": true }
}`

func testAssets(t *testing.T) *Assets {
	t.Helper()
	a, err := FromManifest(fstest.MapFS{".vite/manifest.json": {Data: []byte(manifest)}})
	if err != nil {
		t.Fatal(err)
	}
	return a
}

func TestFromManifest(t *testing.T) {
	e, err := testAssets(t).Entry(publicEntry)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(e.Scripts, []string{"/assets/public-a1.js"}) {
		t.Errorf("scripts = %v", e.Scripts)
	}
	// CSS of imported chunks is needed too, or shared styles would be missing.
	if !slices.Equal(e.Styles, []string{"/assets/public-a1.css", "/assets/shared-c3.css"}) {
		t.Errorf("styles = %v", e.Styles)
	}
	if !slices.Equal(e.Preloads, []string{"/assets/shared-c3.js"}) {
		t.Errorf("preloads = %v", e.Preloads)
	}
	if _, err := testAssets(t).Entry("src/unknown.tsx"); err == nil {
		t.Error("unknown entry should fail")
	}
}

func TestFromManifestMissing(t *testing.T) {
	if _, err := FromManifest(fstest.MapFS{}); err == nil {
		t.Error("a missing manifest should fail")
	}
}

func TestDevServer(t *testing.T) {
	e, err := FromDevServer("http://localhost:5173/").Entry(adminEntry)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"http://localhost:5173/@vite/client", "http://localhost:5173/src/admin/main.tsx"}
	if !slices.Equal(e.Scripts, want) {
		t.Errorf("scripts = %v, want %v", e.Scripts, want)
	}
}

func renderPublic(t *testing.T, status int, s Shell) (*httptest.ResponseRecorder, string) {
	t.Helper()
	r, err := New(testAssets(t))
	if err != nil {
		t.Fatal(err)
	}
	rec := httptest.NewRecorder()
	if err := r.Public(rec, status, s); err != nil {
		t.Fatal(err)
	}
	return rec, rec.Body.String()
}

func TestPublicEscapesUserContent(t *testing.T) {
	evil := `</script><script>alert(1)</script>"`
	_, body := renderPublic(t, http.StatusOK, Shell{
		Title:       evil,
		Description: evil,
		Data:        map[string]string{"title": evil},
	})
	if strings.Contains(body, "<script>alert(1)") {
		t.Fatalf("user content was not escaped:\n%s", body)
	}
	// The JSON must still be there, with "<" escaped so the browser cannot end the tag.
	if !strings.Contains(body, `</script>`) {
		t.Errorf("page data missing or not JSON-escaped:\n%s", body)
	}
}

func TestPublicMetaAndStatus(t *testing.T) {
	rec, body := renderPublic(t, http.StatusNotFound, Shell{
		Title:   "Musique",
		URL:     "https://link.zazi.cool/musique",
		Image:   "https://link.zazi.cool/og-default.png",
		NoIndex: true,
		Theme:   "nuit",
		Accent:  "violet",
	})
	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d", rec.Code)
	}
	for _, want := range []string{
		`<html lang="fr" data-theme="nuit" data-accent="violet">`,
		`<meta name="robots" content="noindex">`,
		`<meta property="og:url" content="https://link.zazi.cool/musique">`,
		`<meta property="og:image" content="https://link.zazi.cool/og-default.png">`,
		`<link rel="stylesheet" href="/assets/public-a1.css">`,
		`<script type="module" src="/assets/public-a1.js"></script>`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("missing %s in:\n%s", want, body)
		}
	}
	if strings.Contains(body, "page-data") {
		t.Error("no page data was given: the script tag should be absent")
	}
}

func TestFallback(t *testing.T) {
	_, body := renderPublic(t, http.StatusOK, Shell{
		Title: "Musique",
		Fallback: &Fallback{
			Heading: "Musique",
			Items: []FallbackItem{
				{Title: "Appeler", URL: "tel:+33600000000"},
				{Title: "Titre 1", Links: []FallbackItem{{Title: "Spotify", URL: "https://open.spotify.com/track/1"}}},
			},
		},
	})
	for _, want := range []string{
		`<a href="tel:&#43;33600000000">Appeler</a>`,
		`<summary>Titre 1</summary>`,
		`<a href="https://open.spotify.com/track/1">Spotify</a>`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("missing %s in:\n%s", want, body)
		}
	}
}

func TestAdmin(t *testing.T) {
	r, err := New(testAssets(t))
	if err != nil {
		t.Fatal(err)
	}
	rec := httptest.NewRecorder()
	if err := r.Admin(rec, "n0nce"); err != nil {
		t.Fatal(err)
	}
	body := rec.Body.String()
	for _, want := range []string{`src="/assets/admin-b2.js"`, `content="noindex"`, `<meta name="csp-nonce" content="n0nce">`} {
		if !strings.Contains(body, want) {
			t.Errorf("missing %s in admin shell:\n%s", want, body)
		}
	}
}
