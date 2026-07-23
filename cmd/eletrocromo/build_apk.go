package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/lewtec/eletrocromo/internal/apkgen"
	"github.com/lewtec/eletrocromo/internal/icons"
	"github.com/lucasew/workspaced/pkg/logging"
	"github.com/lucasew/workspaced/pkg/taskgroup"
	"github.com/spf13/cobra"
)

func newBuildAndroidCmd() *cobra.Command {
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
		Use:   "android",
		Short: "JIT Android host + multiarch Go + debug APK",
		Long: `Generate the Android WebView host, cross-compile the Go app
(GOOS=android, multi-ABI), and assemble a debug APK.

Runs "build icons" first when the icon tree is incomplete (or always with
--refresh-icons). Launcher mipmaps are copied into the JIT host.

Standard config: eletrocromo.json (or --config). Flags override.

Requires for a full APK:
  - Go toolchain (CGO_ENABLED=0 GOOS=android)
  - JDK 17+ (java on PATH)
  - Android SDK (ANDROID_HOME or ANDROID_SDK_ROOT)
  - Gradle 8.9+ on PATH (or gradlew in --workdir)

Use --go-only to stop after jniLibs (no SDK).

Example (from examples/counter):
  eletrocromo build android
  eletrocromo build android --out ../../dist/counter-debug.apk`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cwd, err := os.Getwd()
			if err != nil {
				return err
			}

			cfg, baseDir, err := loadAPKConfig(cwd, configPath, id, name, goMain, version, code, cmd)
			if err != nil {
				return err
			}
			if strings.TrimSpace(cfg.PackageID) == "" {
				return fmt.Errorf("package id required: set package_id in %s or pass --id", apkgen.ConfigFileName)
			}

			// Resolve icon source: flag > config
			iconSrc := strings.TrimSpace(iconPath)
			if iconSrc == "" {
				iconSrc = strings.TrimSpace(cfg.Icon)
				if iconSrc != "" && !filepath.IsAbs(iconSrc) {
					iconSrc = filepath.Join(baseDir, iconSrc)
				}
			} else if !filepath.IsAbs(iconSrc) {
				iconSrc = filepath.Join(cwd, iconSrc)
			}

			iconOut := strings.TrimSpace(iconOutput)
			if iconOut == "" {
				iconOut = icons.DefaultOutputDir
			}
			if !filepath.IsAbs(iconOut) {
				iconOut = filepath.Join(cwd, iconOut)
			}

			outAPK := strings.TrimSpace(out)
			if outAPK == "" && !goOnly {
				outAPK = apkgen.DefaultOutAPK(cfg.PackageID, cwd)
			}

			ctx := logging.NewWriterContext(cmd.ErrOrStderr())
			// quieter: discard slog noise from taskgroup unless needed
			ctx = logging.ContextWithLogger(ctx, slog.New(slog.NewTextHandler(cmd.ErrOrStderr(), &slog.HandlerOptions{Level: slog.LevelWarn})))

			g, ctx := taskgroup.New(ctx, taskgroup.DefaultLimits())
			var iconRoot string

			g.Go("icons", taskgroup.CPU, func(ctx context.Context, s *taskgroup.Status) error {
				force := refresh
				if !force && icons.Complete(iconOut) {
					iconRoot = iconOut
					fmt.Fprintf(cmd.OutOrStdout(), "eletrocromo: icons already present at %s\n", iconOut)
					return nil
				}
				man, err := icons.Generate(icons.Options{
					SourcePath: iconSrc,
					OutputDir:  iconOut,
					Force:      force || !icons.Complete(iconOut),
				})
				if err != nil {
					return err
				}
				iconRoot = man.OutputDir
				fmt.Fprintf(cmd.OutOrStdout(), "eletrocromo: icons → %s\n", iconRoot)
				return nil
			})

			g.Go("android", taskgroup.IO, func(ctx context.Context, s *taskgroup.Status) error {
				result, err := apkgen.Build(apkgen.BuildOptions{
					Config:      cfg,
					BaseDir:     baseDir,
					WorkDir:     workDir,
					KeepWorkDir: keepWorkDir || workDir != "",
					OutAPK:      outAPK,
					GoOnly:      goOnly,
					IconRoot:    iconRoot,
					Stdout:      cmd.OutOrStdout(),
					Stderr:      cmd.ErrOrStderr(),
				})
				if err != nil {
					return err
				}
				if goOnly {
					fmt.Fprintf(cmd.OutOrStdout(), "go libs:\n")
					for _, p := range result.JNILibs {
						fmt.Fprintf(cmd.OutOrStdout(), "  %s\n", p)
					}
					fmt.Fprintf(cmd.OutOrStdout(), "work dir: %s\n", result.WorkDir)
					return nil
				}
				fmt.Fprintf(cmd.OutOrStdout(), "ok %s\n", result.APKPath)
				return nil
			}, "icons")

			return g.Wait()
		},
	}

	cmd.Flags().StringVar(&configPath, "config", "", "path to eletrocromo.json (default: ./eletrocromo.json if present)")
	cmd.Flags().StringVar(&id, "id", "", "package id / applicationId (overrides config)")
	cmd.Flags().StringVar(&name, "name", "", "launcher label (overrides config)")
	cmd.Flags().StringVar(&goMain, "go-main", ".", "Go main package directory (overrides config)")
	cmd.Flags().StringVar(&version, "version", "", "versionName (default: git describe / goreleaser -X / devel)")
	cmd.Flags().IntVar(&code, "code", 0, "versionCode (default: semver map or git rev-list count)")
	cmd.Flags().StringVar(&out, "out", "", "output APK path (default: dist/<name>-debug.apk)")
	cmd.Flags().StringVar(&workDir, "workdir", "", "Gradle project dir (default: temp; kept if set)")
	cmd.Flags().BoolVar(&keepWorkDir, "keep-workdir", false, "do not delete temp workdir after success")
	cmd.Flags().BoolVar(&goOnly, "go-only", false, "only cross-compile Go into jniLibs (skip Gradle/SDK)")
	cmd.Flags().StringVar(&iconPath, "icon", "", "master PNG/JPEG (overrides config icon)")
	cmd.Flags().StringVar(&iconOutput, "output", icons.DefaultOutputDir, "icon tree root")
	cmd.Flags().BoolVar(&refresh, "refresh-icons", false, "regenerate icons even if present")

	return cmd
}

func loadAPKConfig(cwd, configPath, id, name, goMain, version string, code int, cmd *cobra.Command) (apkgen.Config, string, error) {
	var cfg apkgen.Config
	baseDir := cwd
	cfgPath := strings.TrimSpace(configPath)
	if cfgPath == "" {
		try := filepath.Join(cwd, apkgen.ConfigFileName)
		if st, err := os.Stat(try); err == nil && !st.IsDir() {
			cfgPath = try
		}
	}
	if cfgPath != "" {
		loaded, dir, err := apkgen.LoadConfig(cfgPath)
		if err != nil {
			return apkgen.Config{}, "", err
		}
		cfg = loaded
		baseDir = dir
	}

	overlay := apkgen.Config{
		PackageID: id,
		AppName:   name,
		GoMain:    goMain,
	}
	if cmd.Flags().Changed("code") {
		overlay.VersionCode = code
	}
	if cmd.Flags().Changed("version") {
		overlay.VersionName = version
	}
	if !cmd.Flags().Changed("go-main") {
		overlay.GoMain = ""
	}
	cfg = apkgen.Merge(cfg, overlay)
	return cfg, baseDir, nil
}
