package web

import (
	"bytes"
	"image/jpeg"
	"regexp"
	"strings"
	"testing"

	"zazi.cool/home/internal/content"
)

func TestLoadAssets(t *testing.T) {
	a, err := LoadAssets()
	if err != nil {
		t.Fatal(err)
	}
	css, err := a.URL("home.css")
	if err != nil {
		t.Fatal(err)
	}
	font, err := a.URL("geist.woff2")
	if err != nil {
		t.Fatal(err)
	}
	f, ok := a.Asset(strings.TrimPrefix(css, "/assets/"))
	if !ok {
		t.Fatalf("%s is not served", css)
	}
	if !strings.Contains(string(f.Data), `url("`+strings.TrimPrefix(font, "/assets/")+`")`) {
		t.Error("the CSS does not point to the fingerprinted font")
	}
	if _, err := a.URL("missing.css"); err == nil {
		t.Error("URL of an unknown asset: no error")
	}
	for _, name := range []string{"favicon.svg", "og-default.png", "robots.txt"} {
		if _, ok := a.Root(name); !ok {
			t.Errorf("root file %s is missing", name)
		}
	}
}

// Every template parses and every {{asset}} it uses exists: rendering is covered
// by the server tests.
func TestTemplates(t *testing.T) {
	a, err := LoadAssets()
	if err != nil {
		t.Fatal(err)
	}
	tmpl, err := Templates(a)
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"home.html", "notfound.html"} {
		if tmpl.Lookup(name) == nil {
			t.Errorf("template %s is not defined", name)
		}
	}
}

// F16b: every illustration of the gallery has its picture, in the format of D6, and
// no picture is left out of the gallery.
func TestGalleryPictures(t *testing.T) {
	a, err := LoadAssets()
	if err != nil {
		t.Fatal(err)
	}
	for _, il := range content.Gallery {
		url, ok := a.IllustrationURL(il.ID)
		if !ok {
			t.Errorf("%s: no picture", il.ID)
			continue
		}
		if !strings.HasPrefix(url, "/assets/banners/"+il.ID+".") || !strings.HasSuffix(url, ".jpg") {
			t.Errorf("%s: URL %q", il.ID, url)
		}
		f, ok := a.Asset(strings.TrimPrefix(url, "/assets/"))
		if !ok || f.ContentType != "image/jpeg" {
			t.Errorf("%s: %q is not served as a JPEG", il.ID, url)
			continue
		}
		if len(f.Data) > 250<<10 {
			t.Errorf("%s: %d bytes, over 250 Kio", il.ID, len(f.Data))
		}
		cfg, err := jpeg.DecodeConfig(bytes.NewReader(f.Data))
		if err != nil {
			t.Errorf("%s: %v", il.ID, err)
			continue
		}
		if cfg.Width != content.IllustrationWidth || cfg.Height != content.IllustrationHeight {
			t.Errorf("%s: %d × %d, want %d × %d", il.ID, cfg.Width, cfg.Height, content.IllustrationWidth, content.IllustrationHeight)
		}
	}
	banners := 0
	for name := range a.urls {
		if strings.HasPrefix(name, "banners/") {
			banners++
		}
	}
	if banners != len(content.Gallery) {
		t.Errorf("%d pictures for %d illustrations", banners, len(content.Gallery))
	}
}

// The bootstrap script comes from the Vite build (web/dist): `make test` builds it first.
func TestLoadAssetsServesTheFront(t *testing.T) {
	a, err := LoadAssets()
	if err != nil {
		t.Fatal(err)
	}
	script := a.Script()
	if script == "" {
		t.Fatal("no bootstrap script: run `npm run build` in web/ (or `make test`)")
	}
	f, ok := a.Asset(strings.TrimPrefix(script, "/assets/"))
	if !ok {
		t.Fatalf("%s is not served", script)
	}
	if f.ContentType != "text/javascript; charset=utf-8" {
		t.Errorf("Content-Type = %q", f.ContentType)
	}
	// The script loads the modules by relative URL: each must be served under its own name.
	for _, module := range regexp.MustCompile(`import\("\./([^"]+\.js)"\)`).FindAllStringSubmatch(string(f.Data), -1) {
		if _, ok := a.Asset(module[1]); !ok {
			t.Errorf("module %s, loaded by the bootstrap script, is not served", module[1])
		}
	}
}
