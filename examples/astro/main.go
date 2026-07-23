// Astro SSR (Cloudflare adapter) hosted by orvalho workers inside eletrocromo.
// Guest script + client assets are //go:embed'd (produce with: mise run build).
//
//	mise run build
//	mise run run
//
// Ctrl+C shuts down the process.
package main

import (
	"context"
	"embed"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/signal"

	"github.com/lewtec/eletrocromo"
	"github.com/lucasew/orvalho/pkg/workers"
)

// statusRecorder captures WriteHeader for post-request logging.
type statusRecorder struct {
	http.ResponseWriter
	code int
}

func (s *statusRecorder) WriteHeader(code int) {
	s.code = code
	s.ResponseWriter.WriteHeader(code)
}

//go:embed embed/guest.js
var guestJS string

//go:embed all:embed/assets
var assetsRoot embed.FS

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	assets, err := fs.Sub(assetsRoot, "embed/assets")
	if err != nil {
		log.Fatalf("embed assets: %v", err)
	}

	// Guest is already an orvalho IIFE (globalThis.default); no runtime esbuild.
	iso := workers.New(guestJS, workers.Options{
		Env: map[string]string{"SITE": "eletrocromo-astro"},
		Bindings: map[string]workers.Binding{
			"ASSETS": workers.NewAssetBinding(assets, "."),
		},
		Fetch: workers.HTTPFetch(workers.EgressList{
			"catfact.ninja",
			"https://catfact.ninja",
		}, nil, 0),
	})

	// Log worker failures (orvalho already prints to stderr; keep a clear
	// prefix so Android logcat is easy to filter).
	h := workers.Handler(iso)
	app := eletrocromo.App{
		ID: "br.tec.lew.eletrocromo.astro",
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			rw := &statusRecorder{ResponseWriter: w, code: http.StatusOK}
			h.ServeHTTP(rw, r)
			if rw.code >= 500 {
				log.Printf("astro handler: %s %s -> %d", r.Method, r.URL.RequestURI(), rw.code)
			}
		}),
		Context: ctx,
	}
	log.Printf("astro example: launching Helium window (//go:embed guest + assets)")
	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
