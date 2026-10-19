package server

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

// fakeClock lets tests move time forward without sleeping.
type fakeClock struct{ t time.Time }

func (c *fakeClock) now() time.Time { return c.t }

func newTestLimiter(limit int) (*RateLimiter, *fakeClock) {
	clock := &fakeClock{t: time.Date(2026, 10, 19, 12, 0, 0, 0, time.UTC)}
	l := NewRateLimiter(limit, time.Hour)
	l.now = clock.now
	return l, clock
}

func TestRateLimiterSlidingWindow(t *testing.T) {
	l, clock := newTestLimiter(2)

	if ok, _ := l.Allow("a"); !ok {
		t.Fatal("1st event refused")
	}
	clock.t = clock.t.Add(30 * time.Minute)
	if ok, _ := l.Allow("a"); !ok {
		t.Fatal("2nd event refused")
	}
	ok, retry := l.Allow("a")
	if ok {
		t.Fatal("3rd event within the hour allowed")
	}
	if retry != 30*time.Minute {
		t.Errorf("retry after = %v, want 30m (until the 1st event leaves the window)", retry)
	}
	if ok, _ := l.Allow("b"); !ok {
		t.Error("another client is limited too")
	}

	clock.t = clock.t.Add(30*time.Minute + time.Second)
	if ok, _ := l.Allow("a"); !ok {
		t.Error("event refused once the 1st one left the window")
	}
}

func TestRateLimiterForgetsIdleClients(t *testing.T) {
	l, clock := newTestLimiter(1)
	l.Allow("a")
	clock.t = clock.t.Add(2 * time.Hour)
	l.Allow("b")
	if _, found := l.hits["a"]; found {
		t.Error("idle client still in memory after the sweep")
	}
}

func TestClientKey(t *testing.T) {
	tests := []struct {
		name       string
		remoteAddr string
		forwarded  []string
		want       string
	}{
		{"no proxy", "203.0.113.7:51000", nil, "203.0.113.7"},
		{"behind Traefik", "10.42.0.1:51000", []string{"203.0.113.7"}, "203.0.113.7"},
		{"forged entries are ignored", "10.42.0.1:51000", []string{"1.2.3.4, 203.0.113.7"}, "203.0.113.7"},
		{"repeated headers", "10.42.0.1:51000", []string{"1.2.3.4", "203.0.113.7"}, "203.0.113.7"},
		{"IPv6 grouped by /64", "10.42.0.1:51000", []string{"2001:db8:1:2:aaaa::1"}, "2001:db8:1:2::/64"},
		{"IPv4-mapped IPv6", "[::ffff:203.0.113.7]:51000", nil, "203.0.113.7"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodPost, "/", nil)
			r.RemoteAddr = tt.remoteAddr
			for _, v := range tt.forwarded {
				r.Header.Add("X-Forwarded-For", v)
			}
			if got := clientKey(r); got != tt.want {
				t.Errorf("clientKey = %q, want %q", got, tt.want)
			}
		})
	}
}

func postForm(t *testing.T, h http.Handler, ip string, form url.Values) *httptest.ResponseRecorder {
	t.Helper()
	r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(form.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r.Header.Set("X-Forwarded-For", ip)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, r)
	return rec
}

// countingHandler stands in for paste creation and counts how often it is reached.
type countingHandler struct{ calls int }

func (c *countingHandler) ServeHTTP(w http.ResponseWriter, _ *http.Request) {
	c.calls++
	w.WriteHeader(http.StatusSeeOther)
}

// CA7: the 11th creation from one IP within the hour gets a 429.
func TestEleventhCreationIsRateLimited(t *testing.T) {
	next := &countingHandler{}
	h := guardCreate(NewRateLimiter(10, RateWindow), next)
	form := url.Values{"content": {"hello"}}

	for i := 1; i <= 10; i++ {
		if rec := postForm(t, h, "203.0.113.7", form); rec.Code == http.StatusTooManyRequests {
			t.Fatalf("creation %d got 429", i)
		}
	}
	rec := postForm(t, h, "203.0.113.7", form)
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("11th creation status = %d, want 429", rec.Code)
	}
	if rec.Header().Get("Retry-After") == "" {
		t.Error("429 without Retry-After")
	}
	if next.calls != 10 {
		t.Errorf("creation handler reached %d times, want 10", next.calls)
	}
	if rec := postForm(t, h, "198.51.100.1", form); rec.Code == http.StatusTooManyRequests {
		t.Error("another IP is rate limited")
	}
}

// CA7: a submission with the honeypot filled in creates nothing.
func TestHoneypotDropsSubmission(t *testing.T) {
	next := &countingHandler{}
	h := guardCreate(NewRateLimiter(10, RateWindow), next)

	rec := postForm(t, h, "203.0.113.7", url.Values{"content": {"spam"}, "website": {"http://spam.example"}})
	if next.calls != 0 {
		t.Error("honeypot submission reached the creation handler")
	}
	if rec.Code != http.StatusSeeOther || rec.Header().Get("Location") != "/" {
		t.Errorf("honeypot response = %d to %q, want a silent 303 to /", rec.Code, rec.Header().Get("Location"))
	}

	postForm(t, h, "203.0.113.7", url.Values{"content": {"hello"}, "website": {""}})
	if next.calls != 1 {
		t.Error("empty honeypot was rejected")
	}
}

func TestCreateRouteIsGuarded(t *testing.T) {
	h := NewHandler(fakeDB{}, "dev", NewRateLimiter(1, RateWindow))
	postForm(t, h, "203.0.113.7", url.Values{"content": {"hello"}})
	if rec := postForm(t, h, "203.0.113.7", url.Values{"content": {"hello"}}); rec.Code != http.StatusTooManyRequests {
		t.Errorf("2nd POST / status = %d, want 429", rec.Code)
	}
}
