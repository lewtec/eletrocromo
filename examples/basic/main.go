package main

import (
	"context"
	"fmt"
	"net/http"

	"github.com/lewtec/eletrocromo"
)

func main() {
	ctx := context.Background()
	app := eletrocromo.App{
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = fmt.Fprintf(w, "it works!")
		}),
		Context: ctx,
	}
	err := app.Run()
	if err != nil {
		panic(err)
	}
}
