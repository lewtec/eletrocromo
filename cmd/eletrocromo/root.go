package main

import "github.com/lewtec/lewkit/x/cmd"

type root struct {
	build   *buildCmd
	run     *runCmd
	icons   *iconsCmd
	android *androidCmd
	version *cmd.VersionCmd
}

func (root) Description() string {
	return "CLI for packaging eletrocromo apps. build and run take eletrocromo.json and follow GOOS/GOARCH."
}
