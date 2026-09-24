package main

import (
	"fmt"

	"github.com/lewtec/eletrocromo/internal/gen/mac"
	"github.com/spf13/cobra"
)

func newBuildMacOSCmd() *cobra.Command {
	var f buildCmdFlags

	cmd := &cobra.Command{
		Use:   "macos",
		Short: "JIT macOS WKWebView host + darwin Go + unsigned Debug .app",
		Long: `Generate an ephemeral XcodeGen host, compile the Go app
(CGO_ENABLED=0 GOOS=darwin, host arch), and assemble an unsigned Debug .app.

Runs "build icons" first when macos/icon.icns is missing (or always with
--refresh-icons). The .icns is copied into the bundle.

Standard config: eletrocromo.json (or --config). Flags override.
Same keys as "build android" (package_id becomes CFBundleIdentifier).

Requires for a full .app (Mac only):
  - Xcode (xcodebuild)
  - xcodegen on PATH

Use --go-only to stop after the darwin helper (no Xcode).

Example (from examples/counter):
  eletrocromo build macos
  eletrocromo build macos --out ../../dist/Counter.app`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cwd, baseDir, apkCfg, iconSrc, iconOut, err := f.load(cmd)
			if err != nil {
				return err
			}
			cfg := macConfigFromAPK(apkCfg)
			outApp := f.hostOutApp(cfg.AppName, cfg.PackageID, cwd)

			return runIconsThen(cmd, iconThen{src: iconSrc, out: iconOut, refresh: f.refresh, name: "macos"}, func(iconRoot string) error {
				result, err := mac.Build(mac.BuildOptions{
					Config:      cfg,
					BaseDir:     baseDir,
					WorkDir:     f.workDir,
					KeepWorkDir: f.keepWorkDir || f.workDir != "",
					OutApp:      outApp,
					GoOnly:      f.goOnly,
					IconRoot:    iconRoot,
					Stdout:      cmd.OutOrStdout(),
					Stderr:      cmd.ErrOrStderr(),
				})
				if err != nil {
					return err
				}
				outw := cmd.OutOrStdout()
				if f.goOnly {
					if _, err := fmt.Fprintf(outw, "helper: %s\n", result.HelperPath); err != nil {
						return err
					}
					_, err = fmt.Fprintf(outw, "work dir: %s\n", result.WorkDir)
					return err
				}
				_, err = fmt.Fprintf(outw, "ok %s\n", result.AppPath)
				return err
			})
		},
	}

	f.bind(cmd, buildFlagHelp{
		id:      "package id / CFBundleIdentifier (overrides config)",
		name:    "app display name (overrides config)",
		version: "CFBundleShortVersionString (default: git describe / goreleaser -X / devel)",
		code:    "CFBundleVersion (default: semver map or git rev-list count)",
		out:     "output .app path (default: dist/<app_name>.app)",
		workDir: "XcodeGen project dir (default: temp; kept if set)",
		goOnly:  "only cross-compile darwin Go helper (skip Xcode)",
	})

	return cmd
}
