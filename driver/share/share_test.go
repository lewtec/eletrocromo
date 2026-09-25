package share

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAppendAndParse(t *testing.T) {
	dir := t.TempDir()
	path := FilePath(dir)
	item := Item{Text: "hello", URL: "https://example.com"}
	require.NoError(t, AppendFile(path, item))
	raw, err := os.ReadFile(path)
	require.NoError(t, err)
	got, err := ParseLine(raw)
	require.NoError(t, err)
	assert.Equal(t, "hello", got.Text)
	assert.Equal(t, "https://example.com", got.URL)
}

func TestValidate_Empty(t *testing.T) {
	require.Error(t, (Item{}).validate())
}

func TestValidate_AbsPath(t *testing.T) {
	require.Error(t, (Item{Paths: []string{"rel.txt"}}).validate())
	p := filepath.Join(t.TempDir(), "a.txt")
	require.NoError(t, os.WriteFile(p, []byte("x"), 0o600))
	require.NoError(t, (Item{Paths: []string{p}}).validate())
}
