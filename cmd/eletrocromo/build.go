package main

import (
	"errors"

	"github.com/spf13/cobra"
)

// ErrMissingBuildTarget is returned for bare "build" with no subcommand.
var ErrMissingBuildTarget = errors.New("missing build target; use one of: icons, android, macos, ios\n\nExamples:\n  eletrocromo build icons\n  eletrocromo build android\n  eletrocromo build macos\n  eletrocromo build ios")

func newBuildCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "build",
		Short: "Packaging targets (icons, android, macos, ios, …)",
		Long: `Build packaging artifacts for eletrocromo apps.

Targets:
  icons    Multi-platform icon matrix (dist/icons by default)
  android  JIT Android host + Go binary + debug APK (runs icons if missing)
  macos    JIT macOS WKWebView host + darwin Go + unsigned Debug .app
  ios      JIT iOS WKWebView host + GOOS=ios c-archive + Debug .app

Bare "build" with no target is an error.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return ErrMissingBuildTarget
		},
	}
	cmd.AddCommand(newBuildIconsCmd())
	cmd.AddCommand(newBuildAndroidCmd())
	cmd.AddCommand(newBuildMacOSCmd())
	cmd.AddCommand(newBuildIOSCmd())
	return cmd
}
