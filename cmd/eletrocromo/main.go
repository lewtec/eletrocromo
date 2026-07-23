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
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}
