// Named build_iosapp.go because *_ios.go is only compiled for GOOS=ios.
package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/lewtec/eletrocromo/internal/gen/ios"
	"github.com/lewtec/lewkit/x/cmd"
)

type iosCmd struct {
	hostFlags
	sdk cmd.StringArg `long:"sdk" default:"iphonesimulator" help:"iphonesimulator or iphoneos (also: simulator, device)"`
}

func (iosCmd) Description() string {
	return "JIT iOS WKWebView host and GOOS=ios c-archive Debug .app. The archive exports EletrocromoStart. Use --go-only to stop after the archive (host tree only off macOS)."
}

func (c *iosCmd) Run(ctx context.Context) error {
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	apkCfg, baseDir, err := c.load(cwd)
	if err != nil {
		return err
	}
	if strings.TrimSpace(apkCfg.PackageID) == "" {
		return fmt.Errorf("%w: set package_id in eletrocromo.json or pass --id", ErrMissingPackageID)
	}
	iconSrc := resolveIconSource(cwd, c.icon.Value(), baseDir, apkCfg.Icon)
	iconOut := resolveIconOutput(cwd, c.output.Value())
	cfg := ios.Config{
		PackageID:    apkCfg.PackageID,
		AppName:      apkCfg.AppName,
		VersionName:  apkCfg.VersionName,
		VersionCode:  apkCfg.VersionCode,
		GoMain:       apkCfg.GoMain,
		Icon:         apkCfg.Icon,
		Capabilities: apkCfg.Capabilities,
	}
	outApp := strings.TrimSpace(c.out.Value())
	if outApp == "" && !c.goOnly.Value() {
		appName := cfg.AppName
		if appName == "" {
			parts := strings.Split(cfg.PackageID, ".")
			appName = parts[len(parts)-1]
		}
		outApp = ios.DefaultOutApp(appName, cwd)
	}
	return runIconsThen(ctx, iconThen{src: iconSrc, out: iconOut, refresh: c.refresh.Value(), name: "ios"}, func(iconRoot string) error {
		result, err := ios.Build(ios.BuildOptions{
			Config:      cfg,
			BaseDir:     baseDir,
			WorkDir:     c.workDir.Value(),
			KeepWorkDir: c.keepWorkDir.Value() || c.workDir.Value() != "",
			OutApp:      outApp,
			GoOnly:      c.goOnly.Value(),
			SDK:         c.sdk.Value(),
			IconRoot:    iconRoot,
			Stdout:      os.Stdout,
			Stderr:      os.Stderr,
		})
		if err != nil {
			return err
		}
		if c.goOnly.Value() {
			if result.ArchivePath != "" {
				if _, err := fmt.Fprintf(os.Stdout, "archive: %s\n", result.ArchivePath); err != nil {
					return err
				}
			}
			_, err = fmt.Fprintf(os.Stdout, "work dir: %s\n", result.WorkDir)
			return err
		}
		_, err = fmt.Fprintf(os.Stdout, "ok %s\n", result.AppPath)
		return err
	})
}
