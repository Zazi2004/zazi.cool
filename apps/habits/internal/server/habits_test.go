package server

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"zazi.cool/habits/internal/habit"
)

// memStore is an in-memory Store: it records the calls the handlers make and answers
// from fields the tests set. SQL behaviour is covered by the store package tests.
type memStore struct {
	habits  []habit.Habit
	items   []habit.DayItem
	history []habit.History
	histAsk [][2]string // from, to requested through History
	dayAsk  []time.Time // days requested through DayHabits
	setDone []setDoneCall
	err     error // returned by every habit write
}

type setDoneCall struct {
	id   int64
	day  string
	done bool
}

func (m *memStore) Ping(context.Context) error { return nil }

func (m *memStore) CreateHabit(_ context.Context, name string, days habit.Weekdays, _ time.Time) error {
	if m.err == nil {
		m.habits = append(m.habits, habit.Habit{ID: int64(len(m.habits) + 1), Name: name, Weekdays: days})
	}
	return m.err
}

func (m *memStore) UpdateHabit(context.Context, int64, string, habit.Weekdays, time.Time) error {
	return m.err
}
func (m *memStore) ArchiveHabit(context.Context, int64, time.Time) error { return m.err }
func (m *memStore) RestoreHabit(context.Context, int64) error            { return m.err }

func (m *memStore) ListHabits(context.Context, time.Time) ([]habit.Habit, error) {
	return m.habits, nil
}

func (m *memStore) DayHabits(_ context.Context, day time.Time) ([]habit.DayItem, error) {
	m.dayAsk = append(m.dayAsk, day)
	return m.items, nil
}

func (m *memStore) History(_ context.Context, from, to time.Time) ([]habit.History, error) {
	m.histAsk = append(m.histAsk, [2]string{habit.FormatDay(from), habit.FormatDay(to)})
	return m.history, nil
}

func (m *memStore) SetDone(_ context.Context, id int64, day time.Time, done bool) error {
	if m.err != nil {
		return m.err
	}
	m.setDone = append(m.setDone, setDoneCall{id, habit.FormatDay(day), done})
	return nil
}

// fixedNow is 23:30 UTC on 30 June 2026: already 1 July in Paris (CA4).
var fixedNow = time.Date(2026, time.June, 30, 23, 30, 0, 0, time.UTC)

func newHabitsHandler(t *testing.T, m *memStore) http.Handler {
	t.Helper()
	paris, err := time.LoadLocation("Europe/Paris")
	if err != nil {
		t.Fatal(err)
	}
	return NewHandler(Config{
		Store: m, Auth: fakeAuth{}, BaseURL: testBaseURL, LoginLimit: 10,
		Location: paris, EditWindowDays: 14, Now: func() time.Time { return fixedNow },
	})
}

func do(h http.Handler, method, path string, form url.Values, headers map[string]string) *httptest.ResponseRecorder {
	var req *http.Request
	if form != nil {
		req = httptest.NewRequest(method, path, strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	} else {
		req = httptest.NewRequest(method, path, nil)
	}
	req.AddCookie(&http.Cookie{Name: "session", Value: "ok"})
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

// CA4: "today" follows TZ_NAME, not the server clock's zone.
func TestHomeShowsTodayInConfiguredZone(t *testing.T) {
	m := &memStore{items: []habit.DayItem{{HabitID: 1, Name: "Lire"}}}
	rec := do(newHabitsHandler(t, m), http.MethodGet, "/", nil, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	if got := habit.FormatDay(m.dayAsk[0]); got != "2026-07-01" {
		t.Errorf("day asked = %s, want 2026-07-01", got)
	}
	body := rec.Body.String()
	for _, want := range []string{"mercredi 1 juillet 2026", "Lire", `action="/day/2026-07-01/toggle"`} {
		if !strings.Contains(body, want) {
			t.Errorf("page lacks %q", want)
		}
	}
}

// CA1 (F4): the habit appears unchecked, and the button offers to check it.
func TestDayPageCheckBoxes(t *testing.T) {
	m := &memStore{items: []habit.DayItem{{HabitID: 1, Name: "Lire"}, {HabitID: 2, Name: "Sport", Done: true}}}
	body := do(newHabitsHandler(t, m), http.MethodGet, "/day/2026-06-29", nil, nil).Body.String()
	if !strings.Contains(body, `name="done" value="1"`) || !strings.Contains(body, `name="done" value="0"`) {
		t.Errorf("want one form checking and one unchecking:\n%s", body)
	}
	if !strings.Contains(body, `aria-pressed="true"`) || !strings.Contains(body, `aria-pressed="false"`) {
		t.Error("aria-pressed missing")
	}
}

func TestDayPageNavigationAndEditability(t *testing.T) {
	h := func() http.Handler { return newHabitsHandler(t, &memStore{}) }
	tests := []struct {
		path         string
		wantEditable bool
		wantPrev     string
		wantNext     string
	}{
		{"/", true, "/day/2026-06-30", ""},
		{"/day/2026-07-01", true, "/day/2026-06-30", ""},
		{"/day/2026-06-30", true, "/day/2026-06-29", "/"},
		{"/day/2026-06-17", true, "", "/day/2026-06-18"}, // oldest editable day: nothing older to offer
		{"/day/2026-06-16", false, "", "/day/2026-06-17"},
		{"/day/2026-07-02", false, "/", ""}, // future
	}
	for _, tt := range tests {
		body := do(h(), http.MethodGet, tt.path, nil, nil).Body.String()
		if got := !strings.Contains(body, "n'est pas modifiable"); got != tt.wantEditable {
			t.Errorf("%s editable = %v, want %v", tt.path, got, tt.wantEditable)
		}
		if got := strings.Contains(body, `rel="prev"`); got != (tt.wantPrev != "") {
			t.Errorf("%s has prev link = %v", tt.path, got)
		}
		if tt.wantPrev != "" && !strings.Contains(body, `href="`+tt.wantPrev+`" rel="prev"`) {
			t.Errorf("%s: prev link should be %s", tt.path, tt.wantPrev)
		}
		if got := strings.Contains(body, `rel="next"`); got != (tt.wantNext != "") {
			t.Errorf("%s has next link = %v", tt.path, got)
		}
		if tt.wantNext != "" && !strings.Contains(body, `href="`+tt.wantNext+`" rel="next"`) {
			t.Errorf("%s: next link should be %s", tt.path, tt.wantNext)
		}
	}
}

func TestDayInvalidDate(t *testing.T) {
	h := newHabitsHandler(t, &memStore{})
	for _, path := range []string{"/day/hier", "/day/2026-02-30", "/day/2026-7-1"} {
		if rec := do(h, http.MethodGet, path, nil, nil); rec.Code != http.StatusBadRequest {
			t.Errorf("GET %s = %d, want 400", path, rec.Code)
		}
	}
	if rec := do(h, http.MethodPost, "/day/x/toggle", url.Values{}, nil); rec.Code != http.StatusBadRequest {
		t.Errorf("POST toggle on a bad date = %d, want 400", rec.Code)
	}
}

// CA3 (F5): inside the window it works; outside it, or in the future, 422 and no write.
func TestToggleWindow(t *testing.T) {
	tests := []struct {
		day      string
		wantCode int
		wantCall bool
	}{
		{"2026-07-01", http.StatusSeeOther, true}, // today
		{"2026-06-28", http.StatusSeeOther, true}, // 3 days ago
		{"2026-06-17", http.StatusSeeOther, true}, // window edge
		{"2026-06-16", http.StatusUnprocessableEntity, false},
		{"2026-07-02", http.StatusUnprocessableEntity, false},
	}
	for _, tt := range tests {
		m := &memStore{}
		rec := do(newHabitsHandler(t, m), http.MethodPost, "/day/"+tt.day+"/toggle",
			url.Values{"habit_id": {"7"}, "done": {"1"}}, nil)
		if rec.Code != tt.wantCode {
			t.Errorf("%s: status = %d, want %d", tt.day, rec.Code, tt.wantCode)
		}
		if got := len(m.setDone) == 1; got != tt.wantCall {
			t.Errorf("%s: store written = %v, want %v", tt.day, got, tt.wantCall)
		}
	}
}

func TestToggleRedirectAndJSON(t *testing.T) {
	m := &memStore{}
	h := newHabitsHandler(t, m)
	form := url.Values{"habit_id": {"7"}, "done": {"1"}}

	rec := do(h, http.MethodPost, "/day/2026-07-01/toggle", form, nil)
	if rec.Code != http.StatusSeeOther || rec.Header().Get("Location") != "/" {
		t.Errorf("today: %d -> %q, want 303 -> /", rec.Code, rec.Header().Get("Location"))
	}
	rec = do(h, http.MethodPost, "/day/2026-06-28/toggle", form, nil)
	if rec.Header().Get("Location") != "/day/2026-06-28" {
		t.Errorf("past day redirect = %q", rec.Header().Get("Location"))
	}

	rec = do(h, http.MethodPost, "/day/2026-06-28/toggle", url.Values{"habit_id": {"7"}, "done": {"0"}},
		map[string]string{"Accept": "application/json"})
	if rec.Code != http.StatusOK || rec.Header().Get("Content-Type") != "application/json" {
		t.Fatalf("json: %d %q", rec.Code, rec.Header().Get("Content-Type"))
	}
	if got, want := strings.TrimSpace(rec.Body.String()), `{"habit_id":7,"day":"2026-06-28","done":false}`; got != want {
		t.Errorf("json body = %s, want %s", got, want)
	}
	if last := m.setDone[len(m.setDone)-1]; last != (setDoneCall{7, "2026-06-28", false}) {
		t.Errorf("last call = %+v", last)
	}
}

func TestToggleErrors(t *testing.T) {
	tests := []struct {
		name     string
		form     url.Values
		storeErr error
		want     int
	}{
		{"missing habit", url.Values{"done": {"1"}}, nil, http.StatusUnprocessableEntity},
		{"bad habit id", url.Values{"habit_id": {"x"}, "done": {"1"}}, nil, http.StatusUnprocessableEntity},
		{"bad done", url.Values{"habit_id": {"1"}, "done": {"yes"}}, nil, http.StatusUnprocessableEntity},
		{"unknown habit", url.Values{"habit_id": {"1"}, "done": {"1"}}, habit.ErrNotFound, http.StatusNotFound},
		{"not scheduled", url.Values{"habit_id": {"1"}, "done": {"1"}}, habit.ErrNotScheduled, http.StatusUnprocessableEntity},
		{"store failure", url.Values{"habit_id": {"1"}, "done": {"1"}}, errors.New("db down"), http.StatusInternalServerError},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newHabitsHandler(t, &memStore{err: tt.storeErr})
			if rec := do(h, http.MethodPost, "/day/2026-07-01/toggle", tt.form, nil); rec.Code != tt.want {
				t.Errorf("status = %d, want %d", rec.Code, tt.want)
			}
		})
	}
}

func TestHabitWritesNeedASession(t *testing.T) {
	h := newHabitsHandler(t, &memStore{})
	for _, path := range []string{"/habits", "/habits/1", "/habits/1/archive", "/habits/1/restore", "/day/2026-07-01/toggle"} {
		req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(""))
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if rec.Code != http.StatusSeeOther || rec.Header().Get("Location") != "/login" {
			t.Errorf("POST %s without session = %d -> %q, want 303 -> /login", path, rec.Code, rec.Header().Get("Location"))
		}
	}
}

func TestCrossOriginHabitPostIsRefused(t *testing.T) {
	m := &memStore{}
	rec := do(newHabitsHandler(t, m), http.MethodPost, "/habits", url.Values{"name": {"Lire"}},
		map[string]string{"Origin": "https://evil.example"})
	if rec.Code != http.StatusForbidden || len(m.habits) != 0 {
		t.Errorf("cross-origin POST = %d, habits = %d; want 403 and none", rec.Code, len(m.habits))
	}
}

func TestCreateHabit(t *testing.T) {
	m := &memStore{}
	h := newHabitsHandler(t, m)

	// No weekday ticked: every day (F1).
	rec := do(h, http.MethodPost, "/habits", url.Values{"name": {"  Lire "}}, nil)
	if rec.Code != http.StatusSeeOther || rec.Header().Get("Location") != "/habits" {
		t.Fatalf("create = %d -> %q", rec.Code, rec.Header().Get("Location"))
	}
	if got := m.habits[0]; got.Name != "Lire" || got.Weekdays != habit.AllDays {
		t.Errorf("created = %+v, want trimmed name and all days", got)
	}

	do(h, http.MethodPost, "/habits", url.Values{"name": {"Sport"}, "weekdays": {"1", "3", "5"}}, nil)
	if got := m.habits[1].Weekdays; got != 0b0010101 {
		t.Errorf("weekdays = %07b, want 0010101", got)
	}
}

func TestCreateHabitRefusals(t *testing.T) {
	tests := []struct {
		name     string
		form     url.Values
		storeErr error
		want     int
	}{
		{"empty name", url.Values{"name": {" "}}, nil, http.StatusUnprocessableEntity},
		{"long name", url.Values{"name": {strings.Repeat("a", 61)}}, nil, http.StatusUnprocessableEntity},
		{"bad weekday", url.Values{"name": {"x"}, "weekdays": {"8"}}, nil, http.StatusUnprocessableEntity},
		{"duplicate", url.Values{"name": {"Lire"}}, habit.ErrNameTaken, http.StatusConflict},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &memStore{err: tt.storeErr}
			rec := do(newHabitsHandler(t, m), http.MethodPost, "/habits", tt.form, nil)
			if rec.Code != tt.want {
				t.Errorf("status = %d, want %d", rec.Code, tt.want)
			}
			if !strings.Contains(rec.Body.String(), `role="alert"`) {
				t.Error("the page should show the error")
			}
		})
	}
}

func TestUpdateArchiveRestore(t *testing.T) {
	h := newHabitsHandler(t, &memStore{})
	form := url.Values{"name": {"Lire"}, "weekdays": {"1"}}
	for _, path := range []string{"/habits/3", "/habits/3/archive", "/habits/3/restore"} {
		rec := do(h, http.MethodPost, path, form, nil)
		if rec.Code != http.StatusSeeOther || rec.Header().Get("Location") != "/habits" {
			t.Errorf("POST %s = %d -> %q, want 303 -> /habits", path, rec.Code, rec.Header().Get("Location"))
		}
	}

	if rec := do(h, http.MethodPost, "/habits/3", url.Values{"name": {"Lire"}}, nil); rec.Code != http.StatusUnprocessableEntity {
		t.Errorf("update with no weekday = %d, want 422", rec.Code)
	}
	if rec := do(h, http.MethodPost, "/habits/abc/archive", nil, nil); rec.Code != http.StatusNotFound {
		t.Errorf("bad id = %d, want 404", rec.Code)
	}
	for _, tt := range []struct {
		err  error
		want int
	}{{habit.ErrNotFound, 404}, {habit.ErrNameTaken, 409}, {errors.New("db down"), 500}} {
		h := newHabitsHandler(t, &memStore{err: tt.err})
		if rec := do(h, http.MethodPost, "/habits/3", form, nil); rec.Code != tt.want {
			t.Errorf("update with %v = %d, want %d", tt.err, rec.Code, tt.want)
		}
	}
}

// CA13: a name is text, never markup.
func TestHabitNameIsEscaped(t *testing.T) {
	const name = `<script>alert(1)</script>`
	m := &memStore{
		items:  []habit.DayItem{{HabitID: 1, Name: name}},
		habits: []habit.Habit{{ID: 1, Name: name, Weekdays: habit.AllDays}},
	}
	h := newHabitsHandler(t, m)
	for _, path := range []string{"/", "/habits"} {
		body := do(h, http.MethodGet, path, nil, nil).Body.String()
		if strings.Contains(body, "<script>") {
			t.Errorf("GET %s: name rendered as markup", path)
		}
		if !strings.Contains(body, "&lt;script&gt;") {
			t.Errorf("GET %s: escaped name missing", path)
		}
	}
}

func TestHabitsPageSplitsActiveAndArchived(t *testing.T) {
	archived := time.Date(2026, time.June, 1, 0, 0, 0, 0, time.UTC)
	m := &memStore{habits: []habit.Habit{
		{ID: 1, Name: "Lire", Weekdays: 0b0000001},
		{ID: 2, Name: "Ancienne", ArchivedOn: &archived, Weekdays: habit.AllDays},
	}}
	body := do(newHabitsHandler(t, m), http.MethodGet, "/habits", nil, nil).Body.String()
	if !strings.Contains(body, `action="/habits/1/archive"`) || strings.Contains(body, `action="/habits/2/archive"`) {
		t.Error("only the active habit can be archived")
	}
	if !strings.Contains(body, `action="/habits/2/restore"`) {
		t.Error("the archived habit can be restored")
	}
}
