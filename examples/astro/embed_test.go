package main

import (
	"io"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/lucasew/orvalho/pkg/workers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEmbeddedGuestServesHome(t *testing.T) {
	if len(guestJS) < 10_000 {
		t.Skip("embed/guest.js looks like a placeholder — run: mise run build")
	}
	assets, err := fs.Sub(assetsRoot, "embed/assets")
	require.NoError(t, err)
	iso := workers.New(guestJS, workers.Options{
		Bindings: map[string]workers.Binding{
			"ASSETS": workers.NewAssetBinding(assets, "."),
		},
		Fetch: workers.HTTPFetch(workers.EgressList{
			"catfact.ninja",
			"https://catfact.ninja",
		}, nil, 0),
	})
	srv := httptest.NewServer(workers.Handler(iso))
	t.Cleanup(srv.Close)

	res, err := http.Get(srv.URL + "/")
	require.NoError(t, err)
	t.Cleanup(func() { _ = res.Body.Close() })
	body, err := io.ReadAll(res.Body)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, res.StatusCode, truncate(string(body), 400))
	assert.Contains(t, string(body), "blockquote", truncate(string(body), 400))
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
