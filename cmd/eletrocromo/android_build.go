package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/lewtec/eletrocromo/internal/apkgen"
	"github.com/spf13/cobra"
)

func newAndroidBuildCmd() *cobra.Command {
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
	)

	cmd := &cobra.Command{
		Use:   "build",
		Short: "Build a multiarch debug APK from an eletrocromo app",
		Long: `Generate the Android WebView host, cross-compile the Go app
(GOOS=android, multi-ABI), and assemble a debug APK.

Standard config: eletrocromo.json next to the app (or --config). Flags override
config fields.

Requires for a full APK:
  - Go toolchain (CGO_ENABLED=0 GOOS=android)
  - JDK 17+ (java on PATH)
  - Android SDK (ANDROID_HOME or ANDROID_SDK_ROOT)
  - Gradle 8.9+ on PATH (or gradlew in --workdir)

Use --go-only to stop after jniLibs (no SDK).

Example (from examples/counter):
  eletrocromo android build
  eletrocromo android build --out ../../dist/counter-debug.apk`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cwd, err := os.Getwd()
			if err != nil {
				return err
			}

			var cfg apkgen.Config
			baseDir := cwd
			cfgPath := strings.TrimSpace(configPath)
			if cfgPath == "" {
				// Default: ./eletrocromo.json when present.
				try := filepath.Join(cwd, apkgen.ConfigFileName)
				if st, err := os.Stat(try); err == nil && !st.IsDir() {
					cfgPath = try
				}
			}
			if cfgPath != "" {
				loaded, dir, err := apkgen.LoadConfig(cfgPath)
				if err != nil {
					return err
				}
				cfg = loaded
				baseDir = dir
			}

			overlay := apkgen.Config{
				PackageID:   id,
				AppName:     name,
				GoMain:      goMain,
				VersionName: version,
			}
			if cmd.Flags().Changed("code") {
				overlay.VersionCode = code
			}
			// Only apply go-main overlay when flag set (default "." would clobber).
			if !cmd.Flags().Changed("go-main") {
				overlay.GoMain = ""
			}
			if !cmd.Flags().Changed("version") {
				overlay.VersionName = ""
			}
			cfg = apkgen.Merge(cfg, overlay)

			if strings.TrimSpace(cfg.PackageID) == "" {
				return fmt.Errorf("package id required: set package_id in %s or pass --id", apkgen.ConfigFileName)
			}

			outAPK := strings.TrimSpace(out)
			if outAPK == "" && !goOnly {
				outAPK = apkgen.DefaultOutAPK(cfg.PackageID, cwd)
			}

			result, err := apkgen.Build(apkgen.BuildOptions{
				Config:      cfg,
				BaseDir:     baseDir,
				WorkDir:     workDir,
				KeepWorkDir: keepWorkDir || workDir != "",
				OutAPK:      outAPK,
				GoOnly:      goOnly,
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
		},
	}

	cmd.Flags().StringVar(&configPath, "config", "", "path to eletrocromo.json (default: ./eletrocromo.json if present)")
	cmd.Flags().StringVar(&id, "id", "", "package id / applicationId (overrides config)")
	cmd.Flags().StringVar(&name, "name", "", "launcher label (overrides config)")
	cmd.Flags().StringVar(&goMain, "go-main", ".", "Go main package directory (overrides config)")
	cmd.Flags().StringVar(&version, "version", "0.1.0", "versionName (overrides config)")
	cmd.Flags().IntVar(&code, "code", 1, "versionCode (overrides config)")
	cmd.Flags().StringVar(&out, "out", "", "output APK path (default: dist/<name>-debug.apk)")
	cmd.Flags().StringVar(&workDir, "workdir", "", "Gradle project dir (default: temp; kept if set)")
	cmd.Flags().BoolVar(&keepWorkDir, "keep-workdir", false, "do not delete temp workdir after success")
	cmd.Flags().BoolVar(&goOnly, "go-only", false, "only cross-compile Go into jniLibs (skip Gradle/SDK)")

	return cmd
}
