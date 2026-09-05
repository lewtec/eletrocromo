package apkgen

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ConfigFileName)
	body := `{
  "schema_version": 1,
  "package_id": "br.tec.lew.demo",
  "app_name": "Demo",
  "go_main": "."
}
`
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, base, err := LoadConfig(dir)
	if err != nil {
		t.Fatal(err)
	}
	if base != dir {
		t.Fatalf("base=%q", base)
	}
	if cfg.PackageID != "br.tec.lew.demo" || cfg.AppName != "Demo" {
		t.Fatalf("%+v", cfg)
	}
}

func TestMerge_FlagsWin(t *testing.T) {
	base := Config{PackageID: "a.b.c", AppName: "A", GoMain: "."}
	out := Merge(base, Config{AppName: "B", GoMain: "./cmd"})
	if out.PackageID != "a.b.c" || out.AppName != "B" || out.GoMain != "./cmd" {
		t.Fatalf("%+v", out)
	}
}

func TestBuild_GoOnly_Counter(t *testing.T) {
	// examples/counter is a sibling module with replace → ../..
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
	// Single ABI for speed.
	res, err := Build(BuildOptions{
		Config: Config{
			PackageID: "br.tec.lew.eletrocromo.counter",
			AppName:   "Counter",
			GoMain:    ".",
			ABIs:      []string{"arm64-v8a"},
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
	if len(res.JNILibs) != 1 {
		t.Fatalf("libs=%v", res.JNILibs)
	}
	st, err := os.Stat(res.JNILibs[0])
	if err != nil || st.Size() < 1000 {
		t.Fatalf("lib missing or tiny: %v size=%d", err, st.Size())
	}
	if !strings.Contains(buf.String(), "arm64-v8a") {
		t.Fatalf("log: %s", buf.String())
	}
}

func TestDefaultOutAPK(t *testing.T) {
	p := DefaultOutAPK("br.tec.lew.counter", "/tmp/proj")
	if p != filepath.Join("/tmp/proj", "dist", "counter-debug.apk") {
		t.Fatal(p)
	}
}

func TestAndroidSDK_MissingMessage(t *testing.T) {
	t.Setenv("ANDROID_HOME", "")
	t.Setenv("ANDROID_SDK_ROOT", "")
	// May still find ~/Android/Sdk — only assert error shape when both empty and no default.
	_, err := androidSDK()
	if err == nil {
		t.Skip("SDK present on machine")
	}
	if !strings.Contains(err.Error(), "ANDROID_HOME") {
		t.Fatal(err)
	}
}
