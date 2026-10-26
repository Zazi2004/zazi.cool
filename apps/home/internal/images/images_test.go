package images

import (
	"bytes"
	"errors"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"testing"
)

func encodePNG(t *testing.T, img image.Image) []byte {
	t.Helper()
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func encodeJPEG(t *testing.T, img image.Image) []byte {
	t.Helper()
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, nil); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func filled(w, h int, c color.Color) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := range h {
		for x := range w {
			img.Set(x, y, c)
		}
	}
	return img
}

func TestProcessResizesAndKeepsRatio(t *testing.T) {
	out, err := Process(encodeJPEG(t, filled(3000, 1000, color.White)))
	if err != nil {
		t.Fatal(err)
	}
	if out.Width != 1600 || out.Height != 533 || out.ContentType != "image/jpeg" {
		t.Errorf("got %dx%d %s, want 1600x533 image/jpeg", out.Width, out.Height, out.ContentType)
	}
	if _, format, err := image.DecodeConfig(bytes.NewReader(out.Data)); err != nil || format != "jpeg" {
		t.Errorf("output is not a valid JPEG: %v %s", err, format)
	}
}

func TestProcessKeepsSmallImages(t *testing.T) {
	out, err := Process(encodePNG(t, filled(100, 20, color.Black)))
	if err != nil {
		t.Fatal(err)
	}
	// Opaque PNG becomes a JPEG: much smaller for photos.
	if out.Width != 100 || out.Height != 20 || out.ContentType != "image/jpeg" {
		t.Errorf("got %dx%d %s", out.Width, out.Height, out.ContentType)
	}
}

func TestProcessKeepsTransparency(t *testing.T) {
	img := filled(2000, 100, color.White)
	img.Set(10, 10, color.NRGBA{A: 0})
	out, err := Process(encodePNG(t, img))
	if err != nil {
		t.Fatal(err)
	}
	if out.ContentType != "image/png" || out.Width != 1600 {
		t.Errorf("got %dx%d %s, want a 1600 px PNG", out.Width, out.Height, out.ContentType)
	}
}

func TestProcessRejectsOtherFormats(t *testing.T) {
	inputs := map[string][]byte{
		"svg":           []byte(`<svg xmlns="http://www.w3.org/2000/svg"><script>alert(1)</script></svg>`),
		"gif":           []byte("GIF89a\x01\x00\x01\x00\x00\x00\x00;"),
		"text":          []byte("hello"),
		"empty":         nil,
		"truncated png": encodePNG(t, filled(10, 10, color.White))[:40],
	}
	for name, data := range inputs {
		if _, err := Process(data); !errors.Is(err, ErrFormat) {
			t.Errorf("%s: err = %v, want ErrFormat", name, err)
		}
	}
}

func TestProcessRejectsHugeImagesBeforeDecoding(t *testing.T) {
	// A 1-pixel-high image is tiny to encode, but its width is over the limit.
	if _, err := Process(encodePNG(t, filled(MaxInputSide+1, 1, color.White))); !errors.Is(err, ErrTooLarge) {
		t.Errorf("err = %v, want ErrTooLarge", err)
	}
}

func TestProcessDropsMetadata(t *testing.T) {
	data := encodeJPEG(t, filled(50, 50, color.White))
	// Insert an EXIF segment (APP1) right after the SOI marker, as cameras do.
	payload := []byte("Exif\x00\x00GPS-SECRET")
	size := len(payload) + 2 // the segment length counts its own two bytes
	exif := append([]byte{0xFF, 0xE1, byte(size >> 8), byte(size)}, payload...)
	withExif := append(append(append([]byte{}, data[:2]...), exif...), data[2:]...)
	if !bytes.Contains(withExif, []byte("GPS-SECRET")) {
		t.Fatal("test setup: EXIF not inserted")
	}

	out, err := Process(withExif)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(out.Data, []byte("Exif")) || bytes.Contains(out.Data, []byte("GPS-SECRET")) {
		t.Error("metadata survived re-encoding")
	}
}

func TestProcessAcceptsTheLimit(t *testing.T) {
	out, err := Process(encodePNG(t, filled(MaxInputSide, 1, color.White)))
	if err != nil {
		t.Fatal(err)
	}
	if out.Width != MaxWidth {
		t.Errorf("width = %d, want %d", out.Width, MaxWidth)
	}
}

func TestProcessRejectsTallImages(t *testing.T) {
	if _, err := Process(encodePNG(t, filled(1, MaxInputSide+1, color.White))); !errors.Is(err, ErrTooLarge) {
		t.Errorf("err = %v, want ErrTooLarge", err)
	}
}
