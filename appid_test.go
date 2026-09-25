package eletrocromo

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateAppID(t *testing.T) {
	ok := []string{
		"br.tec.lew.counter",
		"com.example.app",
		"a.b",
		"org.foo_bar.baz",
	}
	for _, id := range ok {
		assert.NoError(t, ValidateAppID(id), id)
	}
	bad := []string{
		"",
		"counter",
		"Counter.app",
		"com.Example.app",
		"../evil",
		"com/example",
		"com.example/app",
		".com.example",
		"com.",
		"1com.example",
	}
	for _, id := range bad {
		assert.Error(t, ValidateAppID(id), id)
	}
}

func TestProfileDir_IsolatesByAppID(t *testing.T) {
	root := t.TempDir()
	t.Setenv("XDG_DATA_HOME", root)

	a, err := ProfileDir("br.tec.lew.counter")
	require.NoError(t, err)
	b, err := ProfileDir("br.tec.lew.basic")
	require.NoError(t, err)
	assert.NotEqual(t, a, b)
	assert.True(t, strings.HasPrefix(a, root), a)
	assert.True(t, strings.HasPrefix(b, root), b)
	assert.Equal(t, "br.tec.lew.counter", filepath.Base(a))
	st, err := os.Stat(a)
	require.NoError(t, err)
	assert.True(t, st.IsDir())
}

func TestUserDataDirFor_ByGOOS(t *testing.T) {
	env := map[string]string{}
	getenv := func(k string) string { return env[k] }

	got := userDataDirFor("linux", "/home/u", getenv)
	want := filepath.Join("/home/u", ".local", "share")
	assert.Equal(t, want, got)

	macish := filepath.Join("/home/u", "Library", "Application Support")
	assert.NotEqual(t, macish, userDataDirFor("linux", "/home/u", getenv))

	got = userDataDirFor("darwin", "/Users/u", getenv)
	want = filepath.Join("/Users/u", "Library", "Application Support")
	assert.Equal(t, want, got)

	env["LOCALAPPDATA"] = filepath.Join("C:", "Users", "u", "AppData", "Local")
	assert.Equal(t, env["LOCALAPPDATA"], userDataDirFor("windows", filepath.Join("C:", "Users", "u"), getenv))

	delete(env, "LOCALAPPDATA")
	want = filepath.Join("C:", "Users", "u", "AppData", "Local")
	assert.Equal(t, want, userDataDirFor("windows", filepath.Join("C:", "Users", "u"), getenv))

	want = filepath.Join("/home/u", ".local", "share")
	assert.Equal(t, want, userDataDirFor("freebsd", "/home/u", getenv))
}
