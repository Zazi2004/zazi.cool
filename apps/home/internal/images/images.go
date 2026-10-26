// Package images turns an uploaded file into a small, clean image: it checks the
// format and the dimensions, resizes, and re-encodes, which drops all metadata
// (EXIF, GPS position…). It does no I/O besides reading its input.
// Copied and simplified from apps/linktree/internal/images/images.go: a security
// fix in one copy must be carried over to the other.
package images

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"

	"golang.org/x/image/draw"
)

const (
	// MaxWidth is the widest a stored banner gets (F14): the page frame is never
	// wider than 1600 px on a high-density screen.
	MaxWidth = 1600
	// MaxInputSide bounds each side of the input (F14): decoding a huge image would
	// use a lot of memory (about 64 MiB at 4000 x 4000).
	MaxInputSide = 4000

	jpegQuality = 85
)

var (
	// ErrFormat means the file is not a JPEG or PNG image.
	ErrFormat = errors.New("unsupported image format")
	// ErrTooLarge means the image dimensions exceed MaxInputSide.
	ErrTooLarge = errors.New("image too large")
)

// Image is a processed image, ready to be stored.
type Image struct {
	Data        []byte
	ContentType string // image/jpeg or image/png
	Width       int
	Height      int
}

// Process validates, resizes and re-encodes an uploaded image.
func Process(data []byte) (Image, error) {
	// DecodeConfig only reads the header: dimensions are checked before the whole
	// image is decoded, which protects memory against "decompression bombs".
	// Only the jpeg and png decoders are imported by this package, so any other
	// format (SVG, GIF, WebP…) is reported as unknown.
	cfg, format, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil || (format != "jpeg" && format != "png") {
		return Image{}, ErrFormat
	}
	if cfg.Width <= 0 || cfg.Height <= 0 {
		return Image{}, ErrFormat
	}
	if cfg.Width > MaxInputSide || cfg.Height > MaxInputSide {
		return Image{}, ErrTooLarge
	}

	src, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return Image{}, ErrFormat
	}
	img := resize(src)

	var buf bytes.Buffer
	out := Image{Width: img.Bounds().Dx(), Height: img.Bounds().Dy()}
	if isOpaque(img) {
		out.ContentType = "image/jpeg"
		err = jpeg.Encode(&buf, img, &jpeg.Options{Quality: jpegQuality})
	} else {
		// JPEG has no transparency: keep PNG for images that use it.
		out.ContentType = "image/png"
		err = (&png.Encoder{CompressionLevel: png.BestCompression}).Encode(&buf, img)
	}
	if err != nil {
		return Image{}, fmt.Errorf("encode image: %w", err)
	}
	out.Data = buf.Bytes()
	return out, nil
}

// resize shrinks img to MaxWidth pixels wide, keeping its aspect ratio. Narrower
// images are returned as they are.
func resize(img image.Image) image.Image {
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	if w <= MaxWidth {
		return img
	}
	h = max(1, h*MaxWidth/w)
	dst := image.NewNRGBA(image.Rect(0, 0, MaxWidth, h))
	draw.CatmullRom.Scale(dst, dst.Bounds(), img, b, draw.Src, nil)
	return dst
}

// isOpaque reports whether no pixel of img is transparent. Most image types
// implement Opaque; the others are treated as transparent, which is safe (PNG).
func isOpaque(img image.Image) bool {
	o, ok := img.(interface{ Opaque() bool })
	return ok && o.Opaque()
}
