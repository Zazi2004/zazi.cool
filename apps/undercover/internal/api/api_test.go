package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"testing/fstest"

	"zazi.cool/undercover/internal/pairs"
	"zazi.cool/undercover/internal/store"
)

// fakeStore is an in-memory PairStore. It only implements what the handlers need,
// without the database rules covered by the store's integration tests.
type fakeStore struct {
	mu      sync.Mutex
	pairs   []pairs.Pair
	reports []string
	pingErr error
	lastF   store.Filter
}

func (f *fakeStore) Ping(context.Context) error { return f.pingErr }

func (f *fakeStore) RandomPair(_ context.Context, filter store.Filter) (pairs.Pair, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.lastF = filter
	for _, p := range f.pairs {
		if !p.Hidden {
			return p, nil
		}
	}
	return pairs.Pair{}, store.ErrNotFound
}

func (f *fakeStore) Pair(_ context.Context, id int64) (pairs.Pair, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, p := range f.pairs {
		if p.ID == id {
			return p, nil
		}
	}
	return pairs.Pair{}, store.ErrNotFound
}

func (f *fakeStore) AllPairs(context.Context) ([]pairs.Pair, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]pairs.Pair(nil), f.pairs...), nil
}

func (f *fakeStore) CreatePair(_ context.Context, c pairs.Candidate, extreme bool, source pairs.Source) (pairs.Pair, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	p := pairs.Pair{ID: int64(len(f.pairs) + 1), WordA: c.WordA, WordB: c.WordB, Extreme: extreme, Source: source}
	f.pairs = append(f.pairs, p)
	return p, nil
}

func (f *fakeStore) Report(_ context.Context, id int64, reason pairs.Reason, hash string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	for i := range f.pairs {
		if f.pairs[i].ID == id {
			f.reports = append(f.reports, string(reason)+":"+hash)
			return nil
		}
	}
	return store.ErrNotFound
}

func newTestHandler(st *fakeStore) http.Handler {
	return NewHandler(st, Config{
		IPHashSalt:      "test",
		Version:         "abc1234",
		SuggestionLimit: 100,
		ReportLimit:     100,
		CheckLimit:      100,
		Static: fstest.MapFS{
			"index.html":      {Data: []byte("<!doctype html>app")},
			"assets/app.js":   {Data: []byte("console.log(1)")},
			"favicon.svg":     {Data: []byte("<svg/>")},
			"assets/sub/x.js": {Data: []byte("x")},
		},
	})
}

func do(t *testing.T, h http.Handler, method, target, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, target, strings.NewReader(body))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func decode[T any](t *testing.T, rec *httptest.ResponseRecorder) T {
	t.Helper()
	var v T
	if err := json.NewDecoder(rec.Body).Decode(&v); err != nil {
		t.Fatalf("decode response %q: %v", rec.Body.String(), err)
	}
	return v
}

func TestHealth(t *testing.T) {
	st := &fakeStore{}
	h := newTestHandler(st)
	if rec := do(t, h, "GET", "/healthz", ""); rec.Code != http.StatusOK {
		t.Errorf("healthz = %d, want 200", rec.Code)
	}
	if rec := do(t, h, "GET", "/readyz", ""); rec.Code != http.StatusOK {
		t.Errorf("readyz = %d, want 200", rec.Code)
	}
	st.pingErr = errors.New("down")
	if rec := do(t, h, "GET", "/readyz", ""); rec.Code != http.StatusServiceUnavailable {
		t.Errorf("readyz with database down = %d, want 503", rec.Code)
	}
}

func TestVersion(t *testing.T) {
	rec := do(t, newTestHandler(&fakeStore{}), "GET", "/version", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("version = %d, want 200", rec.Code)
	}
	if got := strings.TrimSpace(rec.Body.String()); got != "abc1234" {
		t.Errorf("version body = %q, want abc1234", got)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "text/plain; charset=utf-8" {
		t.Errorf("Content-Type = %q", ct)
	}
}

func TestRandomPair(t *testing.T) {
	st := &fakeStore{pairs: []pairs.Pair{{ID: 7, WordA: "Chat", WordB: "Tigre"}}}
	h := newTestHandler(st)

	rec := do(t, h, "GET", "/api/pairs/random?extreme=true&suggestionsOnly=true&exclude=1,2", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", rec.Code, rec.Body)
	}
	got := decode[DrawResponse](t, rec)
	if got.ID != 7 {
		t.Errorf("id = %d, want 7", got.ID)
	}
	if (got.CivilSide == SideA) != (got.CivilWord == "Chat") || got.CivilWord == got.UndercoverWord {
		t.Errorf("inconsistent draw: %+v", got)
	}
	if st.lastF.Extreme != store.ExtremeIncluded || !st.lastF.SuggestionsOnly || len(st.lastF.Exclude) != 2 {
		t.Errorf("filter = %+v, want extreme, suggestions only and 2 exclusions", st.lastF)
	}

	extremeModes := map[string]store.ExtremeMode{
		"":               store.ExtremeExcluded,
		"?extreme=false": store.ExtremeExcluded,
		"?extreme=true":  store.ExtremeIncluded,
		"?extreme=only":  store.ExtremeOnly,
		"?extreme=other": store.ExtremeExcluded,
	}
	for query, want := range extremeModes {
		do(t, h, "GET", "/api/pairs/random"+query, "")
		if st.lastF.Extreme != want {
			t.Errorf("GET /api/pairs/random%s: extreme mode = %v, want %v", query, st.lastF.Extreme, want)
		}
	}

	if rec := do(t, h, "GET", "/api/pairs/random?exclude=1,x", ""); rec.Code != http.StatusBadRequest {
		t.Errorf("invalid exclude: status = %d, want 400", rec.Code)
	}
	st.pairs = nil
	if rec := do(t, h, "GET", "/api/pairs/random", ""); rec.Code != http.StatusNotFound {
		t.Errorf("empty bank: status = %d, want 404", rec.Code)
	}
}

func TestGuess(t *testing.T) {
	st := &fakeStore{pairs: []pairs.Pair{{ID: 1, WordA: "Chat", WordB: "Éléphant"}}}
	h := newTestHandler(st)

	tests := []struct {
		body        string
		wantStatus  int
		wantCorrect bool
	}{
		{`{"civilSide":"b","guess":"elephants"}`, http.StatusOK, true},
		{`{"civilSide":"a","guess":"elephant"}`, http.StatusOK, false},
		{`{"civilSide":"c","guess":"chat"}`, http.StatusUnprocessableEntity, false},
		{`{"civilSide":"a","guess":"chat","extra":1}`, http.StatusBadRequest, false},
		{`not json`, http.StatusBadRequest, false},
	}
	for _, tt := range tests {
		rec := do(t, h, "POST", "/api/pairs/1/guess", tt.body)
		if rec.Code != tt.wantStatus {
			t.Errorf("%s: status = %d, want %d", tt.body, rec.Code, tt.wantStatus)
			continue
		}
		if rec.Code == http.StatusOK {
			if got := decode[GuessResponse](t, rec); got.Correct != tt.wantCorrect {
				t.Errorf("%s: correct = %v, want %v", tt.body, got.Correct, tt.wantCorrect)
			}
		}
	}

	if rec := do(t, h, "POST", "/api/pairs/42/guess", `{"civilSide":"a","guess":"x"}`); rec.Code != http.StatusNotFound {
		t.Errorf("unknown pair: status = %d, want 404", rec.Code)
	}
	if rec := do(t, h, "POST", "/api/pairs/abc/guess", `{"civilSide":"a","guess":"x"}`); rec.Code != http.StatusBadRequest {
		t.Errorf("invalid id: status = %d, want 400", rec.Code)
	}
}

func TestCheckPair(t *testing.T) {
	st := &fakeStore{pairs: []pairs.Pair{{ID: 1, WordA: "Chat", WordB: "Tigre"}}}
	h := newTestHandler(st)

	tests := []struct {
		query string
		want  CheckStatus
	}{
		{"a=Pomme&b=Poire", CheckOK},
		{"a=tigres&b=le+chat", CheckDuplicate},
		{"a=Chats&b=Tigrr", CheckSimilar},
		{"a=Chat&b=", CheckInvalid},
	}
	for _, tt := range tests {
		rec := do(t, h, "GET", "/api/pairs/check?"+tt.query, "")
		if rec.Code != http.StatusOK {
			t.Errorf("%s: status = %d, want 200", tt.query, rec.Code)
			continue
		}
		got := decode[CheckResponse](t, rec)
		if got.Status != tt.want {
			t.Errorf("%s: check status = %q, want %q", tt.query, got.Status, tt.want)
		}
		if got.Status == CheckInvalid && got.Error == "" {
			t.Errorf("%s: invalid without error message", tt.query)
		}
	}
}

func TestCreatePair(t *testing.T) {
	st := &fakeStore{pairs: []pairs.Pair{{ID: 1, WordA: "Chat", WordB: "Tigre"}}}
	h := newTestHandler(st)

	tests := []struct {
		name       string
		body       string
		wantStatus int
	}{
		{"missing category", `{"wordA":"Pomme","wordB":"Poire"}`, http.StatusUnprocessableEntity},
		{"invalid word", `{"wordA":"Pomme","wordB":"","extreme":false}`, http.StatusUnprocessableEntity},
		{"honeypot", `{"wordA":"Pomme","wordB":"Poire","extreme":false,"website":"spam"}`, http.StatusBadRequest},
		{"duplicate", `{"wordA":"Tigre","wordB":"Chat","extreme":false}`, http.StatusConflict},
		{"duplicate even when confirmed", `{"wordA":"Tigre","wordB":"Chat","extreme":false,"confirmSimilar":true}`, http.StatusConflict},
		{"similar without confirmation", `{"wordA":"Chats","wordB":"Tigrr","extreme":false}`, http.StatusConflict},
		{"similar confirmed", `{"wordA":"Chats","wordB":"Tigrr","extreme":false,"confirmSimilar":true}`, http.StatusCreated},
		{"new pair", `{"wordA":"Pomme","wordB":"Poire","extreme":true}`, http.StatusCreated},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := do(t, h, "POST", "/api/pairs", tt.body)
			if rec.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d: %s", rec.Code, tt.wantStatus, rec.Body)
			}
		})
	}

	last := st.pairs[len(st.pairs)-1]
	if last.WordA != "Poire" || last.WordB != "Pomme" || !last.Extreme || last.Source != pairs.SourceSuggestion {
		t.Errorf("stored pair = %+v, want canonical extreme suggestion Poire / Pomme", last)
	}
}

func TestReportPair(t *testing.T) {
	st := &fakeStore{pairs: []pairs.Pair{{ID: 1, WordA: "Chat", WordB: "Tigre"}}}
	h := newTestHandler(st)

	if rec := do(t, h, "POST", "/api/pairs/1/reports", `{"reason":"remove"}`); rec.Code != http.StatusNoContent {
		t.Errorf("valid report: status = %d, want 204", rec.Code)
	}
	if rec := do(t, h, "POST", "/api/pairs/1/reports", `{"reason":"boring"}`); rec.Code != http.StatusUnprocessableEntity {
		t.Errorf("unknown reason: status = %d, want 422", rec.Code)
	}
	if rec := do(t, h, "POST", "/api/pairs/9/reports", `{"reason":"extreme"}`); rec.Code != http.StatusNotFound {
		t.Errorf("unknown pair: status = %d, want 404", rec.Code)
	}
	if len(st.reports) != 1 || strings.Contains(st.reports[0], "192.0.2.1") {
		t.Errorf("reports = %v, want one report with a hashed IP", st.reports)
	}
}

func TestRateLimit(t *testing.T) {
	h := NewHandler(&fakeStore{}, Config{CheckLimit: 2, SuggestionLimit: 1, ReportLimit: 1, Static: fstest.MapFS{}})
	for i, want := range []int{http.StatusOK, http.StatusOK, http.StatusTooManyRequests} {
		if rec := do(t, h, "GET", "/api/pairs/check?a=x&b=y", ""); rec.Code != want {
			t.Errorf("check #%d: status = %d, want %d", i+1, rec.Code, want)
		}
	}
}

func TestClientIP(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "10.0.0.1:1234"
	req.Header.Set("X-Forwarded-For", "6.6.6.6, 203.0.113.9")

	if got := clientIP(req, false); got != "10.0.0.1" {
		t.Errorf("without proxy: %q, want the connection address", got)
	}
	if got := clientIP(req, true); got != "203.0.113.9" {
		t.Errorf("behind proxy: %q, want the address appended by the proxy", got)
	}
}

func TestStatic(t *testing.T) {
	h := newTestHandler(&fakeStore{})

	tests := []struct {
		path, wantBody, wantCache string
	}{
		{"/", "<!doctype html>app", "no-cache"},
		{"/partie", "<!doctype html>app", "no-cache"},
		{"/assets", "<!doctype html>app", "no-cache"},
		{"/assets/app.js", "console.log(1)", "public, max-age=31536000, immutable"},
		{"/favicon.svg", "<svg/>", ""},
	}
	for _, tt := range tests {
		rec := do(t, h, "GET", tt.path, "")
		if rec.Code != http.StatusOK || rec.Body.String() != tt.wantBody {
			t.Errorf("GET %s = %d %q, want 200 %q", tt.path, rec.Code, rec.Body, tt.wantBody)
		}
		if got := rec.Header().Get("Cache-Control"); got != tt.wantCache {
			t.Errorf("GET %s Cache-Control = %q, want %q", tt.path, got, tt.wantCache)
		}
		if rec.Header().Get("Content-Security-Policy") == "" {
			t.Errorf("GET %s has no CSP header", tt.path)
		}
	}

	rec := do(t, h, "GET", "/api/unknown", "")
	if rec.Code != http.StatusNotFound || !strings.Contains(rec.Header().Get("Content-Type"), "json") {
		t.Errorf("unknown API route = %d %s, want a JSON 404", rec.Code, rec.Header().Get("Content-Type"))
	}
}
