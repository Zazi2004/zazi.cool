package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
	"testing/fstest"

	"golang.org/x/crypto/bcrypt"

	"zazi.cool/linktree/internal/auth"
	"zazi.cool/linktree/internal/render"
	"zazi.cool/linktree/internal/store/storetest"
)

const password = "correct horse battery"

const testManifest = `{
  "src/public/main.tsx": { "file": "assets/public.js", "isEntry": true },
  "src/admin/main.tsx": { "file": "assets/admin.js", "isEntry": true }
}`

// testServer is the whole application on a temporary database.
type testServer struct {
	t       *testing.T
	handler http.Handler
	cookie  *http.Cookie // session cookie once logged in
}

func newTestServer(t *testing.T) *testServer {
	t.Helper()
	st := storetest.New(t)
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	am, err := auth.NewManager(string(hash), st, true)
	if err != nil {
		t.Fatal(err)
	}
	assets, err := render.FromManifest(fstest.MapFS{".vite/manifest.json": {Data: []byte(testManifest)}})
	if err != nil {
		t.Fatal(err)
	}
	renderer, err := render.New(assets)
	if err != nil {
		t.Fatal(err)
	}
	static := fstest.MapFS{
		"shapes/star-1.svg": {Data: []byte(`<svg xmlns="http://www.w3.org/2000/svg"/>`)},
		"assets/public.js":  {Data: []byte("console.log(1)")},
	}
	return &testServer{t: t, handler: NewHandler(Config{
		Store:          st,
		Auth:           am,
		Renderer:       renderer,
		Static:         static,
		Shapes:         map[string]bool{"star-1": true},
		PublicBaseURL:  "https://link.zazi.cool",
		LoginLimit:     5,
		MaxUploadBytes: 1 << 20,
		Version:        "abc1234",
	})}
}

// do sends a request; body is JSON-encoded unless it is already an io.Reader.
func (s *testServer) do(method, path string, body any, headers ...string) *httptest.ResponseRecorder {
	s.t.Helper()
	var r io.Reader
	switch b := body.(type) {
	case nil:
	case io.Reader:
		r = b
	default:
		data, err := json.Marshal(b)
		if err != nil {
			s.t.Fatal(err)
		}
		r = bytes.NewReader(data)
	}
	req := httptest.NewRequest(method, path, r)
	for i := 0; i+1 < len(headers); i += 2 {
		req.Header.Set(headers[i], headers[i+1])
	}
	if s.cookie != nil {
		req.AddCookie(s.cookie)
	}
	rec := httptest.NewRecorder()
	s.handler.ServeHTTP(rec, req)
	return rec
}

func (s *testServer) login() {
	s.t.Helper()
	rec := s.do("POST", "/admin/api/login", LoginRequest{Password: password})
	if rec.Code != http.StatusOK {
		s.t.Fatalf("login: %d %s", rec.Code, rec.Body)
	}
	s.cookie = rec.Result().Cookies()[0]
}

func decode[T any](t *testing.T, rec *httptest.ResponseRecorder) T {
	t.Helper()
	var v T
	if err := json.Unmarshal(rec.Body.Bytes(), &v); err != nil {
		t.Fatalf("decode %T from %q: %v", v, rec.Body, err)
	}
	return v
}

func expectStatus(t *testing.T, rec *httptest.ResponseRecorder, want int) {
	t.Helper()
	if rec.Code != want {
		t.Fatalf("status = %d, want %d: %s", rec.Code, want, rec.Body)
	}
}

func uuid(n int) string {
	return fmt.Sprintf("0199e6a8-0000-7000-8000-%012d", n)
}

func TestHealthAndVersion(t *testing.T) {
	s := newTestServer(t)
	expectStatus(t, s.do("GET", "/healthz", nil), http.StatusOK)
	expectStatus(t, s.do("GET", "/readyz", nil), http.StatusOK)
	rec := s.do("GET", "/version", nil)
	expectStatus(t, rec, http.StatusOK)
	if strings.TrimSpace(rec.Body.String()) != "abc1234" {
		t.Errorf("version = %q", rec.Body)
	}
}

func TestAdminAPIRequiresSession(t *testing.T) {
	s := newTestServer(t)
	for _, route := range []struct{ method, path string }{
		{"GET", "/admin/api/session"},
		{"GET", "/admin/api/pages"},
		{"POST", "/admin/api/pages"},
		{"GET", "/admin/api/pages/" + uuid(1)},
		{"PUT", "/admin/api/pages/" + uuid(1)},
		{"DELETE", "/admin/api/pages/" + uuid(1)},
		{"PUT", "/admin/api/pages/order"},
		{"GET", "/admin/api/slugs/musique"},
		{"POST", "/admin/api/images"},
		{"POST", "/admin/api/logout"},
	} {
		rec := s.do(route.method, route.path, nil)
		if rec.Code != http.StatusUnauthorized {
			t.Errorf("%s %s: status %d, want 401", route.method, route.path, rec.Code)
		}
	}
}

func TestLogin(t *testing.T) {
	s := newTestServer(t)
	rec := s.do("POST", "/admin/api/login", LoginRequest{Password: "nope"})
	expectStatus(t, rec, http.StatusUnauthorized)
	if body := decode[ErrorResponse](t, rec); body.Error.Code != CodeBadPassword {
		t.Errorf("code = %s", body.Error.Code)
	}

	s.login()
	expectStatus(t, s.do("GET", "/admin/api/session", nil), http.StatusOK)

	expectStatus(t, s.do("POST", "/admin/api/logout", nil), http.StatusNoContent)
	expectStatus(t, s.do("GET", "/admin/api/session", nil), http.StatusUnauthorized)
}

func TestLoginRateLimit(t *testing.T) {
	s := newTestServer(t)
	for range 5 {
		s.do("POST", "/admin/api/login", LoginRequest{Password: "nope"})
	}
	// Even the right password is refused once the limit is reached.
	expectStatus(t, s.do("POST", "/admin/api/login", LoginRequest{Password: password}), http.StatusTooManyRequests)
}

func TestCrossOriginRequestsAreRefused(t *testing.T) {
	s := newTestServer(t)
	s.login()
	rec := s.do("POST", "/admin/api/pages", CreatePageRequest{Title: "Musique", Slug: "musique"},
		"Origin", "https://evil.example", "Sec-Fetch-Site", "cross-site")
	expectStatus(t, rec, http.StatusForbidden)
	// The same request from the admin itself goes through.
	rec = s.do("POST", "/admin/api/pages", CreatePageRequest{Title: "Musique", Slug: "musique"},
		"Sec-Fetch-Site", "same-origin")
	expectStatus(t, rec, http.StatusCreated)
}

// createPage creates a page through the API and returns its document.
func (s *testServer) createPage(title, slug string) Document {
	s.t.Helper()
	rec := s.do("POST", "/admin/api/pages", CreatePageRequest{Title: title, Slug: slug})
	expectStatus(s.t, rec, http.StatusCreated)
	return decode[Document](s.t, rec)
}

func (s *testServer) save(doc Document) *httptest.ResponseRecorder {
	s.t.Helper()
	return s.do("PUT", "/admin/api/pages/"+doc.ID, doc)
}

func link(n int, title string, published bool) DocItem {
	return DocItem{ID: uuid(n), Kind: "link", Title: title, URL: "https://exemple.fr/" + title, Published: published, Links: []DocItem{}}
}

func TestPublicPageNeverShowsDrafts(t *testing.T) {
	s := newTestServer(t)
	s.login()
	doc := s.createPage("Musique", "musique")
	if doc.Avatar == nil || doc.Avatar.Shape != "star-1" || doc.Published {
		t.Fatalf("new page: %+v", doc)
	}

	// A draft page is not public.
	expectStatus(t, s.do("GET", "/musique", nil), http.StatusNotFound)

	doc.Published = true
	doc.Items = []DocItem{
		link(1, "visible", true),
		link(2, "brouillon-lien", false),
		{ID: uuid(3), Kind: "group", Title: "Titre 1", Published: true, Links: []DocItem{
			link(4, "spotify", true), link(5, "brouillon-dans-groupe", false),
		}},
		{ID: uuid(6), Kind: "group", Title: "brouillon-groupe", Published: false, Links: []DocItem{link(7, "x", true)}},
	}
	rec := s.save(doc)
	expectStatus(t, rec, http.StatusOK)
	if v := decode[SaveResponse](t, rec).Version; v != doc.Version+1 {
		t.Errorf("version = %d", v)
	}

	rec = s.do("GET", "/musique", nil)
	expectStatus(t, rec, http.StatusOK)
	body := rec.Body.String()
	for _, want := range []string{"visible", "spotify", "Titre 1", `data-theme="zazi"`,
		`<meta property="og:url" content="https://link.zazi.cool/musique">`} {
		if !strings.Contains(body, want) {
			t.Errorf("page should contain %q", want)
		}
	}
	for _, draft := range []string{"brouillon-lien", "brouillon-dans-groupe", "brouillon-groupe"} {
		if strings.Contains(body, draft) {
			t.Errorf("draft %q leaked in the public page", draft)
		}
	}
	if strings.Contains(body, "noindex") {
		t.Error("a listed page must be indexable")
	}
	if rec.Header().Get("Content-Security-Policy") != productionCSP {
		t.Error("public pages need the CSP")
	}
}

func TestIndexAndUnlistedPages(t *testing.T) {
	s := newTestServer(t)
	s.login()
	listed := s.createPage("Musique", "musique")
	unlisted := s.createPage("Secret", "secret")
	s.createPage("Brouillon", "brouillon")
	listed.Published = true
	unlisted.Published, unlisted.Listed = true, false
	expectStatus(t, s.save(listed), http.StatusOK)
	expectStatus(t, s.save(unlisted), http.StatusOK)

	rec := s.do("GET", "/", nil)
	expectStatus(t, rec, http.StatusOK)
	body := rec.Body.String()
	if !strings.Contains(body, "Musique") || strings.Contains(body, "Secret") || strings.Contains(body, "Brouillon") {
		t.Errorf("index should only list published, listed pages:\n%s", body)
	}

	rec = s.do("GET", "/secret", nil)
	expectStatus(t, rec, http.StatusOK)
	if !strings.Contains(rec.Body.String(), `<meta name="robots" content="noindex">`) {
		t.Error("an unlisted page must not be indexed")
	}
}

func TestNotFound(t *testing.T) {
	s := newTestServer(t)
	for _, path := range []string{"/inconnue", "/Majuscule", "/admin-x", "/images/" + uuid(1), "/images/pas-un-uuid"} {
		if rec := s.do("GET", path, nil); rec.Code != http.StatusNotFound {
			t.Errorf("GET %s: status %d, want 404", path, rec.Code)
		}
	}
	s.login()
	rec := s.do("GET", "/admin/api/inconnue", nil)
	expectStatus(t, rec, http.StatusNotFound)
	if decode[ErrorResponse](t, rec).Error.Code != CodeNotFound {
		t.Error("unknown API routes answer JSON")
	}
}

func TestSaveValidation(t *testing.T) {
	s := newTestServer(t)
	s.login()
	doc := s.createPage("Musique", "musique")

	bad := link(1, "x", true)
	bad.URL = "javascript:alert(1)"
	doc.Items = []DocItem{bad}
	doc.Theme = "inconnu"
	rec := s.save(doc)
	expectStatus(t, rec, http.StatusUnprocessableEntity)
	fields := decode[ErrorResponse](t, rec).Error.Fields
	if fields["items."+uuid(1)+".url"] == "" || fields["theme"] == "" {
		t.Errorf("fields = %v", fields)
	}

	// Stale version: the editor of another tab saved first.
	doc.Items, doc.Theme = nil, "zazi"
	expectStatus(t, s.save(doc), http.StatusOK)
	expectStatus(t, s.save(doc), http.StatusConflict)

	// Slug already used by another page.
	other := s.createPage("Jeux", "jeux")
	other.Slug = "musique"
	rec = s.save(other)
	expectStatus(t, rec, http.StatusUnprocessableEntity)
	if decode[ErrorResponse](t, rec).Error.Fields["slug"] == "" {
		t.Error("slug conflict should be reported on the slug field")
	}

	// The document must match the URL.
	expectStatus(t, s.do("PUT", "/admin/api/pages/"+uuid(99), other), http.StatusBadRequest)
	// Unknown fields mean a client out of sync.
	expectStatus(t, s.do("PUT", "/admin/api/pages/"+other.ID, strings.NewReader(`{"id":"`+other.ID+`","oops":1}`)),
		http.StatusBadRequest)
}

func TestCreatePageValidation(t *testing.T) {
	s := newTestServer(t)
	s.login()
	rec := s.do("POST", "/admin/api/pages", CreatePageRequest{Title: "", Slug: "Admin"})
	expectStatus(t, rec, http.StatusUnprocessableEntity)
	if f := decode[ErrorResponse](t, rec).Error.Fields; f["title"] == "" || f["slug"] == "" {
		t.Errorf("fields = %v", f)
	}
	s.createPage("Musique", "musique")
	expectStatus(t, s.do("POST", "/admin/api/pages", CreatePageRequest{Title: "Bis", Slug: "musique"}),
		http.StatusUnprocessableEntity)
}

func TestListReorderDelete(t *testing.T) {
	s := newTestServer(t)
	s.login()
	a := s.createPage("A", "a")
	b := s.createPage("B", "b")

	expectStatus(t, s.do("PUT", "/admin/api/pages/order", ReorderRequest{IDs: []string{b.ID, a.ID}}), http.StatusNoContent)
	list := decode[PagesResponse](t, s.do("GET", "/admin/api/pages", nil)).Pages
	if len(list) != 2 || list[0].Slug != "b" {
		t.Errorf("pages = %+v", list)
	}
	expectStatus(t, s.do("PUT", "/admin/api/pages/order", ReorderRequest{IDs: []string{a.ID}}), http.StatusUnprocessableEntity)

	expectStatus(t, s.do("DELETE", "/admin/api/pages/"+a.ID, nil), http.StatusNoContent)
	expectStatus(t, s.do("GET", "/admin/api/pages/"+a.ID, nil), http.StatusNotFound)
}

func TestCheckSlug(t *testing.T) {
	s := newTestServer(t)
	s.login()
	doc := s.createPage("Musique", "musique")

	tests := []struct {
		path      string
		available bool
	}{
		{"/admin/api/slugs/jeux", true},
		{"/admin/api/slugs/musique", false},
		{"/admin/api/slugs/musique?except=" + doc.ID, true},
		{"/admin/api/slugs/admin", false},
		{"/admin/api/slugs/Jeux", false},
	}
	for _, tt := range tests {
		rec := s.do("GET", tt.path, nil)
		expectStatus(t, rec, http.StatusOK)
		resp := decode[SlugCheckResponse](t, rec)
		if resp.Available != tt.available || (resp.Error == "") != tt.available {
			t.Errorf("%s: %+v, want available=%v", tt.path, resp, tt.available)
		}
	}
}

func multipartFile(t *testing.T, data []byte) (io.Reader, string) {
	t.Helper()
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	fw, err := mw.CreateFormFile("file", "photo.png")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := fw.Write(data); err != nil {
		t.Fatal(err)
	}
	if err := mw.Close(); err != nil {
		t.Fatal(err)
	}
	return &buf, mw.FormDataContentType()
}

func TestUploadImage(t *testing.T) {
	s := newTestServer(t)
	s.login()

	img := image.NewNRGBA(image.Rect(0, 0, 800, 400))
	for x := range 800 {
		img.Set(x, 0, color.White)
	}
	var pngData bytes.Buffer
	if err := png.Encode(&pngData, img); err != nil {
		t.Fatal(err)
	}
	body, contentType := multipartFile(t, pngData.Bytes())
	rec := s.do("POST", "/admin/api/images", body, "Content-Type", contentType)
	expectStatus(t, rec, http.StatusCreated)
	vis := decode[ImageResponse](t, rec).Visual
	if vis.Kind != VisualImage || vis.Width != 512 || vis.Height != 256 || vis.URL != "/images/"+vis.ID {
		t.Fatalf("visual = %+v", vis)
	}

	rec = s.do("GET", vis.URL, nil)
	expectStatus(t, rec, http.StatusOK)
	if rec.Header().Get("Content-Type") != "image/png" || !strings.Contains(rec.Header().Get("Cache-Control"), "immutable") {
		t.Errorf("headers = %v", rec.Header())
	}

	// The image can be used as a page avatar, and becomes its Open Graph image.
	doc := s.createPage("Musique", "musique")
	doc.Avatar = &Visual{Kind: VisualImage, ID: vis.ID}
	doc.Published = true
	expectStatus(t, s.save(doc), http.StatusOK)
	if !strings.Contains(s.do("GET", "/musique", nil).Body.String(), "https://link.zazi.cool/images/"+vis.ID) {
		t.Error("the avatar image should be the og:image")
	}

	body, contentType = multipartFile(t, []byte(`<svg xmlns="http://www.w3.org/2000/svg"/>`))
	expectStatus(t, s.do("POST", "/admin/api/images", body, "Content-Type", contentType), http.StatusUnprocessableEntity)

	body, contentType = multipartFile(t, make([]byte, 2<<20))
	expectStatus(t, s.do("POST", "/admin/api/images", body, "Content-Type", contentType), http.StatusRequestEntityTooLarge)
}

func TestAdminShellAndStatic(t *testing.T) {
	s := newTestServer(t)
	for _, path := range []string{"/admin", "/admin/login", "/admin/pages/" + uuid(1)} {
		rec := s.do("GET", path, nil)
		expectStatus(t, rec, http.StatusOK)
		if !strings.Contains(rec.Body.String(), "/assets/admin.js") || rec.Header().Get("X-Robots-Tag") == "" {
			t.Errorf("GET %s: not the admin shell", path)
		}
	}

	// The admin shell carries a fresh CSP nonce, the same in the header and the page.
	first := s.do("GET", "/admin", nil)
	csp := first.Header().Get("Content-Security-Policy")
	nonce := strings.TrimPrefix(regexp.MustCompile(`'nonce-[^']+`).FindString(csp), "'nonce-")
	if nonce == "" || !strings.Contains(first.Body.String(), `<meta name="csp-nonce" content="`+nonce+`">`) {
		t.Errorf("admin CSP %q and page nonce do not match", csp)
	}
	if strings.Contains(s.do("GET", "/admin", nil).Header().Get("Content-Security-Policy"), nonce) {
		t.Error("the nonce must change on every response")
	}

	rec := s.do("GET", "/shapes/star-1.svg", nil)
	expectStatus(t, rec, http.StatusOK)
	if rec.Header().Get("Content-Security-Policy") != shapeCSP {
		t.Error("shapes need their own CSP (inline styles)")
	}
	rec = s.do("GET", "/assets/public.js", nil)
	expectStatus(t, rec, http.StatusOK)
	if !strings.Contains(rec.Header().Get("Cache-Control"), "immutable") {
		t.Error("hashed assets should be cached forever")
	}
}

func TestDevCSP(t *testing.T) {
	csp := contentSecurityPolicy("http://localhost:5173")
	for _, want := range []string{"script-src 'self' http://localhost:5173", "ws://localhost:5173"} {
		if !strings.Contains(csp, want) {
			t.Errorf("dev CSP %q misses %q", csp, want)
		}
	}
}
