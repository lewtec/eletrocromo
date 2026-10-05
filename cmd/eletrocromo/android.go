package main

import (
	"github.com/spf13/cobra"
)

func newAndroidCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:        "android",
		Short:      "Android packaging (prefer: eletrocromo build android)",
		Long:       "Legacy namespace. Prefer \"eletrocromo build android\" for JIT APK builds. \"create\" scaffolds a host project (not the happy path).",
		Deprecated: "use \"eletrocromo build android\" (and \"build icons\")",
	}
	cmd.AddCommand(newAndroidCreateCmd())
	cmd.AddCommand(newAndroidBuildCmd())
	return cmd
}
