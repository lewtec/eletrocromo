package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/lewtec/eletrocromo/internal/gen/apk"
	"github.com/lewtec/eletrocromo/internal/icons"
	"github.com/lewtec/lewkit/x/cmd"
	"github.com/lewtec/lewkit/x/taskgroup"
)

type iconsCmd struct {
	config  cmd.StringArg `long:"config" default:"" help:"path to eletrocromo.json (default: ./eletrocromo.json if present)"`
	icon    cmd.StringArg `long:"icon" default:"" help:"master PNG/JPEG (overrides config icon; default: embedded mark)"`
	output  cmd.StringArg `long:"output" default:"dist/icons" help:"icon tree root"`
	refresh cmd.Flag      `long:"refresh-icons" help:"regenerate even if outputs exist"`
}

func (iconsCmd) Description() string {
	return "Generate multi-platform icons from one master PNG/JPEG into dist/icons (source, windows, macos, linux, android, web, manifest.json). Skip when the tree is already complete unless --refresh-icons. SVG is not rasterized in-process yet."
}

func (c *iconsCmd) Run(context.Context) error {
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	src, out, err := resolveIconIO(cwd, c.config.Value(), c.icon.Value(), c.output.Value())
	if err != nil {
		return err
	}
	man, err := icons.Generate(icons.Options{
		SourcePath: src,
		OutputDir:  out,
		Force:      c.refresh.Value(),
	})
	if err != nil {
		return err
	}
	if !c.refresh.Value() && icons.Complete(out) && man != nil {
		_, err = fmt.Fprintf(os.Stdout, "icons up to date: %s\n", man.OutputDir)
	} else {
		_, err = fmt.Fprintf(os.Stdout, "ok %s (%d files)\n", man.OutputDir, len(man.Files))
	}
	return err
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

type iconThen struct {
	src, out string
	refresh  bool
	name     string
}

// runIconsThen generates the icon tree, then runs work with that root.
// Same schedule as build android / ios / macos.
func runIconsThen(ctx context.Context, ic iconThen, work func(iconRoot string) error) error {
	return taskgroup.WithSession(ctx, func(ctx context.Context) error {
		var iconRoot string
		iconsID := taskgroup.Go(ctx, "icons", taskgroup.CPU, func(context.Context, *taskgroup.Status) error {
			var err error
			iconRoot, err = ensureBuildIcons(os.Stdout, ic.src, ic.out, ic.refresh)
			return err
		})
		taskgroup.Go(ctx, ic.name, taskgroup.IO, func(context.Context, *taskgroup.Status) error {
			return work(iconRoot)
		}, iconsID)
		return nil
	})
}
