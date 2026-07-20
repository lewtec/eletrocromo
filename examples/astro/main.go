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
	"os"
	"os/signal"

	"github.com/lewtec/eletrocromo"
	"github.com/lucasew/orvalho/pkg/workers"
)

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

	app := eletrocromo.App{
		ID:      "br.tec.lew.eletrocromo.astro",
		Handler: workers.Handler(iso),
		Context: ctx,
	}
	log.Printf("astro example: launching Helium window (//go:embed guest + assets)")
	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
