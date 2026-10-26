package web

import (
	"strings"
	"testing"
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
