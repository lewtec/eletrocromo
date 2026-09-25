package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/lewtec/lewkit/x/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildMacOS_Help(t *testing.T) {
	var runErr error
	var out string
	errText := test.Stderr(t, func() {
		out = test.Stdout(t, func() {
			runErr = run(t.Context(), []string{"build", "--help"})
		})
	})
	require.NoError(t, runErr, out+errText)
	assert.Contains(t, out+errText, "--go-only")
}

func TestBuildMacOS_GoOnly_Counter(t *testing.T) {
	repoRoot, err := filepath.Abs("../..")
	require.NoError(t, err)
	// cmd tests run from cmd/eletrocromo
	counter := filepath.Join(repoRoot, "examples", "counter", "eletrocromo.json")
	if _, err := os.Stat(counter); err != nil {
		t.Skip(err)
	}
	t.Setenv("GOOS", "darwin")
	work := t.TempDir()
	iconsOut := filepath.Join(t.TempDir(), "icons")
	var runErr error
	var buf string
	errText := test.Stderr(t, func() {
		buf = test.Stdout(t, func() {
			runErr = run(t.Context(), []string{
				"build", counter,
				"--go-only",
				"--workdir", work,
				"--output", iconsOut,
			})
		})
	})
	require.NoError(t, runErr, buf+errText)
	_, err = os.Stat(filepath.Join(work, "bin", "eletrocromo-server"))
	require.NoError(t, err)
	assert.Contains(t, buf+errText, "helper:")
}
