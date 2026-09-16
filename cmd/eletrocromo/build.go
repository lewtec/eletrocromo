package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newBuildCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "build",
		Short: "Packaging targets (icons, android, …)",
		Long: `Build packaging artifacts for eletrocromo apps.

Targets:
  icons    Multi-platform icon matrix (dist/icons by default)
  android  JIT Android host + Go binary + debug APK (runs icons if missing)

Bare "build" with no target is an error.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return fmt.Errorf("missing build target; use one of: icons, android\n\nExamples:\n  eletrocromo build icons\n  eletrocromo build android")
		},
	}
	cmd.AddCommand(newBuildIconsCmd())
	cmd.AddCommand(newBuildAndroidCmd())
	return cmd
}
