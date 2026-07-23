// Ticker is a dogfood app: a background goroutine increments a counter every
// second; the UI is a read-only html/template at GET /.
//
//	mise run example:ticker
//	# or: go -C examples/ticker run .
//
// Ctrl+C shuts down the process.
package main

import (
	"context"
	"html/template"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync/atomic"
	"time"

	"github.com/lewtec/eletrocromo"
)

var page = template.Must(template.New("ticker").Parse(`<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <meta http-equiv="refresh" content="1">
  <title>eletrocromo ticker</title>
  <style>
    :root { color-scheme: light dark; font-family: system-ui, sans-serif; }
    body { max-width: 28rem; margin: 3rem auto; padding: 0 1rem; text-align: center; }
    h1 { font-size: 1.25rem; font-weight: 600; }
    .count { font-size: 4rem; font-variant-numeric: tabular-nums; margin: 1.5rem 0; }
    p.hint { margin-top: 2rem; font-size: 0.85rem; opacity: 0.7; }
    p.meta { font-size: 0.8rem; opacity: 0.55; }
  </style>
</head>
<body>
  <h1>eletrocromo ticker</h1>
  <p class="count">{{.Count}}</p>
  <p class="meta">seconds since start (server clock)</p>
  <p class="hint">
    A Go goroutine adds 1 every second. This page only reads the value
    (server-rendered template; auto-refresh each second).
  </p>
</body>
</html>
`))

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	var count atomic.Int64

	// Background producer: only the server mutates count.
	go func() {
		t := time.NewTicker(time.Second)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				count.Add(1)
			}
		}
	}()

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store")
		if err := page.Execute(w, map[string]int64{"Count": count.Load()}); err != nil {
			log.Printf("template: %v", err)
		}
	})

	app := eletrocromo.App{
		ID:      "br.tec.lew.eletrocromo.ticker",
		Handler: mux,
		Context: ctx,
	}
	log.Printf("ticker example: background +1/s; UI is read-only template")
	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
