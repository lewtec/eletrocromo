package main

import (
	"errors"
	"fmt"
	"strings"

	"github.com/lewtec/eletrocromo/internal/gen/apk"
	"github.com/spf13/cobra"
)

// ErrMissingPackageID is returned when build android has no package_id (config or --id).
// Callers can use errors.Is.
var ErrMissingPackageID = errors.New("package id required")

func newBuildAndroidCmd() *cobra.Command {
	var f buildCmdFlags

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
			cwd, baseDir, cfg, iconSrc, iconOut, err := f.load(cmd)
			if err != nil {
				return err
			}

			outAPK := strings.TrimSpace(f.out)
			if outAPK == "" && !f.goOnly {
				outAPK = apk.DefaultOutAPK(cfg.PackageID, cwd)
			}

			return runIconsThen(cmd, iconThen{src: iconSrc, out: iconOut, refresh: f.refresh, name: "android"}, func(iconRoot string) error {
				result, err := apk.Build(apk.BuildOptions{
					Config:      cfg,
					BaseDir:     baseDir,
					WorkDir:     f.workDir,
					KeepWorkDir: f.keepWorkDir || f.workDir != "",
					OutAPK:      outAPK,
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
					if _, err := fmt.Fprintf(outw, "go libs:\n"); err != nil {
						return err
					}
					for _, p := range result.JNILibs {
						if _, err := fmt.Fprintf(outw, "  %s\n", p); err != nil {
							return err
						}
					}
					_, err = fmt.Fprintf(outw, "work dir: %s\n", result.WorkDir)
					return err
				}
				_, err = fmt.Fprintf(outw, "ok %s\n", result.APKPath)
				return err
			})
		},
	}

	f.bind(cmd, buildFlagHelp{
		id:      "package id / applicationId (overrides config)",
		name:    "launcher label (overrides config)",
		version: "versionName (default: git describe / goreleaser -X / devel)",
		code:    "versionCode (default: semver map or git rev-list count)",
		out:     "output APK path (default: dist/<name>-debug.apk)",
		workDir: "Gradle project dir (default: temp; kept if set)",
		goOnly:  "only cross-compile Go into jniLibs (skip Gradle/SDK)",
	})

	return cmd
}

func loadAPKConfig(cwd string, f *buildCmdFlags, cmd *cobra.Command) (apk.Config, string, error) {
	var cfg apk.Config
	baseDir := cwd
	cfgPath := strings.TrimSpace(f.configPath)
	if cfgPath == "" {
		cfgPath = defaultConfigPath(cwd)
	}
	if cfgPath != "" {
		loaded, dir, err := apk.LoadConfig(cfgPath)
		if err != nil {
			return apk.Config{}, "", err
		}
		cfg = loaded
		baseDir = dir
	}

	overlay := apk.Config{
		PackageID: f.id,
		AppName:   f.name,
		GoMain:    f.goMain,
	}
	if cmd.Flags().Changed("code") {
		overlay.VersionCode = f.code
	}
	if cmd.Flags().Changed("version") {
		overlay.VersionName = f.version
	}
	if !cmd.Flags().Changed("go-main") {
		overlay.GoMain = ""
	}
	cfg = apk.Merge(cfg, overlay)
	return cfg, baseDir, nil
}
