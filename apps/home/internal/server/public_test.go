package server

import (
	"context"
	"errors"
	"html"
	"io"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
	"unicode/utf8"

	"zazi.cool/home/internal/content"
	"zazi.cool/home/internal/dbtest"
	"zazi.cool/home/internal/migrate"
	"zazi.cool/home/migrations"
)

// fakeContent serves a fixed content, or an error.
type fakeContent struct {
	c   content.Content
	err error
}

func (f fakeContent) Load(context.Context) (content.Content, error) { return f.c, f.err }

func (f fakeContent) Save(context.Context, int, content.Edit) error {
	return errors.New("fake: read-only")
}

func defaultContent() content.Content {
	return content.Content{
		Title:        "zazi.cool",
		Body:         "Bienvenue !",
		BannerChoice: "none",
		Background:   "aurore",
		Version:      1,
	}
}

func publicHandler(t *testing.T, c content.Content) http.Handler {
	t.Helper()
	return newHandler(t, Config{DB: fakeDB{}, Content: fakeContent{c: c}, PublicBaseURL: testBaseURL})
}

// meta returns the content attribute of <meta property="name">, unescaped.
func meta(t *testing.T, body, property string) string {
	t.Helper()
	m := regexp.MustCompile(`<meta property="` + regexp.QuoteMeta(property) + `" content="([^"]*)">`).FindStringSubmatch(body)
	if m == nil {
		t.Fatalf("no <meta property=%q> in:\n%s", property, body)
	}
	return html.UnescapeString(m[1])
}

// CA1, against the content inserted by the migration.
func TestHomeDefaultContent(t *testing.T) {
	pool := dbtest.New(t)
	if err := migrate.Up(context.Background(), pool, migrations.FS); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	h := newHandler(t, Config{DB: pool, Content: content.NewStore(pool), PublicBaseURL: testBaseURL})

	rec := get(t, h, "/")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", rec.Code, rec.Body)
	}
	body := rec.Body.String()
	for _, want := range []string{
		"<title>zazi.cool</title>",
		"<h1>zazi.cool</h1>",
		"<p>Bienvenue ! Ici vivent mes petits services web",
		`data-background="aurore"`,
		`<a class="card" href="https://undercover.zazi.cool">`,
		`<a class="card" href="https://pb.zazi.cool">`,
		`<a class="card" href="https://link.zazi.cool">`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("page lacks %q", want)
		}
	}
	if n := strings.Count(body, `<a class="card"`); n != 3 {
		t.Errorf("%d service cards, want 3", n)
	}
	if !strings.Contains(body, `<ul class="services">`) {
		t.Error("the services are not a list (NF4)")
	}
}

// CA2, without banner.
func TestHomeWithoutBanner(t *testing.T) {
	body := get(t, publicHandler(t, defaultContent()), "/").Body.String()
	if !strings.Contains(body, `<div class="banner banner-none" aria-hidden="true"></div>`) {
		t.Error("no gradient in place of the banner")
	}
	if strings.Contains(body, `<div class="banner"><img`) {
		t.Error("a banner <img> is rendered with the choice « Aucune »")
	}
	if got := meta(t, body, "og:image"); got != testBaseURL+"/og-default.png" {
		t.Errorf("og:image = %q, want the absolute default image", got)
	}
}

// CA2, with an uploaded banner, against the database.
func TestHomeWithUploadedBanner(t *testing.T) {
	pool := dbtest.New(t)
	ctx := context.Background()
	if err := migrate.Up(ctx, pool, migrations.FS); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	var id string
	err := pool.QueryRow(ctx, `INSERT INTO images (content_type, data, width, height)
		VALUES ('image/jpeg', '\xffd8', 1600, 533) RETURNING id::text`).Scan(&id)
	if err != nil {
		t.Fatalf("insert image: %v", err)
	}
	_, err = pool.Exec(ctx, `UPDATE content SET banner_choice = 'upload', upload_image_id = $1,
		banner_alt = 'Un "ciel" <violet>' WHERE id = 1`, id)
	if err != nil {
		t.Fatalf("update content: %v", err)
	}
	h := newHandler(t, Config{DB: pool, Content: content.NewStore(pool), PublicBaseURL: testBaseURL})

	body := get(t, h, "/").Body.String()
	want := `<div class="banner"><img src="/images/` + id +
		`" alt="Un &#34;ciel&#34; &lt;violet&gt;" width="1600" height="533" fetchpriority="high"></div>`
	if !strings.Contains(body, want) {
		t.Errorf("page lacks %s in:\n%s", want, body)
	}
	if strings.Contains(body, "banner-none") {
		t.Error("the gradient is shown along with the banner")
	}
	if got := meta(t, body, "og:image"); got != testBaseURL+"/images/"+id {
		t.Errorf("og:image = %q, want the banner", got)
	}
}

// CA3
func TestHomeBodyIsPlainText(t *testing.T) {
	c := defaultContent()
	c.Body = "<script>alert(1)</script>\n\ndeux"
	rec := get(t, publicHandler(t, c), "/")
	body := rec.Body.String()

	if strings.Contains(body, "<script>alert") {
		t.Error("the script tag of the paragraph is rendered as HTML")
	}
	if !strings.Contains(body, "<p>&lt;script&gt;alert(1)&lt;/script&gt;</p>") {
		t.Error("the script is not shown as text")
	}
	if !strings.Contains(body, "<p>deux</p>") {
		t.Error("no second paragraph")
	}
	if n := strings.Count(body, "<p>"); n != 2 {
		t.Errorf("%d <p>, want 2", n)
	}
	const csp = "default-src 'none'; script-src 'self'; style-src 'self'; img-src 'self'; " +
		"font-src 'self'; form-action 'self'; base-uri 'none'; frame-ancestors 'none'"
	if got := rec.Header().Get("Content-Security-Policy"); got != csp {
		t.Errorf("CSP = %q, want %q", got, csp)
	}
}

func TestHomeLineBreaks(t *testing.T) {
	c := defaultContent()
	c.Body = "un\r\ndeux\n \ntrois"
	body := get(t, publicHandler(t, c), "/").Body.String()
	if !strings.Contains(body, "<p>un<br>deux</p>\n<p>trois</p>") {
		t.Errorf("line breaks not rendered as <br> and <p>:\n%s", body)
	}
}

func TestHomeEmptyBody(t *testing.T) {
	c := defaultContent()
	c.Body = ""
	body := get(t, publicHandler(t, c), "/").Body.String()
	if strings.Contains(body, `class="intro"`) {
		t.Error("an empty paragraph block is rendered")
	}
}

// CA4
func TestHomeOpenGraph(t *testing.T) {
	c := defaultContent()
	c.Title = `Chez "zazi"`
	c.Body = strings.Repeat("Un paragraphe assez long, avec des accents : é à ù.\n", 10)
	body := get(t, publicHandler(t, c), "/").Body.String()

	if got := meta(t, body, "og:title"); got != c.Title {
		t.Errorf("og:title = %q, want %q", got, c.Title)
	}
	desc := meta(t, body, "og:description")
	if n := utf8.RuneCountInString(desc); n == 0 || n > 200 {
		t.Errorf("og:description has %d characters, want 1 to 200: %q", n, desc)
	}
	if got := meta(t, body, "og:url"); got != testBaseURL+"/" {
		t.Errorf("og:url = %q, want %q", got, testBaseURL+"/")
	}
	if got := meta(t, body, "og:image"); got != testBaseURL+"/og-default.png" {
		t.Errorf("og:image = %q", got)
	}
	if got := meta(t, body, "og:type"); got != "website" {
		t.Errorf("og:type = %q, want website", got)
	}
}

func TestHomeUnknownBackgroundIsStatic(t *testing.T) {
	c := defaultContent()
	c.Background = "disco"
	body := get(t, publicHandler(t, c), "/").Body.String()
	if !strings.Contains(body, `data-background="statique"`) {
		t.Error("an unknown background is not replaced by the static one")
	}
}

// CA23 (F22, F23): an animated background loads the bootstrap script (a file of the
// site, never an inline script, NF5); the static one loads no script at all.
func TestHomeBootstrapScript(t *testing.T) {
	inline := regexp.MustCompile(`<script(?:\s[^>]*)?>`)
	external := regexp.MustCompile(`<script type="module" src="(/assets/boot-[^"]+\.js)"></script>`)
	for _, bg := range []string{"aurore", "fils", "grain"} {
		c := defaultContent()
		c.Background = bg
		h := publicHandler(t, c)
		body := get(t, h, "/").Body.String()
		if !strings.Contains(body, `data-background="`+bg+`"`) {
			t.Errorf("%s: no data-background", bg)
		}
		m := external.FindStringSubmatch(body)
		if m == nil {
			t.Fatalf("%s: no bootstrap script (run `npm run build` in web/, or `make test`) in:\n%s", bg, body)
		}
		if n := len(inline.FindAllString(body, -1)); n != 1 {
			t.Errorf("%s: %d script tags, want only the bootstrap script", bg, n)
		}
		rec := get(t, h, m[1])
		if rec.Code != http.StatusOK || rec.Header().Get("Content-Type") != "text/javascript; charset=utf-8" {
			t.Errorf("GET %s: status %d, Content-Type %q", m[1], rec.Code, rec.Header().Get("Content-Type"))
		}
	}

	c := defaultContent()
	c.Background = "statique"
	if body := get(t, publicHandler(t, c), "/").Body.String(); strings.Contains(body, "<script") {
		t.Errorf("the static background loads a script:\n%s", body)
	}
}

func TestHomeDatabaseError(t *testing.T) {
	h := newHandler(t, Config{DB: fakeDB{}, Content: fakeContent{err: errors.New("down")}})
	if rec := get(t, h, "/"); rec.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500", rec.Code)
	}
}

// NF7 and NF9: no cookie for visitors, and the page is never served from a cache.
func TestHomeHeaders(t *testing.T) {
	rec := get(t, publicHandler(t, defaultContent()), "/")
	if got := rec.Header().Values("Set-Cookie"); len(got) != 0 {
		t.Errorf("Set-Cookie = %q, want none", got)
	}
	if got := rec.Header().Get("Cache-Control"); got != "no-cache" {
		t.Errorf("Cache-Control = %q, want no-cache", got)
	}
	if got := rec.Header().Get("X-Content-Type-Options"); got != "nosniff" {
		t.Errorf("X-Content-Type-Options = %q", got)
	}
	if got := rec.Header().Get("Referrer-Policy"); got != "strict-origin-when-cross-origin" {
		t.Errorf("Referrer-Policy = %q", got)
	}
}

// CA5
func TestNotFound(t *testing.T) {
	h := publicHandler(t, defaultContent())
	for _, path := range []string{"/inconnu", "/assets/home.css", "/assets/nope.0123456789.css", "/index.html"} {
		rec := get(t, h, path)
		if rec.Code != http.StatusNotFound {
			t.Errorf("GET %s: status = %d, want 404", path, rec.Code)
		}
		if !strings.Contains(rec.Body.String(), "<h1>Page introuvable</h1>") ||
			!strings.Contains(rec.Body.String(), `<a class="back" href="/">`) {
			t.Errorf("GET %s: not the « Page introuvable » page", path)
		}
	}
}

// assetURLs returns the stylesheet and font URLs of a page.
func assetURLs(t *testing.T, body string) (css, font string) {
	t.Helper()
	cssMatch := regexp.MustCompile(`<link rel="stylesheet" href="(/assets/home\.[0-9a-f]{10}\.css)">`).FindStringSubmatch(body)
	fontMatch := regexp.MustCompile(`<link rel="preload" href="(/assets/geist\.[0-9a-f]{10}\.woff2)"`).FindStringSubmatch(body)
	if cssMatch == nil || fontMatch == nil {
		t.Fatalf("no fingerprinted stylesheet or font in:\n%s", body)
	}
	return cssMatch[1], fontMatch[1]
}

func TestAssetsAreFingerprintedAndCached(t *testing.T) {
	h := publicHandler(t, defaultContent())
	css, font := assetURLs(t, get(t, h, "/").Body.String())

	rec := get(t, h, css)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET %s: status = %d", css, rec.Code)
	}
	if got := rec.Header().Get("Content-Type"); got != "text/css; charset=utf-8" {
		t.Errorf("Content-Type = %q", got)
	}
	if got := rec.Header().Get("Cache-Control"); got != "public, max-age=31536000, immutable" {
		t.Errorf("Cache-Control = %q", got)
	}
	// The stylesheet points to the fingerprinted font and grain.
	if !strings.Contains(rec.Body.String(), `url("`+strings.TrimPrefix(font, "/assets/")+`")`) {
		t.Error("the stylesheet does not reference the fingerprinted font")
	}
	if regexp.MustCompile(`url\("(geist\.woff2|grain\.svg)"\)`).MatchString(rec.Body.String()) {
		t.Error("the stylesheet still references an asset by its plain name")
	}

	rec = get(t, h, font)
	if rec.Code != http.StatusOK || rec.Header().Get("Content-Type") != "font/woff2" {
		t.Errorf("GET %s: status %d, Content-Type %q", font, rec.Code, rec.Header().Get("Content-Type"))
	}
}

func TestRootFiles(t *testing.T) {
	h := publicHandler(t, defaultContent())
	tests := []struct {
		path, contentType, contains string
	}{
		{"/robots.txt", "text/plain; charset=utf-8", "Disallow: /admin"},
		{"/favicon.svg", "image/svg+xml", "<svg"},
		{"/og-default.png", "image/png", "\x89PNG"},
	}
	for _, tt := range tests {
		rec := get(t, h, tt.path)
		if rec.Code != http.StatusOK {
			t.Errorf("GET %s: status = %d", tt.path, rec.Code)
			continue
		}
		if got := rec.Header().Get("Content-Type"); got != tt.contentType {
			t.Errorf("GET %s: Content-Type = %q, want %q", tt.path, got, tt.contentType)
		}
		if !strings.Contains(rec.Body.String(), tt.contains) {
			t.Errorf("GET %s: body lacks %q", tt.path, tt.contains)
		}
	}
	if got := get(t, h, "/favicon.svg").Header().Get("Content-Security-Policy"); got != svgCSP {
		t.Errorf("SVG CSP = %q, want %q", got, svgCSP)
	}
}

// CA18 (NF3): HTML + CSS + font of the page weigh 80 KiB at most, uncompressed.
func TestPageWeight(t *testing.T) {
	c := defaultContent()
	c.Body = strings.Repeat("x", 600) // the longest paragraph allowed (F13)
	h := publicHandler(t, c)
	page := get(t, h, "/").Body.String()
	css, font := assetURLs(t, page)

	total := len(page) + get(t, h, css).Body.Len() + get(t, h, font).Body.Len()
	const budget = 80 << 10
	if total > budget {
		t.Errorf("HTML + CSS + font = %d bytes, budget %d", total, budget)
	}
	t.Logf("HTML + CSS + font = %d bytes (budget %d)", total, budget)
}

// NF8
func TestBodyLimits(t *testing.T) {
	read := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, err := io.ReadAll(r.Body); err != nil {
			var tooLarge *http.MaxBytesError
			if errors.As(err, &tooLarge) {
				w.WriteHeader(http.StatusRequestEntityTooLarge)
				return
			}
			t.Errorf("read body: %v", err)
		}
	})
	h := limitBody(1<<20, read)
	tests := []struct {
		path string
		size int
		want int
	}{
		{"/anything", 16 << 10, http.StatusOK},
		{"/anything", 16<<10 + 1, http.StatusRequestEntityTooLarge},
		{"/admin", 1<<20 + 64<<10, http.StatusOK},
		{"/admin", 1<<20 + 64<<10 + 1, http.StatusRequestEntityTooLarge},
	}
	for _, tt := range tests {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest("POST", tt.path, strings.NewReader(strings.Repeat("a", tt.size))))
		if rec.Code != tt.want {
			t.Errorf("POST %s with %d bytes: status = %d, want %d", tt.path, tt.size, rec.Code, tt.want)
		}
	}
}
