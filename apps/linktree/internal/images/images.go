// Package images turns an uploaded file into a small, clean image: it checks the
// format and the dimensions, resizes, and re-encodes, which drops all metadata
// (EXIF, GPS position…). It does no I/O besides reading its input.
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
	// MaxSide is the largest side of a stored image: enough for a 128 px avatar on a
	// high-density screen.
	MaxSide = 512
	// MaxInputSide bounds the input: the admin client already shrinks images to
	// 1024 px, and decoding a huge image would use a lot of memory.
	MaxInputSide = 2048

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

// resize shrinks img so that its largest side is at most MaxSide, keeping its
// aspect ratio. Smaller images are returned as they are.
func resize(img image.Image) image.Image {
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	if w <= MaxSide && h <= MaxSide {
		return img
	}
	if w >= h {
		h = max(1, h*MaxSide/w)
		w = MaxSide
	} else {
		w = max(1, w*MaxSide/h)
		h = MaxSide
	}
	dst := image.NewNRGBA(image.Rect(0, 0, w, h))
	draw.CatmullRom.Scale(dst, dst.Bounds(), img, b, draw.Src, nil)
	return dst
}

// isOpaque reports whether no pixel of img is transparent. Most image types
// implement Opaque; the others are treated as transparent, which is safe (PNG).
func isOpaque(img image.Image) bool {
	o, ok := img.(interface{ Opaque() bool })
	return ok && o.Opaque()
}
