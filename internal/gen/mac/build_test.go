package mac

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"runtime"
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
	if res.HelperPath == "" {
		t.Fatal("empty helper path")
	}
	st, err := os.Stat(res.HelperPath)
	if err != nil || st.Size() < 1000 {
		t.Fatalf("helper %s: %v size=%d", res.HelperPath, err, st.Size())
	}
	if _, err := os.Stat(filepath.Join(work, "project.yml")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(work, "Resources", "AppIcon.icns")); err != nil {
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
	if !errors.Is(err, ErrMacOSRequired) {
		t.Fatalf("got %v want ErrMacOSRequired", err)
	}
}
