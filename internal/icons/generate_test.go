package icons

import (
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPadCenterSquare(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 100, 50))
	for y := 0; y < 50; y++ {
		for x := 0; x < 100; x++ {
			img.Set(x, y, color.NRGBA{R: 255, A: 255})
		}
	}
	sq := PadCenter(img)
	assert.Equal(t, 100, sq.Bounds().Dx())
	assert.Equal(t, 100, sq.Bounds().Dy())
}

func TestGenerateDefaultAndSkip(t *testing.T) {
	dir := t.TempDir()
	m1, err := Generate(Options{OutputDir: dir, Force: true})
	require.NoError(t, err)
	assert.Equal(t, "default", m1.Source)
	assert.True(t, Complete(dir))
	// second call without force should skip (manifest still readable)
	m2, err := Generate(Options{OutputDir: dir, Force: false})
	require.NoError(t, err)
	assert.Equal(t, m1.GeneratedAt, m2.GeneratedAt)
	// force rebuild
	m3, err := Generate(Options{OutputDir: dir, Force: true})
	require.NoError(t, err)
	if m3.GeneratedAt == m1.GeneratedAt {
		// possible if same second; ensure files still ok
		assert.True(t, Complete(dir))
	}
}

func TestGenerateFromPNG(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "in.png")
	img := image.NewNRGBA(image.Rect(0, 0, 64, 32))
	for y := 0; y < 32; y++ {
		for x := 0; x < 64; x++ {
			img.Set(x, y, color.NRGBA{B: 200, A: 255})
		}
	}
	f, err := os.Create(src)
	require.NoError(t, err)
	require.NoError(t, png.Encode(f, img))
	require.NoError(t, f.Close())

	out := filepath.Join(dir, "icons")
	m, err := Generate(Options{SourcePath: src, OutputDir: out, Force: true})
	require.NoError(t, err)
	assert.Equal(t, src, m.Source)
	_, err = os.Stat(filepath.Join(out, "windows", "icon.ico"))
	require.NoError(t, err)
	_, err = os.Stat(filepath.Join(out, "macos", "icon.icns"))
	require.NoError(t, err)
}

func TestApplyAndroidRes(t *testing.T) {
	iconsDir := t.TempDir()
	_, err := Generate(Options{OutputDir: iconsDir, Force: true})
	require.NoError(t, err)
	res := filepath.Join(t.TempDir(), "res")
	require.NoError(t, ApplyAndroidRes(iconsDir, res))
	_, err = os.Stat(filepath.Join(res, "mipmap-mdpi", "ic_launcher.png"))
	require.NoError(t, err)
}

func TestApplyMacOSICNS(t *testing.T) {
	iconsDir := t.TempDir()
	_, err := Generate(Options{OutputDir: iconsDir, Force: true})
	require.NoError(t, err)
	dest := filepath.Join(t.TempDir(), "Resources", "AppIcon.icns")
	require.NoError(t, ApplyMacOSICNS(iconsDir, dest))
	_, err = os.Stat(dest)
	require.NoError(t, err)
}

func TestDefaultLockupEmbed(t *testing.T) {
	assert.GreaterOrEqual(t, len(DefaultLockupPNG), 100)
}

func TestDefaultAssetsHaveAlpha(t *testing.T) {
	t.Parallel()
	lockup, err := DecodeBytes(DefaultLockupPNG, "lockup.png")
	require.NoError(t, err)
	b := lockup.Bounds()
	_, _, _, a0 := lockup.At(b.Min.X, b.Min.Y).RGBA()
	assert.Equal(t, uint32(0), a0)
	mark, err := defaultMaster()
	require.NoError(t, err)
	mb := mark.Bounds()
	assert.Equal(t, 1024, mb.Dx())
	assert.Equal(t, 1024, mb.Dy())
	_, _, _, ma := mark.At(mb.Min.X, mb.Min.Y).RGBA()
	assert.Equal(t, uint32(0), ma)
}

func TestExtractUpperMarkSplitsLockup(t *testing.T) {
	t.Parallel()
	img := image.NewNRGBA(image.Rect(0, 0, 40, 80))
	// top blob (mark)
	for y := 4; y < 20; y++ {
		for x := 8; x < 32; x++ {
			img.Set(x, y, color.NRGBA{B: 200, A: 255})
		}
	}
	// bottom blob (wordmark)
	for y := 50; y < 60; y++ {
		for x := 4; x < 36; x++ {
			img.Set(x, y, color.NRGBA{R: 40, A: 255})
		}
	}
	got := ExtractUpperMark(img)
	assert.Equal(t, got.Bounds().Dy(), got.Bounds().Dx())
	// bottom of the square must not contain the wordmark band
	_, _, _, a := got.At(got.Bounds().Min.X+got.Bounds().Dx()/2, got.Bounds().Max.Y-1).RGBA()
	assert.Equal(t, uint32(0), a)
	_, _, _, ac := got.At(got.Bounds().Dx()/2, got.Bounds().Dy()/2).RGBA()
	assert.GreaterOrEqual(t, ac, uint32(0x8000))
}

func TestContentBoundsAndTrim(t *testing.T) {
	t.Parallel()
	img := image.NewNRGBA(image.Rect(0, 0, 20, 20))
	for y := 5; y < 9; y++ {
		for x := 6; x < 11; x++ {
			img.Set(x, y, color.NRGBA{G: 200, A: 255})
		}
	}
	box := ContentBounds(img, 8)
	assert.Equal(t, image.Rect(6, 5, 11, 9), box)
	trim := TrimTransparent(img, 0)
	assert.Equal(t, 5, trim.Bounds().Dx())
	assert.Equal(t, 4, trim.Bounds().Dy())
}

func TestKnockoutKeepsInteriorHighlights(t *testing.T) {
	t.Parallel()
	img := image.NewNRGBA(image.Rect(0, 0, 32, 32))
	for y := range 32 {
		for x := range 32 {
			img.Set(x, y, color.NRGBA{R: 255, G: 255, B: 255, A: 255})
		}
	}
	// blue ring with a white highlight in the hole
	for y := 8; y < 24; y++ {
		for x := 8; x < 24; x++ {
			img.Set(x, y, color.NRGBA{B: 200, A: 255})
		}
	}
	for y := 13; y < 19; y++ {
		for x := 13; x < 19; x++ {
			img.Set(x, y, color.NRGBA{R: 255, G: 255, B: 255, A: 255})
		}
	}
	out := KnockoutBackground(img)
	_, _, _, a0 := out.At(0, 0).RGBA()
	assert.Equal(t, uint32(0), a0)
	_, _, _, ac := out.At(16, 16).RGBA()
	assert.GreaterOrEqual(t, ac, uint32(0x8000))
}

func TestKnockoutBackgroundLightCanvas(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 32, 32))
	// white canvas
	for y := 0; y < 32; y++ {
		for x := 0; x < 32; x++ {
			img.Set(x, y, color.NRGBA{R: 255, G: 255, B: 255, A: 255})
		}
	}
	// blue blob in center
	for y := 10; y < 22; y++ {
		for x := 10; x < 22; x++ {
			img.Set(x, y, color.NRGBA{B: 200, A: 255})
		}
	}
	out := KnockoutBackground(img)
	_, _, _, a0 := out.At(0, 0).RGBA()
	assert.Equal(t, uint32(0), a0)
	_, _, _, ac := out.At(16, 16).RGBA()
	assert.GreaterOrEqual(t, ac, uint32(0x8000))
}
