// Command eletrocromo is the packaging CLI for the library.
// Library apps import github.com/lewtec/eletrocromo; this binary builds
// icons and the Android, macOS, and iOS hosts.
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"

	"github.com/lewtec/eletrocromo/internal/version"
	"github.com/lewtec/lewkit/x/cmd"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()
	if err := run(ctx, os.Args[1:]); err != nil {
		if _, werr := fmt.Fprintln(os.Stderr, "Error:", err); werr != nil {
			os.Exit(1)
		}
		os.Exit(1)
	}
}

func run(ctx context.Context, args []string) error {
	app, err := cmd.Parse[cmd.App[root]](args...)
	if err != nil {
		return err
	}
	// App --version prints the lewkit release string. This binary's version
	// is internal/version (goreleaser -X).
	if app.WantVersion() {
		_, err := fmt.Fprintln(os.Stdout, version.Resolve().String())
		return err
	}
	return app.Run(ctx)
}
