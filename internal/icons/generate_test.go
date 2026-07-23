package icons

import (
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

func TestPadCenterSquare(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 100, 50))
	for y := 0; y < 50; y++ {
		for x := 0; x < 100; x++ {
			img.Set(x, y, color.NRGBA{R: 255, A: 255})
		}
	}
	sq := PadCenter(img)
	if sq.Bounds().Dx() != 100 || sq.Bounds().Dy() != 100 {
		t.Fatalf("side got %v", sq.Bounds())
	}
}

func TestGenerateDefaultAndSkip(t *testing.T) {
	dir := t.TempDir()
	m1, err := Generate(Options{OutputDir: dir, Force: true})
	if err != nil {
		t.Fatal(err)
	}
	if m1.Source != "default" {
		t.Fatalf("source %q", m1.Source)
	}
	if !Complete(dir) {
		t.Fatal("expected complete tree")
	}
	// second call without force should skip (manifest still readable)
	m2, err := Generate(Options{OutputDir: dir, Force: false})
	if err != nil {
		t.Fatal(err)
	}
	if m2.GeneratedAt != m1.GeneratedAt {
		t.Fatalf("expected skip, times %s vs %s", m1.GeneratedAt, m2.GeneratedAt)
	}
	// force rebuild
	m3, err := Generate(Options{OutputDir: dir, Force: true})
	if err != nil {
		t.Fatal(err)
	}
	if m3.GeneratedAt == m1.GeneratedAt {
		// possible if same second; ensure files still ok
		if !Complete(dir) {
			t.Fatal("incomplete after force")
		}
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
	if err != nil {
		t.Fatal(err)
	}
	if err := png.Encode(f, img); err != nil {
		t.Fatal(err)
	}
	f.Close()

	out := filepath.Join(dir, "icons")
	m, err := Generate(Options{SourcePath: src, OutputDir: out, Force: true})
	if err != nil {
		t.Fatal(err)
	}
	if m.Source != src {
		t.Fatalf("source %q", m.Source)
	}
	if _, err := os.Stat(filepath.Join(out, "windows", "icon.ico")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(out, "macos", "icon.icns")); err != nil {
		t.Fatal(err)
	}
}

func TestApplyAndroidRes(t *testing.T) {
	iconsDir := t.TempDir()
	if _, err := Generate(Options{OutputDir: iconsDir, Force: true}); err != nil {
		t.Fatal(err)
	}
	res := filepath.Join(t.TempDir(), "res")
	if err := ApplyAndroidRes(iconsDir, res); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(res, "mipmap-mdpi", "ic_launcher.png")); err != nil {
		t.Fatal(err)
	}
}

func TestDefaultMarkEmbed(t *testing.T) {
	if len(DefaultMarkPNG) < 100 {
		t.Fatal("default mark embed empty")
	}
	if len(DefaultLockupPNG) < 100 {
		t.Fatal("default lockup embed empty")
	}
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
	if a0 != 0 {
		t.Fatalf("corner should be transparent, a=%d", a0>>8)
	}
	_, _, _, ac := out.At(16, 16).RGBA()
	if ac < 0x8000 {
		t.Fatalf("center should stay opaque, a=%d", ac>>8)
	}
}
