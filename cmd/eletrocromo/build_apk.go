package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/lewtec/eletrocromo/internal/gen/apk"
	"github.com/lewtec/lewkit/x/cmd"
)

// ErrMissingPackageID is returned when build android has no package_id (config or --id).
var ErrMissingPackageID = errors.New("package id required")

// hostFlags are the flags shared by build android, build macos, and build ios.
type hostFlags struct {
	config      cmd.StringArg   `long:"config" default:"" help:"path to eletrocromo.json (default: ./eletrocromo.json if present)"`
	id          cmd.StringArg   `long:"id" default:"" help:"package id (overrides config)"`
	name        cmd.StringArg   `long:"name" default:"" help:"app display name (overrides config)"`
	goMain      cmd.StringArg   `long:"go-main" default:"" help:"Go main package directory (overrides config; default .)"`
	version     cmd.StringArg   `long:"version" default:"" help:"version name (default: git describe / goreleaser -X / devel)"`
	code        cmd.IntArg[int] `long:"code" default:"0" help:"version code (default: semver map or git rev-list count)"`
	out         cmd.StringArg   `long:"out" default:"" help:"output path"`
	workDir     cmd.StringArg   `long:"workdir" default:"" help:"project dir (default: temp; kept if set)"`
	keepWorkDir cmd.Flag        `long:"keep-workdir" help:"do not delete temp workdir after success"`
	goOnly      cmd.Flag        `long:"go-only" help:"stop before the platform build"`
	icon        cmd.StringArg   `long:"icon" default:"" help:"master PNG/JPEG (overrides config icon)"`
	output      cmd.StringArg   `long:"output" default:"dist/icons" help:"icon tree root"`
	refresh     cmd.Flag        `long:"refresh-icons" help:"regenerate icons even if present"`
}

func (f hostFlags) load(cwd string) (apk.Config, string, error) {
	var cfg apk.Config
	baseDir := cwd
	cfgPath := strings.TrimSpace(f.config.Value())
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
	cfg = apk.Merge(cfg, apk.Config{
		PackageID:   f.id.Value(),
		AppName:     f.name.Value(),
		GoMain:      f.goMain.Value(),
		VersionName: f.version.Value(),
		VersionCode: f.code.Value(),
	})
	return cfg, baseDir, nil
}

type buildAndroidCmd struct {
	hostFlags
}

func (buildAndroidCmd) Description() string {
	return "JIT Android host, cross-compile the Go app (GOOS=android), and assemble a debug APK. Runs icon generation when the tree is incomplete. Use --go-only to stop after jniLibs."
}

func (c *buildAndroidCmd) Run(ctx context.Context) error {
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	cfg, baseDir, err := c.load(cwd)
	if err != nil {
		return err
	}
	if strings.TrimSpace(cfg.PackageID) == "" {
		return fmt.Errorf("%w: set package_id in %s or pass --id", ErrMissingPackageID, apk.ConfigFileName)
	}
	iconSrc := resolveIconSource(cwd, c.icon.Value(), baseDir, cfg.Icon)
	iconOut := resolveIconOutput(cwd, c.output.Value())
	outAPK := strings.TrimSpace(c.out.Value())
	if outAPK == "" && !c.goOnly.Value() {
		outAPK = apk.DefaultOutAPK(cfg.PackageID, cwd)
	}
	return runIconsThen(ctx, iconThen{src: iconSrc, out: iconOut, refresh: c.refresh.Value(), name: "android"}, func(iconRoot string) error {
		result, err := apk.Build(apk.BuildOptions{
			Config:      cfg,
			BaseDir:     baseDir,
			WorkDir:     c.workDir.Value(),
			KeepWorkDir: c.keepWorkDir.Value() || c.workDir.Value() != "",
			OutAPK:      outAPK,
			GoOnly:      c.goOnly.Value(),
			IconRoot:    iconRoot,
			Stdout:      os.Stdout,
			Stderr:      os.Stderr,
		})
		if err != nil {
			return err
		}
		if c.goOnly.Value() {
			if _, err := fmt.Fprintf(os.Stdout, "go libs:\n"); err != nil {
				return err
			}
			for _, p := range result.JNILibs {
				if _, err := fmt.Fprintf(os.Stdout, "  %s\n", p); err != nil {
					return err
				}
			}
			_, err = fmt.Fprintf(os.Stdout, "work dir: %s\n", result.WorkDir)
			return err
		}
		_, err = fmt.Fprintf(os.Stdout, "ok %s\n", result.APKPath)
		return err
	})
}
