package main

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"
)

// ErrMissingBuildTarget is returned when `build` is invoked with no subcommand
// (icons, android, …). Callers can use errors.Is.
var ErrMissingBuildTarget = errors.New("missing build target; use one of: icons, android")

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
			return fmt.Errorf("%w\n\nExamples:\n  eletrocromo build icons\n  eletrocromo build android", ErrMissingBuildTarget)
		},
	}
	cmd.AddCommand(newBuildIconsCmd())
	cmd.AddCommand(newBuildAndroidCmd())
	return cmd
}
