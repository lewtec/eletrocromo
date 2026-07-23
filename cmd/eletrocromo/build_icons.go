package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/lewtec/eletrocromo/internal/apkgen"
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
				fmt.Fprintf(cmd.OutOrStdout(), "icons up to date: %s\n", man.OutputDir)
			} else {
				fmt.Fprintf(cmd.OutOrStdout(), "ok %s (%d files)\n", man.OutputDir, len(man.Files))
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&configPath, "config", "", "path to eletrocromo.json (default: ./eletrocromo.json if present)")
	cmd.Flags().StringVar(&iconPath, "icon", "", "master PNG/JPEG (overrides config icon; default: embedded mark)")
	cmd.Flags().StringVar(&output, "output", icons.DefaultOutputDir, "icon tree root")
	cmd.Flags().BoolVar(&refresh, "refresh-icons", false, "regenerate even if outputs exist")
	return cmd
}

// resolveIconIO returns absolute master path (empty = default mark) and output dir.
func resolveIconIO(cwd, configPath, iconFlag, outputFlag string) (source, output string, err error) {
	output = strings.TrimSpace(outputFlag)
	if output == "" {
		output = icons.DefaultOutputDir
	}
	if !filepath.IsAbs(output) {
		output = filepath.Join(cwd, output)
	}

	iconFlag = strings.TrimSpace(iconFlag)
	if iconFlag != "" {
		if !filepath.IsAbs(iconFlag) {
			iconFlag = filepath.Join(cwd, iconFlag)
		}
		return iconFlag, output, nil
	}

	baseDir := cwd
	cfgPath := strings.TrimSpace(configPath)
	if cfgPath == "" {
		try := filepath.Join(cwd, apkgen.ConfigFileName)
		if st, err := os.Stat(try); err == nil && !st.IsDir() {
			cfgPath = try
		}
	}
	if cfgPath != "" {
		cfg, dir, err := apkgen.LoadConfig(cfgPath)
		if err != nil {
			return "", "", err
		}
		baseDir = dir
		if p := strings.TrimSpace(cfg.Icon); p != "" {
			if !filepath.IsAbs(p) {
				p = filepath.Join(baseDir, p)
			}
			return p, output, nil
		}
	}
	return "", output, nil
}
