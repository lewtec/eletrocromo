// Command eletrocromo is the packaging / tooling CLI for the library.
// Library apps import github.com/lewtec/eletrocromo; this binary is for
// generators (Android host, later packaging helpers).
package main

import (
	"fmt"
	"os"
)

func main() {
	root := newRootCmd()
	if err := root.Execute(); err != nil {
		// Exiting; write failure cannot be recovered usefully.
		if _, werr := fmt.Fprintln(os.Stderr, "Error:", err); werr != nil {
			os.Exit(1)
		}
		os.Exit(1)
	}
}
