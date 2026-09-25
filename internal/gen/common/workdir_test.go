package common

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveWorkDir_CreatesMissingPath(t *testing.T) {
	want := filepath.Join(t.TempDir(), "nested", "work")
	got, cleanup, err := ResolveWorkDir(want, "eletrocromo-test-*")
	require.NoError(t, err)
	assert.False(t, cleanup)
	abs, err := filepath.Abs(want)
	require.NoError(t, err)
	assert.Equal(t, abs, got)
	st, err := os.Stat(got)
	require.NoError(t, err)
	assert.True(t, st.IsDir())
}

func TestResolveWorkDir_EmptyMakesTemp(t *testing.T) {
	dir, cleanup, err := ResolveWorkDir("", "eletrocromo-test-*")
	require.NoError(t, err)
	assert.True(t, cleanup)
	t.Cleanup(func() {
		if err := os.RemoveAll(dir); err != nil {
			t.Logf("cleanup temp work dir: %v", err)
		}
	})
	assert.Contains(t, filepath.Base(dir), "eletrocromo-test-")
	st, err := os.Stat(dir)
	require.NoError(t, err)
	assert.True(t, st.IsDir())
}
