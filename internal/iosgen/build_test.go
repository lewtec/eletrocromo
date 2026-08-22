package iosgen

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestBuild_GoOnly_Counter(t *testing.T) {
	repoRoot, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	counterDir := filepath.Join(repoRoot, "examples", "counter")
	if _, err := os.Stat(filepath.Join(counterDir, "main.go")); err != nil {
		t.Skip("examples/counter not present")
	}

	work := t.TempDir()
	var buf bytes.Buffer
	res, err := Build(BuildOptions{
		Config: Config{
			PackageID: "br.tec.lew.eletrocromo.counter",
			AppName:   "Counter",
			GoMain:    ".",
		},
		BaseDir:     counterDir,
		WorkDir:     work,
		KeepWorkDir: true,
		GoOnly:      true,
		Stdout:      &buf,
		Stderr:      &buf,
	})
	if err != nil {
		t.Fatalf("%v\n%s", err, buf.String())
	}
	if _, err := os.Stat(filepath.Join(work, "project.yml")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(work, "Assets.xcassets", "AppIcon.appiconset", "AppIcon.png")); err != nil {
		t.Fatal(err)
	}
	logoDir := filepath.Join(work, "Assets.xcassets", "SplashLogo.imageset")
	for _, name := range []string{"SplashLogo.png", "SplashLogo@2x.png", "SplashLogo@3x.png"} {
		if _, err := os.Stat(filepath.Join(logoDir, name)); err != nil {
			t.Fatal(err)
		}
	}
	if runtime.GOOS != "darwin" {
		if res.ArchivePath != "" {
			t.Fatalf("expected empty archive path off darwin, got %s", res.ArchivePath)
		}
		return
	}
	if res.ArchivePath == "" {
		t.Fatal("empty archive path")
	}
	st, err := os.Stat(res.ArchivePath)
	if err != nil || st.Size() < 1000 {
		t.Fatalf("archive %s: %v size=%d", res.ArchivePath, err, st.Size())
	}
	if _, err := os.Stat(strings.TrimSuffix(res.ArchivePath, ".a") + ".h"); err != nil {
		t.Fatal(err)
	}
}

func TestBuild_FullRequiresDarwin(t *testing.T) {
	if runtime.GOOS == "darwin" {
		t.Skip("on darwin, full build is allowed")
	}
	_, err := Build(BuildOptions{
		Config:  Config{PackageID: "br.tec.lew.x", AppName: "X", GoMain: "."},
		BaseDir: t.TempDir(),
		OutApp:  filepath.Join(t.TempDir(), "X.app"),
	})
	if !errors.Is(err, ErrDarwinRequired) {
		t.Fatalf("got %v want ErrDarwinRequired", err)
	}
}
