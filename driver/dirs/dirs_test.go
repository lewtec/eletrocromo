package dirs_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lewtec/eletrocromo/driver/dirs"
	_ "github.com/lewtec/eletrocromo/driver/dirs/os"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolve_InboxUnderCache(t *testing.T) {
	root := t.TempDir()
	t.Setenv("ELETROCROMO_DATA_DIR", filepath.Join(root, "data"))
	t.Setenv("ELETROCROMO_CACHE_DIR", filepath.Join(root, "cache"))
	t.Setenv("ELETROCROMO_CONFIG_DIR", filepath.Join(root, "config"))

	got, err := dirs.Resolve(t.Context(), "br.tec.lew.counter")
	require.NoError(t, err)
	wantInbox := filepath.Join(root, "cache", "inbox")
	assert.Equal(t, wantInbox, got.Inbox)
	assert.True(t, strings.HasPrefix(got.Inbox, got.Cache+string(filepath.Separator)) || got.Inbox == filepath.Join(got.Cache, "inbox"))
	for _, dir := range []string{got.Data, got.Cache, got.Config, got.Inbox} {
		st, err := os.Stat(dir)
		require.NoError(t, err)
		assert.True(t, st.IsDir())
	}
}

func TestResolve_RejectsBadAppID(t *testing.T) {
	_, err := dirs.Resolve(t.Context(), "../evil")
	require.ErrorIs(t, err, dirs.ErrInvalidAppID)
}
