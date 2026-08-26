package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/lewtec/eletrocromo/internal/gen/mac"
	"github.com/lewtec/eletrocromo/internal/icons"
	"github.com/lucasew/workspaced/pkg/logging"
	"github.com/lucasew/workspaced/pkg/taskgroup"
	"github.com/spf13/cobra"
)

func newBuildMacOSCmd() *cobra.Command {
	var (
		configPath  string
		id          string
		name        string
		goMain      string
		version     string
		code        int
		out         string
		workDir     string
		keepWorkDir bool
		goOnly      bool
		iconPath    string
		iconOutput  string
		refresh     bool
	)

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
			cwd, err := os.Getwd()
			if err != nil {
				return err
			}

			apkCfg, baseDir, err := loadAPKConfig(cwd, configPath, id, name, goMain, version, code, cmd)
			if err != nil {
				return err
			}
			if strings.TrimSpace(apkCfg.PackageID) == "" {
				return fmt.Errorf("%w: set package_id in eletrocromo.json or pass --id", ErrMissingPackageID)
			}

			iconSrc := resolveIconSource(cwd, iconPath, baseDir, apkCfg.Icon)
			iconOut := resolveIconOutput(cwd, iconOutput)

			cfg := mac.Config{
				PackageID:    apkCfg.PackageID,
				AppName:      apkCfg.AppName,
				VersionName:  apkCfg.VersionName,
				VersionCode:  apkCfg.VersionCode,
				GoMain:       apkCfg.GoMain,
				Icon:         apkCfg.Icon,
				Capabilities: apkCfg.Capabilities,
			}

			outApp := strings.TrimSpace(out)
			if outApp == "" && !goOnly {
				appName := cfg.AppName
				if appName == "" {
					parts := strings.Split(cfg.PackageID, ".")
					appName = parts[len(parts)-1]
				}
				outApp = mac.DefaultOutApp(appName, cwd)
			}

			ctx := logging.NewWriterContext(cmd.ErrOrStderr())
			ctx = logging.ContextWithLogger(ctx, slog.New(slog.NewTextHandler(cmd.ErrOrStderr(), &slog.HandlerOptions{Level: slog.LevelWarn})))

			g, _ := taskgroup.New(ctx, taskgroup.DefaultLimits())
			var iconRoot string

			g.Go("icons", taskgroup.CPU, func(ctx context.Context, s *taskgroup.Status) error {
				var err error
				iconRoot, err = ensureBuildIcons(cmd.OutOrStdout(), iconSrc, iconOut, refresh)
				return err
			})

			g.Go("macos", taskgroup.IO, func(ctx context.Context, s *taskgroup.Status) error {
				result, err := mac.Build(mac.BuildOptions{
					Config:      cfg,
					BaseDir:     baseDir,
					WorkDir:     workDir,
					KeepWorkDir: keepWorkDir || workDir != "",
					OutApp:      outApp,
					GoOnly:      goOnly,
					IconRoot:    iconRoot,
					Stdout:      cmd.OutOrStdout(),
					Stderr:      cmd.ErrOrStderr(),
				})
				if err != nil {
					return err
				}
				outw := cmd.OutOrStdout()
				if goOnly {
					if _, err := fmt.Fprintf(outw, "helper: %s\n", result.HelperPath); err != nil {
						return err
					}
					_, err = fmt.Fprintf(outw, "work dir: %s\n", result.WorkDir)
					return err
				}
				_, err = fmt.Fprintf(outw, "ok %s\n", result.AppPath)
				return err
			}, "icons")

			return g.Wait()
		},
	}

	cmd.Flags().StringVar(&configPath, "config", "", "path to eletrocromo.json (default: ./eletrocromo.json if present)")
	cmd.Flags().StringVar(&id, "id", "", "package id / CFBundleIdentifier (overrides config)")
	cmd.Flags().StringVar(&name, "name", "", "app display name (overrides config)")
	cmd.Flags().StringVar(&goMain, "go-main", ".", "Go main package directory (overrides config)")
	cmd.Flags().StringVar(&version, "version", "", "CFBundleShortVersionString (default: git describe / goreleaser -X / devel)")
	cmd.Flags().IntVar(&code, "code", 0, "CFBundleVersion (default: semver map or git rev-list count)")
	cmd.Flags().StringVar(&out, "out", "", "output .app path (default: dist/<app_name>.app)")
	cmd.Flags().StringVar(&workDir, "workdir", "", "XcodeGen project dir (default: temp; kept if set)")
	cmd.Flags().BoolVar(&keepWorkDir, "keep-workdir", false, "do not delete temp workdir after success")
	cmd.Flags().BoolVar(&goOnly, "go-only", false, "only cross-compile darwin Go helper (skip Xcode)")
	cmd.Flags().StringVar(&iconPath, "icon", "", "master PNG/JPEG (overrides config icon)")
	cmd.Flags().StringVar(&iconOutput, "output", icons.DefaultOutputDir, "icon tree root")
	cmd.Flags().BoolVar(&refresh, "refresh-icons", false, "regenerate icons even if present")

	return cmd
}
