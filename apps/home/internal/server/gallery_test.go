package server

import (
	"context"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"testing"

	"zazi.cool/home/internal/content"
)

var illustrationURL = regexp.MustCompile(`/assets/banners/aube\.[0-9a-f]{10}\.jpg`)

// CA21 (F16, F16b, F16c) and the illustration part of CA13: from a plain form,
// choose an illustration while an uploaded image exists, then go back to the image.
func TestChooseIllustrationKeepsUpload(t *testing.T) {
	db := migrated(t)
	h := adminHandler(t, db, testPassword)
	c := sessionCookie(t, h)

	if rec := postBanner(t, h, c, "1", "upload", jpegBytes(t, 1600, 533)); rec.Code != http.StatusSeeOther {
		t.Fatalf("upload: status %d: %s", rec.Code, rec.Body)
	}
	id := uploadedID(t, db)

	rec := postForm(h, "/admin", aubeForm("2"), c)
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("choose aube: status %d: %s", rec.Code, rec.Body)
	}

	body := get(t, h, "/").Body.String()
	url := illustrationURL.FindString(body)
	if url == "" {
		t.Fatalf("the page does not show the aube picture:\n%s", body)
	}
	aube, _ := content.FindIllustration("aube")
	wantImg := `<img src="` + url + `" alt="` + aube.Alt + `" width="1600" height="533"`
	if !strings.Contains(strings.ReplaceAll(body, "&#39;", "'"), wantImg) {
		t.Errorf("page lacks %s in:\n%s", wantImg, body)
	}
	if got := meta(t, body, "og:image"); got != testBaseURL+url {
		t.Errorf("og:image = %q, want %q", got, testBaseURL+url)
	}

	// The picture is served with a year of cache.
	pic := get(t, h, url)
	if pic.Code != http.StatusOK || pic.Header().Get("Content-Type") != "image/jpeg" ||
		pic.Header().Get("Cache-Control") != "public, max-age=31536000, immutable" {
		t.Errorf("picture: status %d, type %q, cache %q", pic.Code, pic.Header().Get("Content-Type"), pic.Header().Get("Cache-Control"))
	}

	// « Mon image » is still offered, and the uploaded image is still stored.
	admin := getAdmin(h, c).Body.String()
	if !strings.Contains(admin, `value="upload"`) || !strings.Contains(admin, "/images/"+id) {
		t.Errorf("the admin no longer offers « Mon image »:\n%s", admin)
	}
	if !strings.Contains(admin, `value="aube" checked`) {
		t.Error("the aube radio should be checked")
	}

	// Choosing it again brings it back, without any new upload.
	form := editForm("3")
	form.Set("banner", "upload")
	if rec := postForm(h, "/admin", form, c); rec.Code != http.StatusSeeOther {
		t.Fatalf("choose upload: status %d: %s", rec.Code, rec.Body)
	}
	if got := get(t, h, "/").Body.String(); !strings.Contains(got, `src="/images/`+id+`"`) {
		t.Errorf("the uploaded image did not come back:\n%s", got)
	}
	if n := imageCount(t, db); n != 1 {
		t.Errorf("%d images stored, want the single upload", n)
	}
}

// F16c: a saved alternative text wins over the illustration's own.
func TestIllustrationSavedAlt(t *testing.T) {
	db := migrated(t)
	h := adminHandler(t, db, testPassword)
	c := sessionCookie(t, h)

	form := editForm("1")
	form.Set("banner", "lagon")
	form.Set("banner_alt", "Mon texte")
	if rec := postForm(h, "/admin", form, c); rec.Code != http.StatusSeeOther {
		t.Fatalf("status %d: %s", rec.Code, rec.Body)
	}
	if body := get(t, h, "/").Body.String(); !strings.Contains(body, `alt="Mon texte"`) {
		t.Errorf("the saved alternative text is not used:\n%s", body)
	}
}

// F10: every illustration is a radio button with a thumbnail, in gallery order.
func TestAdminShowsGallery(t *testing.T) {
	db := migrated(t)
	h := adminHandler(t, db, testPassword)
	c := sessionCookie(t, h)

	body := getAdmin(h, c).Body.String()
	last := -1
	for _, il := range content.Gallery {
		i := strings.Index(body, `name="banner" value="`+il.ID+`"`)
		if i < 0 {
			t.Fatalf("no radio for %s", il.ID)
		}
		if i < last {
			t.Errorf("%s is out of order", il.ID)
		}
		last = i
		if !strings.Contains(body, `src="/assets/banners/`+il.ID+".") {
			t.Errorf("no thumbnail for %s", il.ID)
		}
	}
	// The migration chose the first illustration.
	if !strings.Contains(body, `value="aube" checked`) {
		t.Error("the first illustration should be checked by default")
	}
	if strings.Contains(body, "n'existe plus") {
		t.Error("no warning expected for a known illustration")
	}
}

// CA22 (F16b): an illustration removed from the code gives a page without banner and
// a warning in the admin.
func TestRemovedIllustration(t *testing.T) {
	db := migrated(t)
	if _, err := db.Exec(context.Background(), `UPDATE content SET banner_choice = 'disparue'`); err != nil {
		t.Fatal(err)
	}
	h := adminHandler(t, db, testPassword)
	c := sessionCookie(t, h)

	rec := get(t, h, "/")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	body := rec.Body.String()
	if strings.Contains(body, `<div class="banner"><img`) || !strings.Contains(body, "banner-none") {
		t.Errorf("the page should have no banner:\n%s", body)
	}
	if got := meta(t, body, "og:image"); got != testBaseURL+"/og-default.png" {
		t.Errorf("og:image = %q, want the default one", got)
	}

	admin := getAdmin(h, c).Body.String()
	if !strings.Contains(admin, "L'illustration choisie n'existe plus") {
		t.Errorf("the admin shows no warning:\n%s", admin)
	}
}

// The Content-Type and cache of an unknown picture: 404, like any unknown asset.
func TestUnknownBannerPicture(t *testing.T) {
	h := publicHandler(t, defaultContent())
	for _, p := range []string{"/assets/banners/aube.0000000000.jpg", "/assets/banners/inconnue.jpg", "/assets/banners/"} {
		if rec := get(t, h, p); rec.Code != http.StatusNotFound {
			t.Errorf("%s: status %d, want 404", p, rec.Code)
		}
	}
}

func aubeForm(version string) url.Values {
	f := editForm(version)
	f.Set("banner", "aube")
	return f
}
