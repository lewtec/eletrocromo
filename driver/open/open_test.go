package open

import (
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestToken(t *testing.T) {
	t.Parallel()
	tests := []struct {
		in   string
		ok   bool
		kind Kind
		url  string
		path string
	}{
		{in: "myapp://item/1", ok: true, kind: KindURL, url: "myapp://item/1"},
		{in: "myapp:opaque", ok: true, kind: KindURL, url: "myapp:opaque"},
		{in: "eletrocromo-inbox://", ok: true, kind: KindURL, url: "eletrocromo-inbox://"},
		{in: "eletrocromo-inbox://x/y?q=1&empty=&sp=a+b#frag", ok: true, kind: KindURL, url: "eletrocromo-inbox://x/y?q=1&empty=&sp=a+b#frag"},
		{in: "mailto:x@y.z", ok: true, kind: KindURL, url: "mailto:x@y.z"},
		{in: "/tmp/a.pdf", ok: true, kind: KindFiles, path: "/tmp/a.pdf"},
		{in: `C:\windows\a.txt`, ok: true, kind: KindFiles, path: `C:\windows\a.txt`},
		{in: "rel/path", ok: true, kind: KindFiles, path: "rel/path"},
		{in: "plainword", ok: false},
		{in: "", ok: false},
	}
	for _, tt := range tests {
		t.Run(strconv.Quote(tt.in), func(t *testing.T) {
			t.Parallel()
			got, ok := Token(tt.in)
			require.Equal(t, tt.ok, ok)
			if !ok {
				return
			}
			assert.Equal(t, tt.kind, got.Kind)
			if tt.kind == KindURL {
				assert.Equal(t, tt.url, got.URL)
			}
			if tt.kind == KindFiles {
				require.Len(t, got.Paths, 1)
				assert.Equal(t, tt.path, got.Paths[0])
			}
		})
	}
}

func TestCollect_EnvAndArgv(t *testing.T) {
	got := Collect([]string{"-test.v", "myapp://item/1", "-flag"}, " /tmp/a.pdf ")
	require.Len(t, got, 2)
	assert.Equal(t, KindURL, got[0].Kind)
	assert.Equal(t, "myapp://item/1", got[0].URL)
	assert.Equal(t, KindFiles, got[1].Kind)
	require.Len(t, got[1].Paths, 1)
	assert.Equal(t, "/tmp/a.pdf", got[1].Paths[0])
}

func TestParseLine(t *testing.T) {
	ev, err := ParseLine([]byte(`{"kind":"url","url":"myapp://x"}`))
	require.NoError(t, err)
	assert.Equal(t, KindURL, ev.Kind)
	assert.Equal(t, "myapp://x", ev.URL)
	ev, err = ParseLine([]byte(`{"kind":"files","paths":["/a"]}`))
	require.NoError(t, err)
	assert.Equal(t, KindFiles, ev.Kind)
	require.Len(t, ev.Paths, 1)
	assert.Equal(t, "/a", ev.Paths[0])
}

func TestTailFile_ReadsAppend(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, FileName)
	ctx := t.Context()

	got := make(chan Event, 2)
	errCh := make(chan error, 1)
	go func() {
		errCh <- TailFile(ctx, path, func(ev Event) error {
			got <- ev
			return nil
		})
	}()

	require.NoError(t, os.WriteFile(path, []byte("{\"kind\":\"url\",\"url\":\"myapp://one\"}\n"), 0o600))
	select {
	case ev := <-got:
		assert.Equal(t, "myapp://one", ev.URL)
	case err := <-errCh:
		require.NoError(t, err)
	case <-time.After(2 * time.Second):
		require.Fail(t, "timeout first line")
	}

	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0o600)
	require.NoError(t, err)
	_, err = f.WriteString("{\"kind\":\"files\",\"paths\":[\"/b\"]}\n")
	require.NoError(t, err)
	require.NoError(t, f.Close())
	select {
	case ev := <-got:
		assert.Equal(t, KindFiles, ev.Kind)
		require.Len(t, ev.Paths, 1)
		assert.Equal(t, "/b", ev.Paths[0])
	case err := <-errCh:
		require.NoError(t, err)
	case <-time.After(2 * time.Second):
		require.Fail(t, "timeout second line")
	}
}
