package common

import (
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMaterializeHost_RendersAndWritesJSON(t *testing.T) {
	src := fstest.MapFS{
		"template/Info.plist.tmpl": {Data: []byte(`<name>{{.Name}}</name>`)},
	}
	out := filepath.Join(t.TempDir(), "host")
	err := MaterializeHost(src, struct{ Name string }{Name: "App"}, out, false, []byte("{\"ok\":true}\n"))
	require.NoError(t, err)
	plist, err := os.ReadFile(filepath.Join(out, "Info.plist"))
	require.NoError(t, err)
	assert.Equal(t, "<name>App</name>", string(plist))
	jsonb, err := os.ReadFile(filepath.Join(out, HostConfigFile))
	require.NoError(t, err)
	assert.Equal(t, "{\"ok\":true}\n", string(jsonb))
}

func TestMaterializeHost_EmptyOutDir(t *testing.T) {
	src := fstest.MapFS{
		"template/static.txt": {Data: []byte("x")},
	}
	err := MaterializeHost(src, nil, "  ", false, nil)
	require.ErrorIs(t, err, ErrOutDirRequired)
}

func TestWalkTemplate_RendersAndCopies(t *testing.T) {
	src := fstest.MapFS{
		"template/Info.plist.tmpl": {Data: []byte(`<name>{{.Name}}</name>`)},
		"template/static.txt":      {Data: []byte("keep")},
		"template/settings.kts":    {Data: []byte(`name = "{{.Name}}"`)},
		"template/scripts/run.sh":  {Data: []byte("#!/bin/sh\n")},
	}
	out := t.TempDir()
	require.NoError(t, WalkTemplate(src, struct{ Name string }{Name: "App"}, out))
	plist, err := os.ReadFile(filepath.Join(out, "Info.plist"))
	require.NoError(t, err)
	assert.Equal(t, "<name>App</name>", string(plist))
	static, err := os.ReadFile(filepath.Join(out, "static.txt"))
	require.NoError(t, err)
	assert.Equal(t, "keep", string(static))
	kts, err := os.ReadFile(filepath.Join(out, "settings.kts"))
	require.NoError(t, err)
	assert.Equal(t, `name = "App"`, string(kts))
	info, err := os.Stat(filepath.Join(out, "scripts/run.sh"))
	require.NoError(t, err)
	assert.True(t, info.Mode()&0o111 != 0)
}

func TestWalkTemplateDest_RemapsPath(t *testing.T) {
	src := fstest.MapFS{
		"template/app/src/main/kotlin/Main.kt.tmpl": {Data: []byte("package {{.Pkg}}\n")},
	}
	out := t.TempDir()
	err := WalkTemplateDest(src, struct{ Pkg string }{Pkg: "br.tec.lew.x"}, out, func(rel, destRel string) string {
		if rel == filepath.FromSlash("app/src/main/kotlin/Main.kt.tmpl") {
			return filepath.Join("app", "src", "main", "java", "br", "tec", "lew", "x", "Main.kt")
		}
		return destRel
	})
	require.NoError(t, err)
	got, err := os.ReadFile(filepath.Join(out, "app/src/main/java/br/tec/lew/x/Main.kt"))
	require.NoError(t, err)
	assert.Equal(t, "package br.tec.lew.x\n", string(got))
}
