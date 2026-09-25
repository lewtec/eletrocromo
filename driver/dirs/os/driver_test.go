package os

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolve_EnvWins(t *testing.T) {
	got, err := paths(
		"br.tec.lew.counter",
		"linux",
		func(k string) string {
			switch k {
			case envData:
				return "/data"
			case envCache:
				return "/cache"
			case envConfig:
				return "/config"
			default:
				return ""
			}
		},
		func() (string, error) { return "/home/u", nil },
		func() (string, error) { return "/unused-cache", nil },
		func() (string, error) { return "/unused-config", nil },
	)
	require.NoError(t, err)
	assert.Equal(t, "/data", got.Data)
	assert.Equal(t, "/cache", got.Cache)
	assert.Equal(t, "/config", got.Config)
	assert.Equal(t, filepath.Join("/cache", "inbox"), got.Inbox)
}

func TestDataHome_ByGOOS(t *testing.T) {
	env := map[string]string{}
	getenv := func(k string) string { return env[k] }
	home := func() (string, error) { return "/home/u", nil }

	got, err := dataHome("linux", getenv, home)
	require.NoError(t, err)
	assert.Equal(t, filepath.Join("/home/u", ".local", "share"), got)

	got, err = dataHome("darwin", getenv, home)
	require.NoError(t, err)
	assert.Equal(t, filepath.Join("/home/u", "Library", "Application Support"), got)

	env["LOCALAPPDATA"] = filepath.Join("C:", "Users", "u", "AppData", "Local")
	got, err = dataHome("windows", getenv, home)
	require.NoError(t, err)
	assert.Equal(t, env["LOCALAPPDATA"], got)
}
