// Named build_iosapp.go because *_ios.go is only compiled for GOOS=ios.
package main

import (
	"fmt"

	"github.com/lewtec/eletrocromo/internal/gen/ios"
	"github.com/spf13/cobra"
)

func newBuildIOSCmd() *cobra.Command {
	var (
		f   buildCmdFlags
		sdk string
	)

	cmd := &cobra.Command{
		Use:   "ios",
		Short: "JIT iOS WKWebView host + GOOS=ios c-archive + Debug .app",
		Long: `Generate an ephemeral XcodeGen host, compile the Go app as a
GOOS=ios c-archive (CGO, Xcode clang), and assemble a Debug .app.

iOS cannot exec a helper binary. The archive exports EletrocromoStart
and the host waits on ELETROCROMO_READY_FILE.

Runs "build icons" first when the icon tree is incomplete (or always with
--refresh-icons). A 1024 AppIcon is written from source/master.png.

Standard config: eletrocromo.json (or --config). Flags override.
Same keys as "build android" / "build macos" (package_id becomes CFBundleIdentifier).

Requires for a full .app (Mac only):
  - Xcode (xcodebuild) with the iOS SDK
  - xcodegen on PATH
  - iOS Simulator runtime to launch (generic/platform=iOS Simulator still compiles)

Use --go-only to stop after the c-archive (no xcodebuild). Off macOS,
--go-only writes the host tree only.

Example (from examples/counter):
  eletrocromo build ios
  eletrocromo build ios --out ../../dist/Counter.app`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cwd, baseDir, apkCfg, iconSrc, iconOut, err := f.load(cmd)
			if err != nil {
				return err
			}
			cfg := iosConfigFromAPK(apkCfg)
			outApp := f.hostOutApp(cfg.AppName, cfg.PackageID, cwd)

			return runIconsThen(cmd, iconThen{src: iconSrc, out: iconOut, refresh: f.refresh, name: "ios"}, func(iconRoot string) error {
				result, err := ios.Build(ios.BuildOptions{
					Config:      cfg,
					BaseDir:     baseDir,
					WorkDir:     f.workDir,
					KeepWorkDir: f.keepWorkDir || f.workDir != "",
					OutApp:      outApp,
					GoOnly:      f.goOnly,
					SDK:         sdk,
					IconRoot:    iconRoot,
					Stdout:      cmd.OutOrStdout(),
					Stderr:      cmd.ErrOrStderr(),
				})
				if err != nil {
					return err
				}
				outw := cmd.OutOrStdout()
				if f.goOnly {
					if result.ArchivePath != "" {
						if _, err := fmt.Fprintf(outw, "archive: %s\n", result.ArchivePath); err != nil {
							return err
						}
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
		goOnly:  "only write host + ios c-archive (skip xcodebuild)",
	})
	cmd.Flags().StringVar(&sdk, "sdk", ios.SDKSimulator, "iphonesimulator or iphoneos (also: simulator, device)")

	return cmd
}
