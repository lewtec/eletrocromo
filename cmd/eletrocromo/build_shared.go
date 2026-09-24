package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/lewtec/eletrocromo/internal/gen/apk"
	"github.com/lewtec/eletrocromo/internal/gen/common"
	"github.com/lewtec/eletrocromo/internal/gen/ios"
	"github.com/lewtec/eletrocromo/internal/gen/mac"
	"github.com/lewtec/eletrocromo/internal/icons"
	"github.com/spf13/cobra"
)

// buildCmdFlags is the flag set shared by build android, ios, and macos.
type buildCmdFlags struct {
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
}

// buildFlagHelp is the platform-specific usage text. Shared flags keep one string.
type buildFlagHelp struct {
	id      string
	name    string
	version string
	code    string
	out     string
	workDir string
	goOnly  string
}

func (f *buildCmdFlags) bind(cmd *cobra.Command, h buildFlagHelp) {
	cmd.Flags().StringVar(&f.configPath, "config", "", "path to eletrocromo.json (default: ./eletrocromo.json if present)")
	cmd.Flags().StringVar(&f.id, "id", "", h.id)
	cmd.Flags().StringVar(&f.name, "name", "", h.name)
	cmd.Flags().StringVar(&f.goMain, "go-main", ".", "Go main package directory (overrides config)")
	cmd.Flags().StringVar(&f.version, "version", "", h.version)
	cmd.Flags().IntVar(&f.code, "code", 0, h.code)
	cmd.Flags().StringVar(&f.out, "out", "", h.out)
	cmd.Flags().StringVar(&f.workDir, "workdir", "", h.workDir)
	cmd.Flags().BoolVar(&f.keepWorkDir, "keep-workdir", false, "do not delete temp workdir after success")
	cmd.Flags().BoolVar(&f.goOnly, "go-only", false, h.goOnly)
	cmd.Flags().StringVar(&f.iconPath, "icon", "", "master PNG/JPEG (overrides config icon)")
	cmd.Flags().StringVar(&f.iconOutput, "output", icons.DefaultOutputDir, "icon tree root")
	cmd.Flags().BoolVar(&f.refresh, "refresh-icons", false, "regenerate icons even if present")
}

func (f *buildCmdFlags) load(cmd *cobra.Command) (cwd, baseDir string, cfg apk.Config, iconSrc, iconOut string, err error) {
	cwd, err = os.Getwd()
	if err != nil {
		return "", "", apk.Config{}, "", "", err
	}
	cfg, baseDir, err = loadAPKConfig(cwd, f, cmd)
	if err != nil {
		return "", "", apk.Config{}, "", "", err
	}
	if strings.TrimSpace(cfg.PackageID) == "" {
		return "", "", apk.Config{}, "", "", fmt.Errorf("%w: set package_id in %s or pass --id", ErrMissingPackageID, apk.ConfigFileName)
	}
	iconSrc = resolveIconSource(cwd, f.iconPath, baseDir, cfg.Icon)
	iconOut = resolveIconOutput(cwd, f.iconOutput)
	return cwd, baseDir, cfg, iconSrc, iconOut, nil
}

func (f *buildCmdFlags) hostOutApp(appName, packageID, cwd string) string {
	out := strings.TrimSpace(f.out)
	if out != "" || f.goOnly {
		return out
	}
	if appName == "" {
		parts := strings.Split(packageID, ".")
		appName = parts[len(parts)-1]
	}
	return common.DefaultOutApp(appName, cwd)
}

func iosConfigFromAPK(cfg apk.Config) ios.Config {
	return ios.Config{
		PackageID:    cfg.PackageID,
		AppName:      cfg.AppName,
		VersionName:  cfg.VersionName,
		VersionCode:  cfg.VersionCode,
		GoMain:       cfg.GoMain,
		Icon:         cfg.Icon,
		Capabilities: cfg.Capabilities,
	}
}

func macConfigFromAPK(cfg apk.Config) mac.Config {
	return mac.Config{
		PackageID:    cfg.PackageID,
		AppName:      cfg.AppName,
		VersionName:  cfg.VersionName,
		VersionCode:  cfg.VersionCode,
		GoMain:       cfg.GoMain,
		Icon:         cfg.Icon,
		Capabilities: cfg.Capabilities,
	}
}
