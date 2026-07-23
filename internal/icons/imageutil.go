package icons

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"strings"

	xdraw "golang.org/x/image/draw"
)

// DecodeImage loads PNG or JPEG from path.
func DecodeImage(path string) (image.Image, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return DecodeBytes(raw, path)
}

// DecodeBytes decodes PNG/JPEG bytes. name is for error context only.
func DecodeBytes(raw []byte, name string) (image.Image, error) {
	r := bytes.NewReader(raw)
	img, format, err := image.Decode(r)
	if err != nil {
		// try jpeg explicitly if extension hints
		ext := strings.ToLower(filepath.Ext(name))
		if ext == ".jpg" || ext == ".jpeg" {
			r.Seek(0, 0)
			return jpeg.Decode(r)
		}
		if ext == ".png" {
			r.Seek(0, 0)
			return png.Decode(r)
		}
		return nil, fmt.Errorf("decode %s: %w (supported: png, jpeg)", name, err)
	}
	_ = format
	return img, nil
}

// PadCenter returns a square image: master centered, transparent margins if needed.
// Side is max(width, height).
func PadCenter(src image.Image) *image.NRGBA {
	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	side := w
	if h > side {
		side = h
	}
	dst := image.NewNRGBA(image.Rect(0, 0, side, side))
	// transparent background (zero value)
	off := image.Pt((side-w)/2, (side-h)/2)
	r := image.Rect(off.X, off.Y, off.X+w, off.Y+h)
	draw.Draw(dst, r, src, b.Min, draw.Over)
	return dst
}

// Resize returns a size×size NRGBA using CatmullRom.
func Resize(src image.Image, size int) *image.NRGBA {
	dst := image.NewNRGBA(image.Rect(0, 0, size, size))
	xdraw.CatmullRom.Scale(dst, dst.Bounds(), src, src.Bounds(), draw.Over, nil)
	return dst
}

// EncodePNG encodes img as PNG bytes.
func EncodePNG(img image.Image) ([]byte, error) {
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// WritePNG writes a PNG file, creating parent dirs.
func WritePNG(path string, img image.Image) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, img)
}

// FlattenOpaque draws src onto white (for formats that dislike alpha).
func FlattenOpaque(src image.Image) *image.NRGBA {
	b := src.Bounds()
	dst := image.NewNRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	draw.Draw(dst, dst.Bounds(), &image.Uniform{C: color.White}, image.Point{}, draw.Src)
	draw.Draw(dst, dst.Bounds(), src, b.Min, draw.Over)
	return dst
}
