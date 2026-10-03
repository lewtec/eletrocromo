package main

import (
	"github.com/spf13/cobra"
)

// Legacy: eletrocromo android build → same as eletrocromo build android.
func newAndroidBuildCmd() *cobra.Command {
	// Reuse the primary implementation; cobra does not allow re-parenting easily,
	// so we clone via re-running the same constructor (separate flag state).
	cmd := newBuildAndroidCmd()
	cmd.Use = "build"
	cmd.Short = "Build a multiarch debug APK (alias of: eletrocromo build android)"
	cmd.Deprecated = "use \"eletrocromo build android\""
	return cmd
}
