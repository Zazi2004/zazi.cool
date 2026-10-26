package server

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/gif"
	"image/jpeg"
	"image/png"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"zazi.cool/home/internal/content"
)

func solid(w, h int) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := range h {
		for x := range w {
			img.SetNRGBA(x, y, color.NRGBA{R: 200, G: 80, B: 120, A: 255})
		}
	}
	return img
}

func jpegBytes(t *testing.T, w, h int) []byte {
	t.Helper()
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, solid(w, h), nil); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func pngBytes(t *testing.T, w, h int) []byte {
	t.Helper()
	var buf bytes.Buffer
	if err := png.Encode(&buf, solid(w, h)); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

// withEXIF inserts an EXIF segment right after the JPEG start marker, as a camera does.
func withEXIF(data []byte) []byte {
	payload := []byte("Exif\x00\x00GPS-SECRET")
	size := len(payload) + 2
	seg := append([]byte{0xFF, 0xE1, byte(size >> 8), byte(size)}, payload...)
	return append(append(append([]byte{}, data[:2]...), seg...), data[2:]...)
}

// postBanner sends the edit form as a browser does, with a file in banner_file.
func postBanner(t *testing.T, h http.Handler, c *http.Cookie, version, banner string, file []byte) *httptest.ResponseRecorder {
	t.Helper()
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	fields := editForm(version)
	fields.Set("banner", banner)
	for k, v := range fields {
		if err := mw.WriteField(k, v[0]); err != nil {
			t.Fatal(err)
		}
	}
	if file != nil {
		part, err := mw.CreateFormFile("banner_file", "banner.jpg")
		if err != nil {
			t.Fatal(err)
		}
		part.Write(file)
	}
	if err := mw.Close(); err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest("POST", "/admin", &buf)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	req.Header.Set("Origin", testBaseURL)
	req.AddCookie(c)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func uploadedID(t *testing.T, db *pgxpool.Pool) string {
	t.Helper()
	var id *string
	if err := db.QueryRow(context.Background(), `SELECT upload_image_id::text FROM content`).Scan(&id); err != nil {
		t.Fatal(err)
	}
	if id == nil {
		return ""
	}
	return *id
}

func imageCount(t *testing.T, db *pgxpool.Pool) int {
	t.Helper()
	var n int
	if err := db.QueryRow(context.Background(), `SELECT count(*) FROM images`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

// CA10 and CA13 (image part): a 3000 x 1000 JPEG with EXIF, sent from a plain form
// (no JavaScript), is served as 1600 x 533 without EXIF.
func TestUploadResizesAndStripsMetadata(t *testing.T) {
	db := migrated(t)
	h := adminHandler(t, db, testPassword)
	c := sessionCookie(t, h)

	rec := postBanner(t, h, c, "1", "upload", withEXIF(jpegBytes(t, 3000, 1000)))
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want 303: %s", rec.Code, rec.Body)
	}
	id := uploadedID(t, db)
	if id == "" {
		t.Fatal("the upload should be the page's image")
	}

	img := get(t, h, "/images/"+id)
	if img.Code != http.StatusOK || img.Header().Get("Content-Type") != "image/jpeg" {
		t.Fatalf("image: status %d, type %q", img.Code, img.Header().Get("Content-Type"))
	}
	cfg, format, err := image.DecodeConfig(img.Body)
	if err != nil || format != "jpeg" || cfg.Width != 1600 || cfg.Height != 533 {
		t.Errorf("served %dx%d %s (%v), want 1600x533 jpeg", cfg.Width, cfg.Height, format, err)
	}
	if bytes.Contains(img.Body.Bytes(), []byte("GPS-SECRET")) {
		t.Error("EXIF survived")
	}

	// CA2 (with an image): the <img> carries alt, width and height.
	home := get(t, h, "/").Body.String()
	want := regexp.MustCompile(`<img src="/images/` + id + `" alt="[^"]*" width="1600" height="533"`)
	if !want.MatchString(home) {
		t.Errorf("home should show the banner with alt, width and height:\n%s", home)
	}
}

// CA10: refused files.
func TestUploadRefusals(t *testing.T) {
	db := migrated(t)
	h := adminHandler(t, db, testPassword)
	c := sessionCookie(t, h)

	var gifData bytes.Buffer
	if err := gif.Encode(&gifData, solid(10, 10), nil); err != nil {
		t.Fatal(err)
	}
	cases := map[string][]byte{
		"gif":          gifData.Bytes(),
		"svg":          []byte(`<svg xmlns="http://www.w3.org/2000/svg"></svg>`),
		"text as jpg":  []byte("not an image at all"),
		"empty":        {},
		"4001 px wide": pngBytes(t, 4001, 1),
		"4001 px high": pngBytes(t, 1, 4001),
	}
	for name, data := range cases {
		rec := postBanner(t, h, c, "1", "upload", data)
		if rec.Code != http.StatusUnprocessableEntity {
			t.Errorf("%s: status = %d, want 422", name, rec.Code)
		}
	}
	if n := imageCount(t, db); n != 0 {
		t.Errorf("%d images stored, want none", n)
	}

	// 4000 px is accepted.
	if rec := postBanner(t, h, c, "1", "upload", pngBytes(t, 4000, 1)); rec.Code != http.StatusSeeOther {
		t.Errorf("4000 px: status = %d, want 303: %s", rec.Code, rec.Body)
	}
}

func TestUploadTooLarge(t *testing.T) {
	db := migrated(t)
	h := adminHandler(t, db, testPassword)
	c := sessionCookie(t, h)

	// Just over MAX_UPLOAD_MB: under the body limit, which leaves room for the
	// other fields, so the file itself is measured.
	big := append(jpegBytes(t, 10, 10), make([]byte, 5<<20)...)
	if rec := postBanner(t, h, c, "1", "upload", big); rec.Code != http.StatusRequestEntityTooLarge {
		t.Errorf("slightly over: status = %d, want 413", rec.Code)
	}
	// Far over: the body limit stops the read.
	huge := make([]byte, 6<<20)
	if rec := postBanner(t, h, c, "1", "upload", huge); rec.Code != http.StatusRequestEntityTooLarge {
		t.Errorf("far over: status = %d, want 413", rec.Code)
	}
	if n := imageCount(t, db); n != 0 {
		t.Errorf("%d images stored, want none", n)
	}
}

// F13: a file sent with another choice is refused.
func TestUploadWithAnotherChoice(t *testing.T) {
	db := migrated(t)
	h := adminHandler(t, db, testPassword)
	c := sessionCookie(t, h)

	rec := postBanner(t, h, c, "1", "none", jpegBytes(t, 100, 50))
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want 422", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "Choisissez soit une illustration, soit votre image") {
		t.Errorf("message missing: %s", rec.Body)
	}
	if n := imageCount(t, db); n != 0 {
		t.Errorf("%d images stored, want none", n)
	}
}

// F16: choosing "none" keeps the uploaded image, which "Mon image" gets back.
func TestUploadKeptWhenAnotherChoice(t *testing.T) {
	db := migrated(t)
	h := adminHandler(t, db, testPassword)
	c := sessionCookie(t, h)

	if rec := postBanner(t, h, c, "1", "upload", jpegBytes(t, 100, 50)); rec.Code != http.StatusSeeOther {
		t.Fatalf("upload: status %d", rec.Code)
	}
	id := uploadedID(t, db)
	if rec := postBanner(t, h, c, "2", "none", nil); rec.Code != http.StatusSeeOther {
		t.Fatalf("none: status %d: %s", rec.Code, rec.Body)
	}
	if got := uploadedID(t, db); got != id {
		t.Errorf("image id = %q, want %q kept", got, id)
	}
	if strings.Contains(get(t, h, "/").Body.String(), "/images/") {
		t.Error("the page should show no banner")
	}
	if !strings.Contains(getAdmin(h, c).Body.String(), `value="upload"`) {
		t.Error("Mon image should stay offered")
	}
	if rec := postBanner(t, h, c, "3", "upload", nil); rec.Code != http.StatusSeeOther {
		t.Fatalf("upload again: status %d: %s", rec.Code, rec.Body)
	}
	if !strings.Contains(get(t, h, "/").Body.String(), "/images/"+id) {
		t.Error("the banner should be back without a new upload")
	}
}

// A conflict stores no image (one transaction).
func TestUploadConflictStoresNothing(t *testing.T) {
	db := migrated(t)
	h := adminHandler(t, db, testPassword)
	c := sessionCookie(t, h)
	if rec := postForm(h, "/admin", editForm("1"), c); rec.Code != http.StatusSeeOther {
		t.Fatalf("first save: %d", rec.Code)
	}
	if rec := postBanner(t, h, c, "1", "upload", jpegBytes(t, 100, 50)); rec.Code != http.StatusConflict {
		t.Fatalf("status = %d, want 409", rec.Code)
	}
	if n := imageCount(t, db); n != 0 {
		t.Errorf("%d images stored, want none", n)
	}
}

// A new image gets a new identifier (F15).
func TestNewUploadNewID(t *testing.T) {
	db := migrated(t)
	h := adminHandler(t, db, testPassword)
	c := sessionCookie(t, h)
	postBanner(t, h, c, "1", "upload", jpegBytes(t, 100, 50))
	first := uploadedID(t, db)
	postBanner(t, h, c, "2", "upload", jpegBytes(t, 100, 50))
	if second := uploadedID(t, db); second == first || second == "" {
		t.Errorf("ids %q then %q, want different", first, second)
	}
}

// CA11
func TestImageEndpoint(t *testing.T) {
	db := migrated(t)
	h := adminHandler(t, db, testPassword)
	c := sessionCookie(t, h)
	postBanner(t, h, c, "1", "upload", pngBytes(t, 100, 50))

	rec := get(t, h, "/images/"+uploadedID(t, db))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	if got := rec.Header().Get("Cache-Control"); got != "public, max-age=31536000, immutable" {
		t.Errorf("Cache-Control = %q", got)
	}
	for _, path := range []string{
		"/images/018f0000-0000-7000-8000-000000000000", // well-formed, unknown
		"/images/not-an-id",
		"/images/1",
		"/images/%27%20OR%201=1",
	} {
		if rec := get(t, h, path); rec.Code != http.StatusNotFound {
			t.Errorf("%s: status = %d, want 404", path, rec.Code)
		}
	}
}

// CA12: the purge removes an image replaced more than 24 hours ago, and keeps
// the current one and a recent one.
func TestPurgeOrphanImages(t *testing.T) {
	db := migrated(t)
	h := adminHandler(t, db, testPassword)
	c := sessionCookie(t, h)
	store := content.NewStore(db)

	postBanner(t, h, c, "1", "upload", jpegBytes(t, 100, 50))
	oldReplaced := uploadedID(t, db)
	postBanner(t, h, c, "2", "upload", jpegBytes(t, 100, 50))
	recentReplaced := uploadedID(t, db)
	postBanner(t, h, c, "3", "upload", jpegBytes(t, 100, 50))
	current := uploadedID(t, db)

	age := func(id string, d time.Duration) {
		t.Helper()
		if _, err := db.Exec(context.Background(),
			`UPDATE images SET created_at = now() - $2::interval WHERE id = $1`, id, d.String()); err != nil {
			t.Fatal(err)
		}
	}
	age(oldReplaced, 25*time.Hour)
	age(recentReplaced, 2*time.Hour)
	age(current, 48*time.Hour) // old, but still used by the page

	cutoff := time.Now().Add(-24 * time.Hour)
	n, err := store.DeleteOrphanImages(context.Background(), cutoff)
	if err != nil || n != 1 {
		t.Fatalf("purge = %d, %v; want 1 deleted", n, err)
	}
	if get(t, h, "/images/"+oldReplaced).Code != http.StatusNotFound {
		t.Error("the old replaced image should be gone")
	}
	for name, id := range map[string]string{"recent": recentReplaced, "current": current} {
		if get(t, h, "/images/"+id).Code != http.StatusOK {
			t.Errorf("the %s image should remain", name)
		}
	}
	// Idempotent.
	if n, err := store.DeleteOrphanImages(context.Background(), cutoff); err != nil || n != 0 {
		t.Errorf("second purge = %d, %v; want 0", n, err)
	}
}
