package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lewtec/eletrocromo/internal/icons"
	"github.com/lewtec/lewkit/x/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRoot_HelpListsBuild(t *testing.T) {
	var runErr error
	var out string
	errText := test.Stderr(t, func() {
		out = test.Stdout(t, func() {
			runErr = run(t.Context(), []string{"--help"})
		})
	})
	require.NoError(t, runErr, out+errText)
	assert.Contains(t, out+errText, "build")
}

func TestBuild_BareErrors(t *testing.T) {
	var runErr error
	test.Stderr(t, func() {
		_ = test.Stdout(t, func() {
			runErr = run(t.Context(), []string{"build"})
		})
	})
	require.Error(t, runErr)
	require.ErrorIs(t, runErr, ErrMissingBuildTarget)
}

func TestBuildIcons_Default(t *testing.T) {
	dir := t.TempDir()
	var runErr error
	var out string
	errText := test.Stderr(t, func() {
		out = test.Stdout(t, func() {
			runErr = run(t.Context(), []string{"build", "icons", "--output", filepath.Join(dir, "icons"), "--refresh-icons"})
		})
	})
	require.NoError(t, runErr, out+errText)
	_, err := os.Stat(filepath.Join(dir, "icons", "manifest.json"))
	require.NoError(t, err)
}

func TestDefaultConfigPath(t *testing.T) {
	dir := t.TempDir()
	assert.Equal(t, "", defaultConfigPath(dir))
	path := filepath.Join(dir, "eletrocromo.json")
	require.NoError(t, os.WriteFile(path, []byte(`{}`), 0o644))
	assert.Equal(t, path, defaultConfigPath(dir))
}

func TestResolveIconSource(t *testing.T) {
	cwd := t.TempDir()
	base := filepath.Join(cwd, "cfg")
	require.NoError(t, os.MkdirAll(base, 0o755))
	// flag wins over config
	got := resolveIconSource(cwd, "flag.png", base, "cfg.png")
	want := filepath.Join(cwd, "flag.png")
	assert.Equal(t, want, got)
	// config only, relative to baseDir
	got = resolveIconSource(cwd, "", base, "cfg.png")
	want = filepath.Join(base, "cfg.png")
	assert.Equal(t, want, got)
	// empty → default mark
	assert.Equal(t, "", resolveIconSource(cwd, "", base, ""))
	// absolute flag preserved
	abs := filepath.Join(cwd, "abs.png")
	assert.Equal(t, abs, resolveIconSource(cwd, abs, base, "cfg.png"))
}

func TestEnsureBuildIcons(t *testing.T) {
	out := filepath.Join(t.TempDir(), "icons")
	var buf bytes.Buffer
	root, err := ensureBuildIcons(&buf, "", out, false)
	require.NoError(t, err)
	assert.True(t, icons.Complete(root), "generated tree incomplete: %s", root)
	assert.Contains(t, buf.String(), "icons →")

	buf.Reset()
	again, err := ensureBuildIcons(&buf, "", out, false)
	require.NoError(t, err)
	assert.Equal(t, out, again)
	assert.Contains(t, buf.String(), "already present")

	buf.Reset()
	_, err = ensureBuildIcons(&buf, "", out, true)
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "icons →")
}

func TestRunIconsThen(t *testing.T) {
	out := filepath.Join(t.TempDir(), "icons")
	var got string
	var runErr error
	log := test.Stdout(t, func() {
		runErr = runIconsThen(t.Context(), iconThen{src: "", out: out, refresh: false, name: "work"}, func(iconRoot string) error {
			got = iconRoot
			return nil
		})
	})
	require.NoError(t, runErr)
	assert.True(t, icons.Complete(got), "work ran without a complete tree: %s", got)
	assert.Contains(t, log, "icons →")
}

func TestResolveIconOutput(t *testing.T) {
	cwd := t.TempDir()
	assert.Equal(t, filepath.Join(cwd, "dist/icons"), resolveIconOutput(cwd, ""))
	assert.Equal(t, filepath.Join(cwd, "out"), resolveIconOutput(cwd, "out"))
	abs := filepath.Join(cwd, "abs-out")
	assert.Equal(t, abs, resolveIconOutput(cwd, abs))
}

func TestVersionCmd(t *testing.T) {
	var runErr error
	var out string
	errText := test.Stderr(t, func() {
		out = test.Stdout(t, func() {
			runErr = run(t.Context(), []string{"version"})
		})
	})
	require.NoError(t, runErr, out+errText)
	require.NotEmpty(t, strings.TrimSpace(out+errText))
}

func TestAndroidCreate_RequiredFlags(t *testing.T) {
	var runErr error
	test.Stderr(t, func() {
		_ = test.Stdout(t, func() {
			runErr = run(t.Context(), []string{"android", "create"})
		})
	})
	require.Error(t, runErr)
}

func TestAndroidCreate_WritesProject(t *testing.T) {
	outDir := t.TempDir()
	// Cobra reuses process; run into empty subdir.
	dest := filepath.Join(outDir, "proj")

	var runErr error
	var buf string
	errText := test.Stderr(t, func() {
		buf = test.Stdout(t, func() {
			runErr = run(t.Context(), []string{
				"android", "create",
				"--id", "br.tec.lew.cli_test",
				"--name", "CLITest",
				"--out", dest,
				"--go-main", ".",
			})
		})
	})
	require.NoError(t, runErr, buf+errText)
	_, err := os.Stat(filepath.Join(dest, "eletrocromo.json"))
	require.NoError(t, err)
	assert.Contains(t, buf+errText, "br.tec.lew.cli_test")
}
