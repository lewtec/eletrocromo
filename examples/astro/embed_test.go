package main

import (
	"io"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/lucasew/orvalho/pkg/workers"
)

func TestEmbeddedGuestServesHome(t *testing.T) {
	if len(guestJS) < 10_000 {
		t.Skip("embed/guest.js looks like a placeholder — run: mise run build")
	}
	assets, err := fs.Sub(assetsRoot, "embed/assets")
	if err != nil {
		t.Fatal(err)
	}
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
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	body, _ := io.ReadAll(res.Body)
	if res.StatusCode != 200 {
		t.Fatalf("status %d: %s", res.StatusCode, truncate(string(body), 400))
	}
	if !strings.Contains(string(body), "blockquote") {
		t.Fatalf("missing blockquote: %s", truncate(string(body), 400))
	}
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
