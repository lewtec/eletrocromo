package common

import (
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCopyDir_PreservesTree(t *testing.T) {
	src := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(src, "nested"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(src, "nested", "a.txt"), []byte("hello"), 0o644))

	dst := filepath.Join(t.TempDir(), "out")
	require.NoError(t, CopyDir(src, dst))
	got, err := os.ReadFile(filepath.Join(dst, "nested", "a.txt"))
	require.NoError(t, err)
	assert.Equal(t, "hello", string(got))
}

func TestReplaceDir_MovesOntoExisting(t *testing.T) {
	root := t.TempDir()
	src := filepath.Join(root, "src")
	dst := filepath.Join(root, "dst")
	require.NoError(t, os.MkdirAll(src, 0o755))
	require.NoError(t, os.MkdirAll(dst, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(src, "new.txt"), []byte("new"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dst, "old.txt"), []byte("old"), 0o644))

	require.NoError(t, ReplaceDir(src, dst))
	_, err := os.Stat(filepath.Join(dst, "old.txt"))
	require.ErrorIs(t, err, fs.ErrNotExist)
	got, err := os.ReadFile(filepath.Join(dst, "new.txt"))
	require.NoError(t, err)
	assert.Equal(t, "new", string(got))
}
