package main

import (
	"context"
	"errors"
)

// ErrMissingBuildTarget is returned for bare "build" with no subcommand.
var ErrMissingBuildTarget = errors.New("missing build target; use one of: icons, android, macos, ios\n\nExamples:\n  eletrocromo build icons\n  eletrocromo build android\n  eletrocromo build macos\n  eletrocromo build ios")

type buildCmd struct {
	icons   *iconsCmd
	android *buildAndroidCmd
	macos   *macosCmd
	ios     *iosCmd
}

func (buildCmd) Description() string {
	return "Packaging targets: icons, android, macos, ios. Bare \"build\" with no target is an error."
}

func (*buildCmd) Run(context.Context) error {
	return ErrMissingBuildTarget
}
