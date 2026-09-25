package apk

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
	require.NoError(t, os.WriteFile(path, []byte(body), 0o644))
	cfg, base, err := LoadConfig(dir)
	require.NoError(t, err)
	assert.Equal(t, dir, base)
	assert.Equal(t, "br.tec.lew.demo", cfg.PackageID)
	assert.Equal(t, "Demo", cfg.AppName)
}

func TestLoadConfig_Capabilities(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ConfigFileName)
	body := `{
  "schema_version": 1,
  "package_id": "br.tec.lew.demo",
  "app_name": "Demo",
  "go_main": ".",
  "capabilities": {
    "url": {"schemes": ["myapp"]},
    "files": {"types": [{"ext": ".md", "mime": "text/markdown"}]}
  }
}
`
	require.NoError(t, os.WriteFile(path, []byte(body), 0o644))
	cfg, _, err := LoadConfig(dir)
	require.NoError(t, err)
	require.NotNil(t, cfg.Capabilities.URL)
	require.NotEmpty(t, cfg.Capabilities.URL.Schemes)
	assert.Equal(t, "myapp", cfg.Capabilities.URL.Schemes[0])
}

func TestLoadConfig_UnknownCapability(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ConfigFileName)
	body := `{
  "schema_version": 1,
  "package_id": "br.tec.lew.demo",
  "app_name": "Demo",
  "capabilities": {"camera": {"usage": "x"}}
}
`
	require.NoError(t, os.WriteFile(path, []byte(body), 0o644))
	_, _, err := LoadConfig(dir)
	require.Error(t, err)
}

func TestMerge_FlagsWin(t *testing.T) {
	base := Config{PackageID: "a.b.c", AppName: "A", GoMain: "."}
	out := Merge(base, Config{AppName: "B", GoMain: "./cmd"})
	assert.Equal(t, "a.b.c", out.PackageID)
	assert.Equal(t, "B", out.AppName)
	assert.Equal(t, "./cmd", out.GoMain)
}

func TestBuild_GoOnly_Counter(t *testing.T) {
	// examples/counter is a sibling module with replace → ../..
	repoRoot, err := filepath.Abs("../..")
	require.NoError(t, err)
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
	require.NoError(t, err, buf.String())
	require.Len(t, res.JNILibs, 1)
	st, err := os.Stat(res.JNILibs[0])
	require.NoError(t, err)
	assert.GreaterOrEqual(t, st.Size(), int64(1000))
	assert.Contains(t, buf.String(), "arm64-v8a")
}

func TestDefaultOutAPK(t *testing.T) {
	p := DefaultOutAPK("br.tec.lew.counter", "/tmp/proj")
	assert.Equal(t, filepath.Join("/tmp/proj", "dist", "counter-debug.apk"), p)
}

func TestAndroidSDK_MissingMessage(t *testing.T) {
	t.Setenv("ANDROID_HOME", "")
	t.Setenv("ANDROID_SDK_ROOT", "")
	// May still find ~/Android/Sdk — only assert error shape when both empty and no default.
	_, err := androidSDK()
	if err == nil {
		t.Skip("SDK present on machine")
	}
	assert.True(t, errors.Is(err, ErrAndroidSDKNotFound) || errors.Is(err, ErrSDKEnvNotDir))
}

func TestCopyFile(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src.bin")
	dst := filepath.Join(dir, "dst.bin")
	want := []byte("apk-bytes")
	require.NoError(t, os.WriteFile(src, want, 0o644))
	require.NoError(t, copyFile(src, dst))
	got, err := os.ReadFile(dst)
	require.NoError(t, err)
	assert.Equal(t, want, got)
	require.Error(t, copyFile(filepath.Join(dir, "missing"), dst))
}
