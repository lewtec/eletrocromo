package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/lewtec/eletrocromo/internal/gen/mac"
)

type macosCmd struct {
	hostFlags
}

func (macosCmd) Description() string {
	return "JIT macOS WKWebView host, darwin Go binary, and unsigned Debug .app. Runs icon generation when macos/icon.icns is missing. Use --go-only to stop after the helper."
}

func (c *macosCmd) Run(ctx context.Context) error {
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
	cfg := mac.Config{
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
		outApp = mac.DefaultOutApp(appName, cwd)
	}
	return runIconsThen(ctx, iconThen{src: iconSrc, out: iconOut, refresh: c.refresh.Value(), name: "macos"}, func(iconRoot string) error {
		result, err := mac.Build(mac.BuildOptions{
			Config:      cfg,
			BaseDir:     baseDir,
			WorkDir:     c.workDir.Value(),
			KeepWorkDir: c.keepWorkDir.Value() || c.workDir.Value() != "",
			OutApp:      outApp,
			GoOnly:      c.goOnly.Value(),
			IconRoot:    iconRoot,
			Stdout:      os.Stdout,
			Stderr:      os.Stderr,
		})
		if err != nil {
			return err
		}
		if c.goOnly.Value() {
			if _, err := fmt.Fprintf(os.Stdout, "helper: %s\n", result.HelperPath); err != nil {
				return err
			}
			_, err = fmt.Fprintf(os.Stdout, "work dir: %s\n", result.WorkDir)
			return err
		}
		_, err = fmt.Fprintf(os.Stdout, "ok %s\n", result.AppPath)
		return err
	})
}
