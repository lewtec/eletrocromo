package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"

	"github.com/lewtec/eletrocromo"
)

func main() {
	// App.Run blocks until Context is cancelled, then waits for background
	// tasks and closes the server. Wire Ctrl+C (SIGINT) so the process
	// shuts down cleanly instead of dying mid-flight with an open browser.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	app := eletrocromo.App{
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = fmt.Fprintf(w, "it works!")
		}),
		Context: ctx,
	}
	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
