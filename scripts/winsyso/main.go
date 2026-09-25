package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/lewtec/eletrocromo/internal/winres"
)

func main() {
	arch := flag.String("arch", "amd64", "GOARCH")
	ico := flag.String("ico", "", "path to a .ico")
	out := flag.String("o", "", "path to the .syso")
	flag.Parse()
	if *ico == "" || *out == "" {
		fmt.Fprintln(os.Stderr, "usage: winsyso -arch amd64 -ico icon.ico -o rsrc_windows_amd64.syso")
		os.Exit(2)
	}
	if err := winres.Syso(*out, *arch, *ico); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
