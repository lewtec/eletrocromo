package main

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/lewtec/lewkit/x/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildIOS_Help(t *testing.T) {
	var runErr error
	var out string
	errText := test.Stderr(t, func() {
		out = test.Stdout(t, func() {
			runErr = run(t.Context(), []string{"build", "ios", "--help"})
		})
	})
	require.NoError(t, runErr, out+errText)
	assert.Contains(t, out+errText, "--go-only")
	assert.Contains(t, out+errText, "--sdk")
}

func TestBuildIOS_GoOnly_Counter(t *testing.T) {
	repoRoot, err := filepath.Abs("../..")
	require.NoError(t, err)
	counter := filepath.Join(repoRoot, "examples", "counter", "eletrocromo.json")
	if _, err := os.Stat(counter); err != nil {
		t.Skip(err)
	}
	work := t.TempDir()
	iconsOut := filepath.Join(t.TempDir(), "icons")
	var runErr error
	var buf string
	errText := test.Stderr(t, func() {
		buf = test.Stdout(t, func() {
			runErr = run(t.Context(), []string{
				"build", "ios",
				"--config", counter,
				"--go-only",
				"--workdir", work,
				"--output", iconsOut,
			})
		})
	})
	require.NoError(t, runErr, buf+errText)
	_, err = os.Stat(filepath.Join(work, "project.yml"))
	require.NoError(t, err)
	if runtime.GOOS == "darwin" {
		_, err := os.Stat(filepath.Join(work, "lib", "libeletrocromo.a"))
		require.NoError(t, err)
		assert.Contains(t, buf+errText, "archive:")
	}
}
