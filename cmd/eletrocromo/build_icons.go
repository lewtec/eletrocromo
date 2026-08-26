package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/lewtec/eletrocromo/internal/gen/apk"
	"github.com/lewtec/eletrocromo/internal/icons"
	"github.com/spf13/cobra"
)

func newBuildIconsCmd() *cobra.Command {
	var (
		configPath string
		iconPath   string
		output     string
		refresh    bool
	)
	cmd := &cobra.Command{
		Use:   "icons",
		Short: "Generate multi-platform icons from one master PNG/JPEG",
		Long: `Rasterize a master image (or the embedded default mark) into dist/icons:

  source/  windows/  macos/  linux/  android/  web/  manifest.json

Config: optional "icon" in eletrocromo.json. Flags override.
Skip when the tree is already complete unless --refresh-icons.

SVG is not rasterized in-process yet — convert to PNG/JPEG first
(or wait for a workspaced catalog tool pin).`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cwd, err := os.Getwd()
			if err != nil {
				return err
			}
			src, out, err := resolveIconIO(cwd, configPath, iconPath, output)
			if err != nil {
				return err
			}
			man, err := icons.Generate(icons.Options{
				SourcePath: src,
				OutputDir:  out,
				Force:      refresh,
			})
			if err != nil {
				return err
			}
			if !refresh && icons.Complete(out) && man != nil {
				_, err = fmt.Fprintf(cmd.OutOrStdout(), "icons up to date: %s\n", man.OutputDir)
			} else {
				_, err = fmt.Fprintf(cmd.OutOrStdout(), "ok %s (%d files)\n", man.OutputDir, len(man.Files))
			}
			return err
		},
	}
	cmd.Flags().StringVar(&configPath, "config", "", "path to eletrocromo.json (default: ./eletrocromo.json if present)")
	cmd.Flags().StringVar(&iconPath, "icon", "", "master PNG/JPEG (overrides config icon; default: embedded mark)")
	cmd.Flags().StringVar(&output, "output", icons.DefaultOutputDir, "icon tree root")
	cmd.Flags().BoolVar(&refresh, "refresh-icons", false, "regenerate even if outputs exist")
	return cmd
}

// defaultConfigPath returns cwd/eletrocromo.json when it is a regular file.
func defaultConfigPath(cwd string) string {
	try := filepath.Join(cwd, apk.ConfigFileName)
	if st, err := os.Stat(try); err == nil && !st.IsDir() {
		return try
	}
	return ""
}

// absPath joins base when p is non-empty and relative; empty stays empty.
func absPath(base, p string) string {
	p = strings.TrimSpace(p)
	if p == "" {
		return ""
	}
	if filepath.IsAbs(p) {
		return p
	}
	return filepath.Join(base, p)
}

// resolveIconOutput returns an absolute icon tree root (default dist/icons under cwd).
func resolveIconOutput(cwd, outputFlag string) string {
	out := strings.TrimSpace(outputFlag)
	if out == "" {
		out = icons.DefaultOutputDir
	}
	if !filepath.IsAbs(out) {
		return filepath.Join(cwd, out)
	}
	return out
}

// resolveIconSource returns absolute master path (empty = default mark).
// Flag (cwd-relative) wins over config icon (baseDir-relative).
func resolveIconSource(cwd, iconFlag, baseDir, cfgIcon string) string {
	if p := absPath(cwd, iconFlag); p != "" {
		return p
	}
	return absPath(baseDir, cfgIcon)
}

// resolveIconIO returns absolute master path (empty = default mark) and output dir.
func resolveIconIO(cwd, configPath, iconFlag, outputFlag string) (source, output string, err error) {
	output = resolveIconOutput(cwd, outputFlag)

	if p := absPath(cwd, iconFlag); p != "" {
		return p, output, nil
	}

	cfgPath := strings.TrimSpace(configPath)
	if cfgPath == "" {
		cfgPath = defaultConfigPath(cwd)
	}
	if cfgPath != "" {
		cfg, baseDir, err := apk.LoadConfig(cfgPath)
		if err != nil {
			return "", "", err
		}
		return resolveIconSource(cwd, "", baseDir, cfg.Icon), output, nil
	}
	return "", output, nil
}

// ensureBuildIcons returns a complete icon tree, generating when missing or refresh is set.
func ensureBuildIcons(outw io.Writer, iconSrc, iconOut string, refresh bool) (string, error) {
	if !refresh && icons.Complete(iconOut) {
		_, err := fmt.Fprintf(outw, "eletrocromo: icons already present at %s\n", iconOut)
		return iconOut, err
	}
	man, err := icons.Generate(icons.Options{
		SourcePath: iconSrc,
		OutputDir:  iconOut,
		Force:      refresh || !icons.Complete(iconOut),
	})
	if err != nil {
		return "", err
	}
	_, err = fmt.Fprintf(outw, "eletrocromo: icons → %s\n", man.OutputDir)
	return man.OutputDir, err
}
