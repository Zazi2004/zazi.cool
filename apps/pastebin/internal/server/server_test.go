package server

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"zazi.cool/pastebin/internal/paste"
	"zazi.cool/pastebin/internal/store"
)

// fakeStore keeps pastes in memory. Pastes whose ID starts with "exp" are expired.
type fakeStore struct {
	mu      sync.Mutex
	pingErr error
	pastes  map[string]paste.Paste
	lastTTL time.Duration
}

func newFakeStore() *fakeStore {
	return &fakeStore{pastes: make(map[string]paste.Paste)}
}

func (f *fakeStore) Ping(context.Context) error { return f.pingErr }

func (f *fakeStore) Create(_ context.Context, content, language string, ttl time.Duration) (paste.Paste, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	id, err := paste.NewID()
	if err != nil {
		return paste.Paste{}, err
	}
	now := time.Now()
	p := paste.Paste{ID: id, Content: content, Language: language, CreatedAt: now, ExpiresAt: now.Add(ttl)}
	f.pastes[id] = p
	f.lastTTL = ttl
	return p, nil
}

func (f *fakeStore) Get(_ context.Context, id string) (paste.Paste, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	p, ok := f.pastes[id]
	if !ok || p.ExpiresAt.Before(time.Now()) {
		return paste.Paste{}, store.ErrNotFound
	}
	return p, nil
}

func (f *fakeStore) count() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.pastes)
}

const testMaxBytes = 1024

func newTestHandler(s *fakeStore) http.Handler {
	return NewHandler(Config{
		Store:         s,
		Version:       "abc1234",
		MaxPasteBytes: testMaxBytes,
		BaseURL:       "https://pb.example/",
		Limiter:       NewRateLimiter(100, RateWindow),
	})
}

func do(t *testing.T, h http.Handler, method, path string) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(method, path, nil))
	return rec
}

func post(t *testing.T, h http.Handler, form url.Values) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestEndpoints(t *testing.T) {
	tests := []struct {
		name       string
		pingErr    error
		path       string
		wantStatus int
		wantBody   string
	}{
		{"healthz ignores the database", errors.New("down"), "/healthz", http.StatusOK, "ok"},
		{"readyz ok", nil, "/readyz", http.StatusOK, "ok"},
		{"readyz database down", errors.New("down"), "/readyz", http.StatusServiceUnavailable, "database unavailable"},
		{"version", nil, "/version", http.StatusOK, "abc1234"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := newFakeStore()
			s.pingErr = tt.pingErr
			rec := do(t, newTestHandler(s), http.MethodGet, tt.path)
			if rec.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
			if got := strings.TrimSpace(rec.Body.String()); got != tt.wantBody {
				t.Errorf("body = %q, want %q", got, tt.wantBody)
			}
			if ct := rec.Header().Get("Content-Type"); ct != "text/plain; charset=utf-8" {
				t.Errorf("Content-Type = %q", ct)
			}
		})
	}
}

func TestNonGetIsRejected(t *testing.T) {
	rec := do(t, newTestHandler(newFakeStore()), http.MethodPost, "/healthz")
	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("POST /healthz status = %d, want 405", rec.Code)
	}
}

// CA9: the form offers exactly the five durations, with 10 minutes preselected.
func TestFormOffersExpirations(t *testing.T) {
	rec := do(t, newTestHandler(newFakeStore()), http.MethodGet, "/")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	body := rec.Body.String()
	for _, opt := range []string{
		`<option value="10m" selected>10 minutes</option>`,
		`<option value="1h">1 heure</option>`,
		`<option value="1d">1 jour</option>`,
		`<option value="1w">1 semaine</option>`,
		`<option value="1mo">1 mois</option>`,
		`<option value="plaintext" selected>Texte brut</option>`,
		`name="website"`,
	} {
		if !strings.Contains(body, opt) {
			t.Errorf("form lacks %s", opt)
		}
	}
	if n := strings.Count(body, `name="expires_in"`); n != 1 {
		t.Errorf("%d expires_in fields", n)
	}
	if strings.Contains(strings.ToLower(body), "jamais") || strings.Contains(body, "never") {
		t.Error("form offers a never-expiring option")
	}
}

// CA1, CA2, CA6: create with the default expiration, read the formatted and raw views.
func TestCreateAndRead(t *testing.T) {
	s := newFakeStore()
	h := newTestHandler(s)
	content := "<script>alert(1)</script>\r\nsecond line é\n"

	rec := post(t, h, url.Values{"content": {content}, "language": {"go"}})
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("POST status = %d, want 303; body: %s", rec.Code, rec.Body)
	}
	loc := rec.Header().Get("Location")
	id := strings.TrimPrefix(loc, "/")
	if !paste.ValidID(id) {
		t.Fatalf("Location = %q, want /<8 base62 characters>", loc)
	}
	if s.lastTTL != 10*time.Minute {
		t.Errorf("TTL = %v, want the 10 minute default", s.lastTTL)
	}

	view := do(t, h, http.MethodGet, loc)
	if view.Code != http.StatusOK {
		t.Fatalf("GET %s status = %d", loc, view.Code)
	}
	body := view.Body.String()
	if strings.Contains(body, "<script>alert") {
		t.Error("pasted script is rendered as HTML")
	}
	for _, want := range []string{
		`<span class="line">&lt;script&gt;alert(1)&lt;/script&gt;</span>`,
		`<span class="line">second line é</span>`,
		`href="/raw/` + id + `"`,
		`value="https://pb.example/` + id + `"`,
		"Go",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("view lacks %s", want)
		}
	}

	raw := do(t, h, http.MethodGet, "/raw/"+id)
	if raw.Code != http.StatusOK {
		t.Fatalf("GET /raw status = %d", raw.Code)
	}
	if got := raw.Body.String(); got != content {
		t.Errorf("raw body = %q, want %q", got, content)
	}
	for header, want := range map[string]string{
		"Content-Type":            "text/plain; charset=utf-8",
		"X-Content-Type-Options":  "nosniff",
		"Content-Security-Policy": "sandbox",
	} {
		if got := raw.Header().Get(header); got != want {
			t.Errorf("raw %s = %q, want %q", header, got, want)
		}
	}
}

func TestCreateKeepsChosenExpiration(t *testing.T) {
	s := newFakeStore()
	rec := post(t, newTestHandler(s), url.Values{"content": {"x"}, "expires_in": {"1mo"}})
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status = %d", rec.Code)
	}
	if s.lastTTL != 30*24*time.Hour {
		t.Errorf("TTL = %v, want 720h", s.lastTTL)
	}
}

// CA5, CA9: invalid input is refused with a French message, the form keeps the
// content, and nothing is stored.
func TestCreateRejectsInvalidInput(t *testing.T) {
	tests := []struct {
		name       string
		form       url.Values
		wantStatus int
		wantMsg    string
	}{
		{"empty", url.Values{"content": {""}}, http.StatusUnprocessableEntity, "Le paste est vide."},
		{"blank", url.Values{"content": {" \n\t"}}, http.StatusUnprocessableEntity, "Le paste est vide."},
		{"too large", url.Values{"content": {strings.Repeat("a", testMaxBytes+1)}}, http.StatusRequestEntityTooLarge, "taille maximale de 1 Kio"},
		{"not UTF-8", url.Values{"content": {"abc\xff"}}, http.StatusUnprocessableEntity, "UTF-8"},
		{"never expires", url.Values{"content": {"keep me"}, "expires_in": {"never"}}, http.StatusUnprocessableEntity, "durée d&#39;expiration"},
		{"unknown language", url.Values{"content": {"keep me"}, "language": {"cobol"}}, http.StatusUnprocessableEntity, "langage"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := newFakeStore()
			rec := post(t, newTestHandler(s), tt.form)
			if rec.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
			body := rec.Body.String()
			if !strings.Contains(body, tt.wantMsg) {
				t.Errorf("body lacks %q", tt.wantMsg)
			}
			if tt.form.Get("content") == "keep me" && !strings.Contains(body, ">keep me</textarea>") {
				t.Error("form does not keep the submitted content")
			}
			if n := s.count(); n != 0 {
				t.Errorf("%d pastes stored, want 0", n)
			}
		})
	}
}

func TestCreateRejectsOversizedBody(t *testing.T) {
	s := newFakeStore()
	// Each "é" becomes "%C3%A9": 6 bytes on the wire, beyond the 3x allowance.
	rec := post(t, newTestHandler(s), url.Values{"content": {strings.Repeat("é", 2*testMaxBytes)}})
	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Errorf("status = %d, want 413", rec.Code)
	}
	if s.count() != 0 {
		t.Error("oversized paste stored")
	}
}

// CA3: expired, unknown and malformed identifiers get the same 404, on both views.
func TestNotFoundIsIndistinguishable(t *testing.T) {
	s := newFakeStore()
	s.pastes["expired1"] = paste.Paste{ID: "expired1", Content: "old", ExpiresAt: time.Now().Add(-time.Second)}
	h := newTestHandler(s)

	for _, prefix := range []string{"/", "/raw/"} {
		var first string
		for _, id := range []string{"expired1", "unknown1", "bad-id"} {
			rec := do(t, h, http.MethodGet, prefix+id)
			if rec.Code != http.StatusNotFound {
				t.Errorf("GET %s%s status = %d, want 404", prefix, id, rec.Code)
			}
			if first == "" {
				first = rec.Body.String()
			} else if rec.Body.String() != first {
				t.Errorf("GET %s%s: 404 body differs from the expired one", prefix, id)
			}
		}
	}
}

// CA6 / NF1 / NF2: every page carries the CSP and the privacy headers.
func TestSecurityHeaders(t *testing.T) {
	s := newFakeStore()
	h := newTestHandler(s)
	p, _ := s.Create(context.Background(), "x", "plaintext", time.Minute)

	for _, path := range []string{"/", "/" + p.ID, "/unknown1", "/healthz"} {
		rec := do(t, h, http.MethodGet, path)
		hdr := rec.Header()
		if got := hdr.Get("Content-Security-Policy"); got != pageCSP {
			t.Errorf("%s CSP = %q", path, got)
		}
		if !strings.Contains(pageCSP, "default-src 'none'") || !strings.Contains(pageCSP, "frame-ancestors 'none'") {
			t.Error("CSP lost its default-src or frame-ancestors directive")
		}
		for header, want := range map[string]string{
			"X-Content-Type-Options": "nosniff",
			"X-Robots-Tag":           "noindex, nofollow",
			"Referrer-Policy":        "no-referrer",
		} {
			if got := hdr.Get(header); got != want {
				t.Errorf("%s %s = %q, want %q", path, header, got, want)
			}
		}
		if hdr.Get("Set-Cookie") != "" {
			t.Errorf("%s sets a cookie", path)
		}
	}
}

func TestStaticFiles(t *testing.T) {
	h := newTestHandler(newFakeStore())
	for _, path := range []string{"/static/style.css", "/static/copy.js"} {
		rec := do(t, h, http.MethodGet, path)
		if rec.Code != http.StatusOK || rec.Body.Len() == 0 {
			t.Errorf("GET %s status = %d, %d bytes", path, rec.Code, rec.Body.Len())
		}
	}
}

func TestLines(t *testing.T) {
	tests := map[string][]string{
		"a":          {"a"},
		"a\n":        {"a"},
		"a\r\nb\r\n": {"a", "b"},
		"a\n\nb":     {"a", "", "b"},
	}
	for in, want := range tests {
		got := lines(in)
		if strings.Join(got, "|") != strings.Join(want, "|") {
			t.Errorf("lines(%q) = %q, want %q", in, got, want)
		}
	}
}
