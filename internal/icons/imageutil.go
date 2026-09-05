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

// KnockoutBackground makes pixels near the corner sample color transparent
// (soft edge). Used so launcher/splash marks are not sitting on a white box.
// Corners of photo-style logos are usually the canvas; solid brand marks are
// left alone if the corner is not near-white/near-uniform.
func KnockoutBackground(src image.Image) *image.NRGBA {
	b := src.Bounds()
	dst := image.NewNRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	// Sample a few corners; use average if similar.
	samples := []image.Point{
		{b.Min.X, b.Min.Y},
		{b.Max.X - 1, b.Min.Y},
		{b.Min.X, b.Max.Y - 1},
		{b.Max.X - 1, b.Max.Y - 1},
	}
	var sr, sg, sb, n int
	for _, p := range samples {
		r, g, bl, a := src.At(p.X, p.Y).RGBA()
		if a < 0x8000 {
			continue // already transparent corner
		}
		sr += int(r >> 8)
		sg += int(g >> 8)
		sb += int(bl >> 8)
		n++
	}
	if n == 0 {
		draw.Draw(dst, dst.Bounds(), src, b.Min, draw.Src)
		return dst
	}
	cr, cg, cb := sr/n, sg/n, sb/n
	// Only knock out light canvases (avoid eating dark logos).
	if cr+cg+cb < 200*3 {
		draw.Draw(dst, dst.Bounds(), src, b.Min, draw.Src)
		return dst
	}

	const hard, soft = 28.0, 58.0
	hs, ss := hard*hard, soft*soft
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			r16, g16, b16, a16 := src.At(x, y).RGBA()
			r, g, bl, a := int(r16>>8), int(g16>>8), int(b16>>8), int(a16>>8)
			dr, dg, db := float64(r-cr), float64(g-cg), float64(bl-cb)
			d := dr*dr + dg*dg + db*db
			var alpha uint8
			switch {
			case d <= hs:
				alpha = 0
			case d >= ss:
				alpha = uint8(a)
			default:
				t := (d - hs) / (ss - hs)
				alpha = uint8(float64(a) * t)
			}
			if alpha == 0 {
				dst.SetNRGBA(x-b.Min.X, y-b.Min.Y, color.NRGBA{})
			} else {
				dst.SetNRGBA(x-b.Min.X, y-b.Min.Y, color.NRGBA{R: uint8(r), G: uint8(g), B: uint8(bl), A: alpha})
			}
		}
	}
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
