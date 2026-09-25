package main

import "github.com/lewtec/lewkit/x/cmd"

type root struct {
	build   *buildCmd
	android *androidCmd
	version *cmd.VersionCmd
}

func (root) Description() string {
	return "CLI for packaging eletrocromo apps (build icons, build android, build macos, build ios). The runtime library is imported as github.com/lewtec/eletrocromo."
}
