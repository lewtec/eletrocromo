package main

import (
	"context"
	"fmt"
	"os"

	"github.com/lewtec/eletrocromo/internal/version"
)

type versionCmd struct{}

func (versionCmd) Description() string {
	return "Print build version (goreleaser -X / VCS / git)."
}

func (versionCmd) Run(context.Context) error {
	_, err := fmt.Fprintln(os.Stdout, version.Resolve().String())
	return err
}
