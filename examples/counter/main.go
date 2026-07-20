// Counter is a dogfood app for eletrocromo host resolve / Helium --app launch.
//
//	go run ./examples/counter
//	# or: mise run example:counter
//
// Ctrl+C shuts down the process (window-owned lifetime is not wired yet).
package main

import (
	"context"
	"html/template"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync/atomic"

	"github.com/lewtec/eletrocromo"
)

var page = template.Must(template.New("counter").Parse(`<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>eletrocromo counter</title>
  <style>
    :root { color-scheme: light dark; font-family: system-ui, sans-serif; }
    body { max-width: 28rem; margin: 3rem auto; padding: 0 1rem; text-align: center; }
    h1 { font-size: 1.25rem; font-weight: 600; }
    .count { font-size: 4rem; font-variant-numeric: tabular-nums; margin: 1.5rem 0; }
    form { display: inline-flex; gap: 0.75rem; }
    button {
      font: inherit; padding: 0.5rem 1.25rem; border-radius: 0.5rem;
      border: 1px solid color-mix(in srgb, CanvasText 25%, transparent);
      background: color-mix(in srgb, CanvasText 8%, Canvas); cursor: pointer;
    }
    button:hover { background: color-mix(in srgb, CanvasText 14%, Canvas); }
    p.hint { margin-top: 2rem; font-size: 0.85rem; opacity: 0.7; }
  </style>
</head>
<body>
  <h1>eletrocromo counter</h1>
  <p class="count">{{.Count}}</p>
  <form method="POST" action="/">
    <button type="submit" name="op" value="dec">−</button>
    <button type="submit" name="op" value="inc">+</button>
  </form>
  <form method="POST" action="/" style="display:block;margin-top:0.75rem">
    <button type="submit" name="op" value="reset">reset</button>
  </form>
  <p class="hint">Server-rendered with Go html/template. Close with Ctrl+C in the terminal.</p>
</body>
</html>
`))

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	var count atomic.Int64

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		switch r.Method {
		case http.MethodGet:
			// ok
		case http.MethodPost:
			if err := r.ParseForm(); err != nil {
				http.Error(w, "bad form", http.StatusBadRequest)
				return
			}
			switch r.Form.Get("op") {
			case "inc":
				count.Add(1)
			case "dec":
				count.Add(-1)
			case "reset":
				count.Store(0)
			}
			// Post/Redirect/Get so refresh does not re-submit.
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := page.Execute(w, map[string]int64{"Count": count.Load()}); err != nil {
			log.Printf("template: %v", err)
		}
	})

	app := eletrocromo.App{
		ID:      "br.tec.lew.eletrocromo.counter",
		Handler: mux,
		Context: ctx,
	}
	log.Printf("counter example: launching app window (Helium-first host resolve)")
	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
