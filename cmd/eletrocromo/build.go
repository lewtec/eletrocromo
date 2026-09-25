package main

import (
	"context"
	"errors"
)

// ErrMissingBuildTarget is returned for bare "build" with no subcommand.
var ErrMissingBuildTarget = errors.New("missing build target; use one of: icons, android, macos, ios, linux, windows\n\nExamples:\n  eletrocromo build icons\n  eletrocromo build android\n  eletrocromo build macos\n  eletrocromo build ios\n  eletrocromo build linux\n  eletrocromo build windows")

type buildCmd struct {
	icons   *iconsCmd
	android *buildAndroidCmd
	macos   *macosCmd
	ios     *iosCmd
	linux   *linuxCmd
	windows *windowsCmd
}

func (buildCmd) Description() string {
	return "Packaging targets: icons, android, macos, ios, linux, windows. Bare \"build\" with no target is an error."
}

func (*buildCmd) Run(context.Context) error {
	return ErrMissingBuildTarget
}
